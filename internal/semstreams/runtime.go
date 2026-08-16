package semstreams

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"time"

	"github.com/c360studio/semlink/internal/cop"
	"github.com/c360studio/semlink/internal/projector"
	"github.com/c360studio/semlink/internal/rules"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/c360studio/semstreams/pkg/projection"
	graphingest "github.com/c360studio/semstreams/processor/graph-ingest"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go/jetstream"
)

type RuntimeOptions struct {
	NATSURL  string
	Embedded bool
	StateDir string
	Logger   *slog.Logger
}

type Runtime struct {
	NATSURL string
	Client  *natsclient.Client
	Graph   *GraphClient

	StateSchemaVersion string
	FreshState         bool
	ReusedState        bool

	ingest         component.LifecycleComponent
	natsServer     *natsserver.Server
	natsStoreDir   string
	natsStoreOwned bool
}

func StartRuntime(ctx context.Context, opts RuntimeOptions) (*Runtime, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}
	contracts := ProjectionContracts()
	if err := projection.ValidateContracts(contracts); err != nil {
		return nil, fmt.Errorf("validate SemLink projection contracts: %w", err)
	}

	var client *natsclient.Client
	var embeddedServer *natsserver.Server
	var embeddedStoreDir string
	var embeddedStoreOwned bool
	var natsURL string
	if opts.Embedded {
		server, storeDir, storeOwned, err := startEmbeddedNATS(opts.StateDir)
		if err != nil {
			return nil, fmt.Errorf("start embedded nats: %w", err)
		}
		embeddedServer = server
		embeddedStoreDir = storeDir
		embeddedStoreOwned = storeOwned
		natsURL = server.ClientURL()

		client, err = natsclient.NewClient(natsURL,
			natsclient.WithName("semlink-demo"),
			natsclient.WithLogger(logger),
			natsclient.WithTimeout(5*time.Second),
		)
		if err != nil {
			cleanupEmbeddedNATS(embeddedServer, embeddedStoreDir, embeddedStoreOwned)
			return nil, fmt.Errorf("create embedded nats client: %w", err)
		}
		if err := client.Connect(ctx); err != nil {
			cleanupEmbeddedNATS(embeddedServer, embeddedStoreDir, embeddedStoreOwned)
			return nil, fmt.Errorf("connect embedded nats: %w", err)
		}
	} else {
		natsURL = opts.NATSURL
		if natsURL == "" {
			natsURL = "nats://127.0.0.1:4222"
		}
		var err error
		client, err = natsclient.NewClient(natsURL,
			natsclient.WithName("semlink-demo"),
			natsclient.WithLogger(logger),
			natsclient.WithTimeout(5*time.Second),
		)
		if err != nil {
			return nil, fmt.Errorf("create nats client: %w", err)
		}
		if err := client.Connect(ctx); err != nil {
			return nil, fmt.Errorf("connect nats: %w", err)
		}
	}
	cleanupStartFailure := func() {
		if client != nil {
			_ = client.Close(ctx)
		}
		if embeddedServer != nil {
			cleanupEmbeddedNATS(embeddedServer, embeddedStoreDir, embeddedStoreOwned)
		}
	}
	posture, err := bootstrapStateSchema(ctx, natsSchemaNamespace{client: client})
	if err != nil {
		cleanupStartFailure()
		return nil, fmt.Errorf("validate NATS state schema: %w", err)
	}

	if err := ensureStreams(ctx, client); err != nil {
		cleanupStartFailure()
		return nil, err
	}

	reg := payloadregistry.New()
	if err := projector.RegisterPayloads(reg); err != nil {
		cleanupStartFailure()
		return nil, fmt.Errorf("register semlink payloads: %w", err)
	}
	if err := cop.RegisterPayloads(reg); err != nil {
		cleanupStartFailure()
		return nil, fmt.Errorf("register semlink cop payloads: %w", err)
	}
	if err := rules.RegisterPayloads(reg); err != nil {
		cleanupStartFailure()
		return nil, fmt.Errorf("register semlink rule payloads: %w", err)
	}

	graphClient, err := NewGraphClient(client, contracts, 5*time.Second)
	if err != nil {
		cleanupStartFailure()
		return nil, err
	}

	ingest, err := startGraphIngest(ctx, client, reg, logger)
	if err != nil {
		cleanupStartFailure()
		return nil, err
	}
	if err := validateProvisionedSchema(ctx, natsSchemaNamespace{client: client}); err != nil {
		_ = ingest.Stop(3 * time.Second)
		cleanupStartFailure()
		return nil, fmt.Errorf("validate provisioned beta.160 schema: %w", err)
	}

	return &Runtime{
		NATSURL:            natsURL,
		Client:             client,
		Graph:              graphClient,
		StateSchemaVersion: posture.StateSchemaVersion,
		FreshState:         posture.FreshState,
		ReusedState:        posture.ReusedState,
		ingest:             ingest,
		natsServer:         embeddedServer,
		natsStoreDir:       embeddedStoreDir,
		natsStoreOwned:     embeddedStoreOwned,
	}, nil
}

type natsNamespaceInspector struct {
	client *natsclient.Client
}

func (i natsNamespaceInspector) ListStreamNames(ctx context.Context) ([]string, error) {
	js, err := i.client.JetStream()
	if err != nil {
		return nil, err
	}
	lister := js.StreamNames(ctx)
	var names []string
	for name := range lister.Name() {
		if name != "" {
			names = append(names, name)
		}
	}
	if err := lister.Err(); err != nil {
		return nil, err
	}
	return names, nil
}

