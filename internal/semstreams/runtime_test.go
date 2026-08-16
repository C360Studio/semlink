package semstreams

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/projector"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/c360studio/semstreams/pkg/projection"
	"github.com/nats-io/nats.go/jetstream"
)

func TestProjectionContractsValidateAsOneCompleteSet(t *testing.T) {
	contracts := ProjectionContracts()
	if got, want := len(contracts), 7; got != want {
		t.Fatalf("contract count = %d, want %d", got, want)
	}
	if err := projection.ValidateContracts(contracts); err != nil {
		t.Fatalf("ValidateContracts: %v", err)
	}
	seen := make(map[string]struct{}, len(contracts))
	for _, contract := range contracts {
		if _, duplicate := seen[contract.Name]; duplicate {
			t.Fatalf("duplicate contract %q", contract.Name)
		}
		seen[contract.Name] = struct{}{}
		for _, group := range contract.Groups {
			if group.Name == "" || group.Mode != projection.ModeReconcile {
				t.Fatalf("contract %q group = %#v", contract.Name, group)
			}
		}
	}
}

func TestRuntimeOptionsAcceptNoStateGenerationAttestation(t *testing.T) {
	if _, ok := reflect.TypeOf(RuntimeOptions{}).FieldByName("StateGeneration"); ok {
		t.Fatal("RuntimeOptions still accepts caller-supplied StateGeneration")
	}
}

func TestEmbeddedBeta160MutationAndFrameworkBucketIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	runtime := startTestRuntime(t, ctx)

	if !runtime.FreshState || runtime.ReusedState || runtime.StateSchemaVersion != StateSchemaVersionBeta160 {
		t.Fatalf("state posture fresh=%t reused=%t version=%q", runtime.FreshState, runtime.ReusedState, runtime.StateSchemaVersion)
	}
	buckets, err := runtime.Client.ListKeyValueBuckets(ctx)
	if err != nil {
		t.Fatalf("ListKeyValueBuckets: %v", err)
	}
	if !containsBucket(buckets, graph.BucketEntitySuffixIndex) {
		t.Fatalf("framework suffix bucket missing: %#v", buckets)
	}
	if containsBucket(buckets, "COMPONENT_STATUS") {
		t.Fatalf("removed component status bucket was provisioned: %#v", buckets)
	}

	value := vehicleProjection()
	value.Entity.UpdatedAt = time.Unix(10, 0).UTC()
	created, err := runtime.Graph.UpsertProjection(ctx, value)
	if err != nil {
		t.Fatalf("first UpsertProjection: %v", err)
	}
	if !created.Created || created.Revision == 0 {
		t.Fatalf("first write = %#v", created)
	}

	exact, err := runtime.Graph.QueryExactEntity(ctx, value.Entity.ID)
	if err != nil {
		t.Fatalf("first QueryExactEntity: %v", err)
	}
	if exact.KVRevision != created.Revision || len(exact.Entity.Triples) != len(value.Triples)+1 {
		t.Fatalf("first exact revision=%d want=%d triples=%d want=%d values=%#v", exact.KVRevision, created.Revision, len(exact.Entity.Triples), len(value.Triples)+1, exact.Entity.Triples)
	}

	withoutMode := value
	withoutMode.Triples = removePredicate(value.Triples, projector.PredicateFlightMode)
	updated, err := runtime.Graph.UpsertProjection(ctx, withoutMode)
	if err != nil {
		t.Fatalf("second UpsertProjection: %v", err)
	}
	if updated.Created || updated.Revision <= created.Revision {
		t.Fatalf("second write = %#v", updated)
	}
	exact, err = runtime.Graph.QueryExactEntity(ctx, value.Entity.ID)
	if err != nil {
		t.Fatalf("second QueryExactEntity: %v", err)
	}
	if exact.Entity.GetTriple(projector.PredicateFlightMode) != nil || len(exact.Entity.Triples) != len(withoutMode.Triples)+1 {
		t.Fatalf("optional predicate was not reconciled away: %#v", exact.Entity.Triples)
	}

	stopTestRuntime(t, runtime)
	runtime = nil
	restarted := startTestRuntime(t, ctx)
	defer stopTestRuntime(t, restarted)
	if _, err := restarted.Graph.QueryExactEntity(ctx, value.Entity.ID); err == nil {
		t.Fatal("fresh embedded restart unexpectedly reused prior state")
	}
	recreated, err := restarted.Graph.UpsertProjection(ctx, value)
	if err != nil || !recreated.Created {
		t.Fatalf("recreate after fresh restart result=%#v err=%v", recreated, err)
	}
}

