package semstreams

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/c360studio/semlink/internal/projector"
	"github.com/c360studio/semstreams/component"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/payloadregistry"
	graphingest "github.com/c360studio/semstreams/processor/graph-ingest"
	natsserver "github.com/nats-io/nats-server/v2/server"
	"github.com/nats-io/nats.go/jetstream"
)

type RuntimeOptions struct {
	NATSURL  string
	Embedded bool
	Logger   *slog.Logger
}

type Runtime struct {
	NATSURL string
	Client  *natsclient.Client
	Graph   *GraphClient

	ingest       component.LifecycleComponent
	natsServer   *natsserver.Server
	natsStoreDir string
}

func StartRuntime(ctx context.Context, opts RuntimeOptions) (*Runtime, error) {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	var client *natsclient.Client
	var embeddedServer *natsserver.Server
	var embeddedStoreDir string
	var natsURL string
	if opts.Embedded {
		server, storeDir, err := startEmbeddedNATS()
		if err != nil {
			return nil, fmt.Errorf("start embedded nats: %w", err)
		}
		embeddedServer = server
		embeddedStoreDir = storeDir
		natsURL = server.ClientURL()

		client, err = natsclient.NewClient(natsURL,
			natsclient.WithName("semlink-demo"),
			natsclient.WithLogger(logger),
			natsclient.WithTimeout(5*time.Second),
		)
		if err != nil {
			cleanupEmbeddedNATS(embeddedServer, embeddedStoreDir)
			return nil, fmt.Errorf("create embedded nats client: %w", err)
		}
		if err := client.Connect(ctx); err != nil {
			cleanupEmbeddedNATS(embeddedServer, embeddedStoreDir)
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

	if err := ensureGraphBuckets(ctx, client); err != nil {
		if embeddedServer != nil {
			cleanupEmbeddedNATS(embeddedServer, embeddedStoreDir)
		}
		return nil, err
	}

	if err := ensureStreams(ctx, client); err != nil {
		if embeddedServer != nil {
			cleanupEmbeddedNATS(embeddedServer, embeddedStoreDir)
		}
		return nil, err
	}

	reg := payloadregistry.New()
	if err := projector.RegisterPayloads(reg); err != nil {
		if embeddedServer != nil {
			cleanupEmbeddedNATS(embeddedServer, embeddedStoreDir)
		}
		return nil, fmt.Errorf("register semlink payloads: %w", err)
	}

	ingest, err := startGraphIngest(ctx, client, reg, logger)
	if err != nil {
		if embeddedServer != nil {
			cleanupEmbeddedNATS(embeddedServer, embeddedStoreDir)
		}
		return nil, err
	}

	return &Runtime{
		NATSURL:      natsURL,
		Client:       client,
		Graph:        NewGraphClient(client, 5*time.Second),
		ingest:       ingest,
		natsServer:   embeddedServer,
		natsStoreDir: embeddedStoreDir,
	}, nil
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
		cleanupEmbeddedNATS(r.natsServer, r.natsStoreDir)
	}
	return nil
}

func startEmbeddedNATS() (*natsserver.Server, string, error) {
	storeDir, err := os.MkdirTemp("", "semlink-nats-*")
	if err != nil {
		return nil, "", err
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
		_ = os.RemoveAll(storeDir)
		return nil, "", err
	}
	go ns.Start()
	if !ns.ReadyForConnections(10 * time.Second) {
		err := fmt.Errorf("embedded nats did not become ready (running=%t url=%q addr=%v)", ns.Running(), ns.ClientURL(), ns.Addr())
		cleanupEmbeddedNATS(ns, storeDir)
		return nil, "", err
	}
	return ns, storeDir, nil
}

func cleanupEmbeddedNATS(ns *natsserver.Server, storeDir string) {
	if ns != nil {
		ns.Shutdown()
		ns.WaitForShutdown()
	}
	if storeDir != "" {
		_ = os.RemoveAll(storeDir)
	}
}

func ensureGraphBuckets(ctx context.Context, client *natsclient.Client) error {
	for _, cfg := range []jetstream.KeyValueConfig{
		{
			Bucket:      graph.BucketEntityStates,
			Description: "Entity state storage for graph-ingest",
			Storage:     jetstream.FileStorage,
		},
		{
			Bucket:      "ENTITY_SUFFIX_INDEX",
			Description: "Suffix-to-full-ID reverse index for partial entity ID resolution",
			Storage:     jetstream.FileStorage,
		},
		{
			Bucket:      graph.BucketComponentStatus,
			Description: "Component lifecycle status tracking",
			Storage:     jetstream.FileStorage,
		},
	} {
		if _, err := client.CreateKeyValueBucket(ctx, cfg); err != nil {
			return fmt.Errorf("ensure SemStreams KV bucket %s: %w", cfg.Bucket, err)
		}
	}
	return nil
}

func ensureStreams(ctx context.Context, client *natsclient.Client) error {
	if _, err := client.EnsureStream(ctx, jetstream.StreamConfig{
		Name:     "ENTITY",
		Subjects: []string{"entity.>"},
		Storage:  jetstream.MemoryStorage,
	}); err != nil {
		return fmt.Errorf("ensure SemStreams ENTITY stream: %w", err)
	}
	if _, err := client.EnsureStream(ctx, jetstream.StreamConfig{
		Name:              "MAVLINK_RAW",
		Subjects:          []string{"mavlink.raw.>"},
		Storage:           jetstream.MemoryStorage,
		MaxAge:            5 * time.Minute,
		MaxMsgs:           250_000,
		MaxMsgsPerSubject: 5_000,
	}); err != nil {
		return fmt.Errorf("ensure raw MAVLink stream: %w", err)
	}
	return nil
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