func managedStreamNames() []string {
	return []string{"ENTITY", "MAVLINK_RAW"}
}

func managedGraphBucketNames() []string {
	catalog := graph.KVCatalog()
	names := make([]string, 0, len(catalog)+1)
	for _, spec := range catalog {
		names = append(names, spec.Name)
	}
	names = append(names, "COMPONENT_STATUS")
	sort.Strings(names)
	return names
}

func stringSet(values []string) map[string]struct{} {
	set := make(map[string]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

// ProjectionContracts returns the complete contract set validated before any
// graph component starts or mutation client is constructed.
func ProjectionContracts() []projection.Contract {
	contracts := make([]projection.Contract, 0, 7)
	contracts = append(contracts, projector.Contracts()...)
	contracts = append(contracts, cop.Contracts()...)
	contracts = append(contracts, rules.Contracts()...)
	return contracts
}

func (r *Runtime) Stop(ctx context.Context) error {
	if r == nil {
		return nil
	}
	if r.ingest != nil {
		_ = r.ingest.Stop(3 * time.Second)
	}
	if r.Client != nil {
		_ = r.Client.Close(ctx)
	}
	if r.natsServer != nil {
		cleanupEmbeddedNATS(r.natsServer, r.natsStoreDir, r.natsStoreOwned)
	}
	return nil
}

func startEmbeddedNATS(stateDir string) (*natsserver.Server, string, bool, error) {
	storeDir := stateDir
	owned := false
	var err error
	if storeDir == "" {
		storeDir, err = os.MkdirTemp("", "semlink-nats-*")
		owned = true
	} else {
		err = os.MkdirAll(storeDir, 0o750)
	}
	if err != nil {
		return nil, "", false, err
	}
	opts := &natsserver.Options{
		ServerName:         "semlink-embedded",
		Host:               "127.0.0.1",
		Port:               -1,
		HTTPPort:           0,
		NoLog:              true,
		NoSigs:             true,
		JetStream:          true,
		StoreDir:           storeDir,
		JetStreamMaxMemory: 256 * 1024 * 1024,
		JetStreamMaxStore:  512 * 1024 * 1024,
	}
	ns, err := natsserver.NewServer(opts)
	if err != nil {
		if owned {
			_ = os.RemoveAll(storeDir)
		}
		return nil, "", false, err
	}
	go ns.Start()
	if !ns.ReadyForConnections(10 * time.Second) {
		err := fmt.Errorf("embedded nats did not become ready (running=%t url=%q addr=%v)", ns.Running(), ns.ClientURL(), ns.Addr())
		cleanupEmbeddedNATS(ns, storeDir, owned)
		return nil, "", false, err
	}
	return ns, storeDir, owned, nil
}

func cleanupEmbeddedNATS(ns *natsserver.Server, storeDir string, owned bool) {
	if ns != nil {
		ns.Shutdown()
		ns.WaitForShutdown()
	}
	if owned && storeDir != "" {
		_ = os.RemoveAll(storeDir)
	}
}

func ensureStreams(ctx context.Context, client *natsclient.Client) error {
	if err := ensureAndValidateStream(ctx, client, entityStreamConfig()); err != nil {
		return fmt.Errorf("ensure SemStreams ENTITY stream: %w", err)
	}
	if err := ensureAndValidateStream(ctx, client, rawMAVLinkStreamConfig()); err != nil {
		return fmt.Errorf("ensure raw MAVLink stream: %w", err)
	}
	return nil
}

func ensureAndValidateStream(ctx context.Context, client *natsclient.Client, expected jetstream.StreamConfig) error {
	stream, err := client.EnsureStream(ctx, expected)
	if err != nil {
		return err
	}
	info, err := stream.Info(ctx)
	if err != nil {
		return fmt.Errorf("read bound stream configuration: %w", err)
	}
	return validateManagedStreamConfig(info.Config, expected)
}

func entityStreamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:     "ENTITY",
		Subjects: []string{"entity.>"},
		Storage:  jetstream.FileStorage,
		MaxAge:   24 * time.Hour,
		MaxBytes: 64 * 1024 * 1024,
		Discard:  jetstream.DiscardOld,
	}
}

func rawMAVLinkStreamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:              "MAVLINK_RAW",
		Subjects:          []string{"mavlink.raw.>"},
		Storage:           jetstream.FileStorage,
		MaxAge:            5 * time.Minute,
		MaxBytes:          128 * 1024 * 1024,
		MaxMsgs:           250_000,
		MaxMsgsPerSubject: 5_000,
		Discard:           jetstream.DiscardOld,
	}
}

func startGraphIngest(ctx context.Context, client *natsclient.Client, reg *payloadregistry.Registry, logger *slog.Logger) (component.LifecycleComponent, error) {
	cfg := graphingest.DefaultConfig()
	raw, err := json.Marshal(cfg)
	if err != nil {
		return nil, err
	}
	discoverable, err := graphingest.CreateGraphIngest(raw, component.Dependencies{
		NATSClient:      client,
		Logger:          logger,
		PayloadRegistry: reg,
	})
	if err != nil {
		return nil, fmt.Errorf("create graph-ingest: %w", err)
	}
	lifecycle, ok := discoverable.(component.LifecycleComponent)
	if !ok {
		return nil, fmt.Errorf("graph-ingest does not implement LifecycleComponent")
	}
	if err := lifecycle.Initialize(); err != nil {
		return nil, fmt.Errorf("initialize graph-ingest: %w", err)
	}
	if err := lifecycle.Start(ctx); err != nil {
		return nil, fmt.Errorf("start graph-ingest: %w", err)
	}
	return lifecycle, nil
}