func TestEmbeddedStateDirPersistsAcrossGracefulAndAbruptRestarts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stateDir := filepath.Join(t.TempDir(), "nats-beta160")

	runtime, err := StartRuntime(ctx, RuntimeOptions{Embedded: true, StateDir: stateDir})
	if err != nil {
		t.Fatalf("first StartRuntime: %v", err)
	}
	value := vehicleProjection()
	written, err := runtime.Graph.UpsertProjection(ctx, value)
	if err != nil {
		t.Fatalf("write projection: %v", err)
	}
	want, err := runtime.Graph.QueryExactEntity(ctx, value.Entity.ID)
	if err != nil {
		t.Fatalf("read projection: %v", err)
	}
	if want.KVRevision != written.Revision {
		t.Fatalf("revision=%d want=%d", want.KVRevision, written.Revision)
	}
	stopTestRuntime(t, runtime)
	if _, err := os.Stat(stateDir); err != nil {
		t.Fatalf("caller-owned state directory removed: %v", err)
	}

	runtime, err = StartRuntime(ctx, RuntimeOptions{Embedded: true, StateDir: stateDir})
	if err != nil {
		t.Fatalf("graceful restart: %v", err)
	}
	if runtime.FreshState || !runtime.ReusedState {
		t.Fatalf("graceful restart posture fresh=%t reused=%t", runtime.FreshState, runtime.ReusedState)
	}
	assertExactEntityUnchanged(t, ctx, runtime, value.Entity.ID, want)

	// Simulate an unexpected embedded NATS process termination: do not call
	// Runtime.Stop (the only lifecycle path allowed to clean owned temp state).
	runtime.natsServer.Shutdown()
	runtime.natsServer.WaitForShutdown()
	_ = runtime.Client.Close(ctx)
	runtime.natsServer = nil
	runtime, err = StartRuntime(ctx, RuntimeOptions{Embedded: true, StateDir: stateDir})
	if err != nil {
		t.Fatalf("abrupt restart: %v", err)
	}
	defer stopTestRuntime(t, runtime)
	if runtime.FreshState || !runtime.ReusedState {
		t.Fatalf("abrupt restart posture fresh=%t reused=%t", runtime.FreshState, runtime.ReusedState)
	}
	assertExactEntityUnchanged(t, ctx, runtime, value.Entity.ID, want)
}

func TestEmbeddedOmittedStateDirUsesOwnedTemporaryStore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	runtime := startTestRuntime(t, ctx)
	storeDir := runtime.natsStoreDir
	if storeDir == "" || !runtime.natsStoreOwned {
		t.Fatalf("temporary state posture dir=%q owned=%t", storeDir, runtime.natsStoreOwned)
	}
	stopTestRuntime(t, runtime)
	if _, err := os.Stat(storeDir); !os.IsNotExist(err) {
		t.Fatalf("owned temporary state directory retained: %v", err)
	}
}

func assertExactEntityUnchanged(t *testing.T, ctx context.Context, runtime *Runtime, id string, want *graph.ExactEntity) {
	t.Helper()
	got, err := runtime.Graph.QueryExactEntity(ctx, id)
	if err != nil {
		t.Fatalf("QueryExactEntity after restart: %v", err)
	}
	if got.KVRevision != want.KVRevision || !reflect.DeepEqual(got.Entity.Triples, want.Entity.Triples) {
		t.Fatalf("entity changed across restart: got=%#v want=%#v", got, want)
	}
}

func TestExternalRuntimeSchemaBootstrapRejectsUnversionedAndReusesCurrentState(t *testing.T) {
	t.Run("dirty namespace remains unchanged", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		server, storeDir, owned, err := startEmbeddedNATS("")
		if err != nil {
			t.Fatalf("start test NATS: %v", err)
		}
		defer cleanupEmbeddedNATS(server, storeDir, owned)
		seed := connectTestClient(t, ctx, server.ClientURL())
		defer seed.Close(ctx)
		if _, err := seed.EnsureStream(ctx, entityStreamConfig()); err != nil {
			t.Fatalf("seed ENTITY stream: %v", err)
		}
		bucket, err := seed.CreateKeyValueBucket(ctx, jetstream.KeyValueConfig{Bucket: graph.BucketEntityStates})
		if err != nil {
			t.Fatalf("seed entity bucket: %v", err)
		}
		revision, err := bucket.Put(ctx, "sentinel", []byte("do-not-touch"))
		if err != nil {
			t.Fatalf("seed bucket value: %v", err)
		}

		_, err = StartRuntime(ctx, RuntimeOptions{NATSURL: server.ClientURL()})
		if err == nil || !strings.Contains(err.Error(), "stream:ENTITY") || !strings.Contains(err.Error(), "kv:"+graph.BucketEntityStates) {
			t.Fatalf("StartRuntime dirty error = %v", err)
		}
		entry, err := bucket.Get(ctx, "sentinel")
		if err != nil || entry.Revision() != revision || string(entry.Value()) != "do-not-touch" {
			t.Fatalf("dirty state changed entry=%#v err=%v", entry, err)
		}
		buckets, err := seed.ListKeyValueBuckets(ctx)
		if err != nil || len(buckets) != 1 || buckets[0] != graph.BucketEntityStates {
			t.Fatalf("preflight provisioned buckets: %#v err=%v", buckets, err)
		}
	})

	t.Run("clean namespace starts once with unrelated resources", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		server, storeDir, owned, err := startEmbeddedNATS("")
		if err != nil {
			t.Fatalf("start test NATS: %v", err)
		}
		defer cleanupEmbeddedNATS(server, storeDir, owned)
		seed := connectTestClient(t, ctx, server.ClientURL())
		if _, err := seed.EnsureStream(ctx, jetstream.StreamConfig{
			Name: "UNRELATED", Subjects: []string{"unrelated.>"}, Storage: jetstream.MemoryStorage,
			MaxAge: time.Hour, MaxBytes: 1024 * 1024, Discard: jetstream.DiscardOld,
		}); err != nil {
			t.Fatalf("seed unrelated stream: %v", err)
		}
		if _, err := seed.CreateKeyValueBucket(ctx, jetstream.KeyValueConfig{Bucket: "UNRELATED_BUCKET"}); err != nil {
			t.Fatalf("seed unrelated bucket: %v", err)
		}
		_ = seed.Close(ctx)

		runtime, err := StartRuntime(ctx, RuntimeOptions{NATSURL: server.ClientURL()})
		if err != nil {
			t.Fatalf("first StartRuntime: %v", err)
		}
		if !runtime.FreshState || runtime.ReusedState || runtime.StateSchemaVersion != StateSchemaVersionBeta160 {
			t.Fatalf("runtime posture fresh=%t reused=%t version=%q", runtime.FreshState, runtime.ReusedState, runtime.StateSchemaVersion)
		}
		if _, err := runtime.Client.GetKeyValueBucket(ctx, graph.BucketEntitySuffixIndex); err != nil {
			t.Fatalf("beta.160 suffix bucket missing: %v", err)
		}
		stopTestRuntime(t, runtime)
		restarted, err := StartRuntime(ctx, RuntimeOptions{NATSURL: server.ClientURL()})
		if err != nil {
			t.Fatalf("second StartRuntime: %v", err)
		}
		defer stopTestRuntime(t, restarted)
		if restarted.FreshState || !restarted.ReusedState || restarted.StateSchemaVersion != StateSchemaVersionBeta160 {
			t.Fatalf("restart posture fresh=%t reused=%t version=%q", restarted.FreshState, restarted.ReusedState, restarted.StateSchemaVersion)
		}
	})
}

func TestExternalRuntimeRejectsDivergentOwnedConfigurationReadOnly(t *testing.T) {
	t.Run("metadata bucket before stamping", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		server, storeDir, owned, err := startEmbeddedNATS("")
		if err != nil {
			t.Fatalf("start NATS: %v", err)
		}
		defer cleanupEmbeddedNATS(server, storeDir, owned)
		seed := connectTestClient(t, ctx, server.ClientURL())
		defer seed.Close(ctx)
		bucket, err := seed.CreateKeyValueBucket(ctx, jetstream.KeyValueConfig{
			Bucket: runtimeMetadataBucket, Storage: jetstream.MemoryStorage, History: 2, TTL: time.Hour,
		})
		if err != nil {
			t.Fatalf("seed divergent metadata: %v", err)
		}
		before, err := bucket.Status(ctx)
		if err != nil {
			t.Fatalf("metadata status before: %v", err)
		}
		_, err = StartRuntime(ctx, RuntimeOptions{NATSURL: server.ClientURL()})
		if err == nil || !strings.Contains(err.Error(), "metadata") || !strings.Contains(err.Error(), "clear/reset") {
			t.Fatalf("divergent metadata StartRuntime error = %v", err)
		}
		if _, err := bucket.Get(ctx, runtimeSchemaVersionKey); !errors.Is(err, jetstream.ErrKeyNotFound) {
			t.Fatalf("rejected metadata bucket was stamped: %v", err)
		}
		after, err := bucket.Status(ctx)
		if err != nil || after.History() != before.History() || after.TTL() != before.TTL() {
			t.Fatalf("metadata config changed before=%#v after=%#v err=%v", before, after, err)
		}
	})

	for _, expected := range []jetstream.StreamConfig{entityStreamConfig(), rawMAVLinkStreamConfig()} {
		t.Run(expected.Name+" stream after exact stamp", func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()
			server, storeDir, owned, err := startEmbeddedNATS("")
			if err != nil {
				t.Fatalf("start NATS: %v", err)
			}
			defer cleanupEmbeddedNATS(server, storeDir, owned)
			seed := connectTestClient(t, ctx, server.ClientURL())
			namespace := natsSchemaNamespace{client: seed}
			store, err := namespace.EnsureSchemaStore(ctx)
			if err != nil || store.CreateSchemaStamp(ctx, StateSchemaVersionBeta160) != nil {
				t.Fatalf("seed exact stamp: %v", err)
			}
			divergent := expected
			divergent.Storage = jetstream.MemoryStorage
			if _, err := seed.EnsureStream(ctx, divergent); err != nil {
				t.Fatalf("seed divergent %s: %v", expected.Name, err)
			}
			before, err := namespace.ReadStreamConfig(ctx, expected.Name)
			if err != nil {
				t.Fatalf("read divergent stream: %v", err)
			}
			_ = seed.Close(ctx)
			_, err = StartRuntime(ctx, RuntimeOptions{NATSURL: server.ClientURL()})
			if err == nil || !strings.Contains(err.Error(), expected.Name+".storage") || !strings.Contains(err.Error(), "clear/reset") {
				t.Fatalf("divergent stream StartRuntime error = %v", err)
			}
			verify := connectTestClient(t, ctx, server.ClientURL())
			defer verify.Close(ctx)
			after, err := (natsSchemaNamespace{client: verify}).ReadStreamConfig(ctx, expected.Name)
			if err != nil || after.Storage != before.Storage || !reflect.DeepEqual(after.Subjects, before.Subjects) {
				t.Fatalf("stream config changed before=%#v after=%#v err=%v", before, after, err)
			}
		})
	}
}

func TestExternalRuntimePreservesStateAcrossNATSRestart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stateDir := filepath.Join(t.TempDir(), "external-nats-beta160")
	server, _, _, err := startEmbeddedNATS(stateDir)
	if err != nil {
		t.Fatalf("start external NATS: %v", err)
	}
	runtime, err := StartRuntime(ctx, RuntimeOptions{NATSURL: server.ClientURL()})
	if err != nil {
		t.Fatalf("first runtime: %v", err)
	}
	value := vehicleProjection()
	written, err := runtime.Graph.UpsertProjection(ctx, value)
	if err != nil {
		t.Fatalf("write projection: %v", err)
	}
	want, err := runtime.Graph.QueryExactEntity(ctx, value.Entity.ID)
	if err != nil || want.KVRevision != written.Revision {
		t.Fatalf("read before restart=%#v err=%v", want, err)
	}
	stopTestRuntime(t, runtime)
	server.Shutdown()
	server.WaitForShutdown()

	server, _, _, err = startEmbeddedNATS(stateDir)
	if err != nil {
		t.Fatalf("restart external NATS: %v", err)
	}
	defer cleanupEmbeddedNATS(server, stateDir, false)
	runtime, err = StartRuntime(ctx, RuntimeOptions{NATSURL: server.ClientURL()})
	if err != nil {
		t.Fatalf("runtime after external NATS restart: %v", err)
	}
	defer stopTestRuntime(t, runtime)
	if runtime.FreshState || !runtime.ReusedState {
		t.Fatalf("restart posture fresh=%t reused=%t", runtime.FreshState, runtime.ReusedState)
	}
	assertExactEntityUnchanged(t, ctx, runtime, value.Entity.ID, want)
}

func TestExternalRuntimePreservesStateAfterUnexpectedNATSTermination(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	stateDir := filepath.Join(t.TempDir(), "abrupt-external-nats-beta160")
	server, _, _, err := startEmbeddedNATS(stateDir)
	if err != nil {
		t.Fatalf("start external NATS: %v", err)
	}
	runtime, err := StartRuntime(ctx, RuntimeOptions{NATSURL: server.ClientURL()})
	if err != nil {
		t.Fatalf("first runtime: %v", err)
	}
	value := vehicleProjection()
	if _, err := runtime.Graph.UpsertProjection(ctx, value); err != nil {
		t.Fatalf("write projection: %v", err)
	}
	want, err := runtime.Graph.QueryExactEntity(ctx, value.Entity.ID)
	if err != nil {
		t.Fatalf("read before abrupt termination: %v", err)
	}

	// Terminate the external NATS server while the runtime is still active.
	// This deliberately bypasses Runtime.Stop and proves recovery from the
	// durable store rather than from a coordinated application shutdown.
	server.Shutdown()
	server.WaitForShutdown()
	_ = runtime.ingest.Stop(3 * time.Second)
	_ = runtime.Client.Close(ctx)

	server, _, _, err = startEmbeddedNATS(stateDir)
	if err != nil {
		t.Fatalf("restart external NATS after unexpected termination: %v", err)
	}
	defer cleanupEmbeddedNATS(server, stateDir, false)
	restarted, err := StartRuntime(ctx, RuntimeOptions{NATSURL: server.ClientURL()})
	if err != nil {
		t.Fatalf("runtime after unexpected NATS termination: %v", err)
	}
	defer stopTestRuntime(t, restarted)
	if restarted.FreshState || !restarted.ReusedState {
		t.Fatalf("restart posture fresh=%t reused=%t", restarted.FreshState, restarted.ReusedState)
	}
	assertExactEntityUnchanged(t, ctx, restarted, value.Entity.ID, want)
}

func TestExternalRuntimeRecoversStampedPartialBootstrap(t *testing.T) {
	graphBuckets := []string{
		graph.BucketEntityStates,
		graph.BucketEntitySuffixIndex,
		graph.BucketGraphIngestAppliedSeq,
		graph.BucketGraphStatus,
	}
	phases := []struct {
		name        string
		streamCount int
		bucketCount int
	}{
		{name: "stamp-only"},
		{name: "entity-stream", streamCount: 1},
		{name: "both-streams", streamCount: 2},
		{name: "entity-states", streamCount: 2, bucketCount: 1},
		{name: "entity-suffix-index", streamCount: 2, bucketCount: 2},
		{name: "applied-sequence", streamCount: 2, bucketCount: 3},
		{name: "graph-status", streamCount: 2, bucketCount: 4},
	}
	for _, phase := range phases {
		t.Run(phase.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			server, storeDir, owned, err := startEmbeddedNATS("")
			if err != nil {
				t.Fatalf("start NATS: %v", err)
			}
			defer cleanupEmbeddedNATS(server, storeDir, owned)
			seed := connectTestClient(t, ctx, server.ClientURL())
			namespace := natsSchemaNamespace{client: seed}
			store, err := namespace.EnsureSchemaStore(ctx)
			if err != nil || store.CreateSchemaStamp(ctx, StateSchemaVersionBeta160) != nil {
				t.Fatalf("seed schema stamp: %v", err)
			}
			if phase.streamCount >= 1 {
				if _, err := seed.EnsureStream(ctx, entityStreamConfig()); err != nil {
					t.Fatalf("seed ENTITY: %v", err)
				}
			}
			if phase.streamCount >= 2 {
				if _, err := seed.EnsureStream(ctx, rawMAVLinkStreamConfig()); err != nil {
					t.Fatalf("seed MAVLINK_RAW: %v", err)
				}
			}
			for _, bucket := range graphBuckets[:phase.bucketCount] {
				if _, err := graph.EnsureCatalogBucket(ctx, seed, bucket); err != nil {
					t.Fatalf("seed %s: %v", bucket, err)
				}
			}
			_ = seed.Close(ctx)
			runtime, err := StartRuntime(ctx, RuntimeOptions{NATSURL: server.ClientURL()})
			if err != nil {
				t.Fatalf("recover phase %s: %v", phase.name, err)
			}
			defer stopTestRuntime(t, runtime)
			if runtime.FreshState || !runtime.ReusedState {
				t.Fatalf("partial recovery posture fresh=%t reused=%t", runtime.FreshState, runtime.ReusedState)
			}
		})
	}
}

func TestConcurrentExternalInitializersConvergeOnOneCASWinner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	server, storeDir, owned, err := startEmbeddedNATS("")
	if err != nil {
		t.Fatalf("start NATS: %v", err)
	}
	defer cleanupEmbeddedNATS(server, storeDir, owned)
	start := make(chan struct{})
	results := make(chan *Runtime, 2)
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			runtime, runErr := StartRuntime(ctx, RuntimeOptions{NATSURL: server.ClientURL()})
			if runErr != nil {
				errs <- runErr
				return
			}
			results <- runtime
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent initializer: %v", err)
	}
	fresh := 0
	reused := 0
	for runtime := range results {
		defer stopTestRuntime(t, runtime)
		if runtime.FreshState {
			fresh++
		}
		if runtime.ReusedState {
			reused++
		}
	}
	if fresh != 1 || reused != 1 {
		t.Fatalf("CAS postures fresh=%d reused=%d", fresh, reused)
	}
}

func connectTestClient(t *testing.T, ctx context.Context, url string) *natsclient.Client {
	t.Helper()
	client, err := natsclient.NewClient(url, natsclient.WithName(fmt.Sprintf("semlink-preflight-test-%d", time.Now().UnixNano())))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	return client
}

func startTestRuntime(t *testing.T, ctx context.Context) *Runtime {
	t.Helper()
	runtime, err := StartRuntime(ctx, RuntimeOptions{Embedded: true})
	if err != nil {
		t.Fatalf("StartRuntime: %v", err)
	}
	return runtime
}

func stopTestRuntime(t *testing.T, runtime *Runtime) {
	t.Helper()
	if runtime == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := runtime.Stop(ctx); err != nil {
		t.Fatalf("Runtime.Stop: %v", err)
	}
}

func containsBucket(buckets []string, want string) bool {
	for _, bucket := range buckets {
		if bucket == want {
			return true
		}
	}
	return false
}

func removePredicate(triples []message.Triple, predicate string) []message.Triple {
	out := make([]message.Triple, 0, len(triples))
	for _, triple := range triples {
		if triple.Predicate != predicate {
			out = append(out, triple)
		}
	}
	return out
}
