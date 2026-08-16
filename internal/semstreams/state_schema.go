package semstreams

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/natsclient"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	StateSchemaVersionBeta160 = "beta.160"
	runtimeMetadataBucket     = "SEMLINK_RUNTIME_META"
	runtimeSchemaVersionKey   = "state-schema-version"
)

var (
	errSchemaStoreNotFound = errors.New("schema metadata store not found")
	errSchemaStampNotFound = errors.New("schema stamp not found")
	errSchemaCASLost       = errors.New("schema stamp compare-and-set lost")
)

type StatePosture struct {
	StateSchemaVersion string
	FreshState         bool
	ReusedState        bool
}

type schemaStampStore interface {
	ReadSchemaStamp(context.Context) (string, error)
	CreateSchemaStamp(context.Context, string) error
}

type schemaNamespace interface {
	ListStreamNames(context.Context) ([]string, error)
	OpenSchemaStore(context.Context) (schemaStampStore, error)
	EnsureSchemaStore(context.Context) (schemaStampStore, error)
	ValidateSchemaStore(context.Context) error
}

type provisionedSchemaNamespace interface {
	schemaNamespace
	ReadStreamConfig(context.Context, string) (jetstream.StreamConfig, error)
}

type natsSchemaNamespace struct {
	client *natsclient.Client
}

func (n natsSchemaNamespace) ListStreamNames(ctx context.Context) ([]string, error) {
	return natsNamespaceInspector{client: n.client}.ListStreamNames(ctx)
}

func (n natsSchemaNamespace) OpenSchemaStore(ctx context.Context) (schemaStampStore, error) {
	bucket, err := n.client.GetKeyValueBucket(ctx, runtimeMetadataBucket)
	if errors.Is(err, jetstream.ErrBucketNotFound) {
		return nil, errSchemaStoreNotFound
	}
	if err != nil {
		return nil, err
	}
	return natsSchemaStampStore{bucket: bucket}, nil
}

func (n natsSchemaNamespace) EnsureSchemaStore(ctx context.Context) (schemaStampStore, error) {
	bucket, err := n.client.CreateKeyValueBucket(ctx, jetstream.KeyValueConfig{
		Bucket:  runtimeMetadataBucket,
		Storage: jetstream.FileStorage,
		History: 1,
	})
	if err != nil {
		return nil, err
	}
	return natsSchemaStampStore{bucket: bucket}, nil
}

func (n natsSchemaNamespace) ValidateSchemaStore(ctx context.Context) error {
	config, err := n.ReadStreamConfig(ctx, natsclient.KVStreamPrefix+runtimeMetadataBucket)
	if err != nil {
		return err
	}
	return validateRuntimeMetadataStreamConfig(config)
}

func (n natsSchemaNamespace) ReadStreamConfig(ctx context.Context, name string) (jetstream.StreamConfig, error) {
	js, err := n.client.JetStream()
	if err != nil {
		return jetstream.StreamConfig{}, err
	}
	stream, err := js.Stream(ctx, name)
	if err != nil {
		return jetstream.StreamConfig{}, err
	}
	info, err := stream.Info(ctx)
	if err != nil {
		return jetstream.StreamConfig{}, err
	}
	return info.Config, nil
}

type natsSchemaStampStore struct {
	bucket jetstream.KeyValue
}

func (s natsSchemaStampStore) ReadSchemaStamp(ctx context.Context) (string, error) {
	entry, err := s.bucket.Get(ctx, runtimeSchemaVersionKey)
	if errors.Is(err, jetstream.ErrKeyNotFound) || errors.Is(err, jetstream.ErrKeyDeleted) {
		return "", errSchemaStampNotFound
	}
	if err != nil {
		return "", err
	}
	return string(entry.Value()), nil
}

func (s natsSchemaStampStore) CreateSchemaStamp(ctx context.Context, version string) error {
	_, err := s.bucket.Create(ctx, runtimeSchemaVersionKey, []byte(version))
	if errors.Is(err, jetstream.ErrKeyExists) {
		return errSchemaCASLost
	}
	return err
}

func bootstrapStateSchema(ctx context.Context, namespace schemaNamespace) (StatePosture, error) {
	streams, err := namespace.ListStreamNames(ctx)
	if err != nil {
		return StatePosture{}, fmt.Errorf("enumerate NATS state before schema validation: %w", err)
	}
	conflicts := managedResourceConflicts(streams)
	metadataPresent := containsString(streams, natsclient.KVStreamPrefix+runtimeMetadataBucket)

	var store schemaStampStore
	if metadataPresent {
		store, err = namespace.OpenSchemaStore(ctx)
		if err != nil {
			return StatePosture{}, fmt.Errorf("open schema metadata bucket: %w", err)
		}
		if err := namespace.ValidateSchemaStore(ctx); err != nil {
			return StatePosture{}, fmt.Errorf("schema metadata configuration rejected: %w", err)
		}
		if version, readErr := store.ReadSchemaStamp(ctx); readErr == nil {
			return postureForObservedSchema(version)
		} else if !errors.Is(readErr, errSchemaStampNotFound) {
			return StatePosture{}, fmt.Errorf("read schema stamp: %w", readErr)
		}
	}

	if len(conflicts) != 0 {
		return StatePosture{}, schemaStateError("missing", conflicts)
	}
	if store == nil {
		store, err = namespace.EnsureSchemaStore(ctx)
		if err != nil {
			return StatePosture{}, fmt.Errorf("create schema metadata bucket: %w", err)
		}
		if err := namespace.ValidateSchemaStore(ctx); err != nil {
			return StatePosture{}, fmt.Errorf("new schema metadata configuration invalid: %w", err)
		}
		if version, readErr := store.ReadSchemaStamp(ctx); readErr == nil {
			return postureForObservedSchema(version)
		} else if !errors.Is(readErr, errSchemaStampNotFound) {
			return StatePosture{}, fmt.Errorf("read schema stamp after metadata initialization: %w", readErr)
		}
	}

	// Re-inventory immediately before CAS so a resource that appeared while the
	// metadata bucket was being established is never silently adopted.
	streams, err = namespace.ListStreamNames(ctx)
	if err != nil {
		return StatePosture{}, fmt.Errorf("re-enumerate NATS state before schema stamp: %w", err)
	}
	if conflicts = managedResourceConflicts(streams); len(conflicts) != 0 {
		if version, readErr := store.ReadSchemaStamp(ctx); readErr == nil {
			return postureForObservedSchema(version)
		}
		return StatePosture{}, schemaStateError("missing", conflicts)
	}

	if err := store.CreateSchemaStamp(ctx, StateSchemaVersionBeta160); err == nil {
		return StatePosture{StateSchemaVersion: StateSchemaVersionBeta160, FreshState: true}, nil
	} else if !errors.Is(err, errSchemaCASLost) {
		return StatePosture{}, fmt.Errorf("CAS-create schema stamp: %w", err)
	}
	version, err := store.ReadSchemaStamp(ctx)
	if err != nil {
		return StatePosture{}, fmt.Errorf("read winning schema stamp after CAS loss: %w", err)
	}
	return postureForObservedSchema(version)
}

func validateProvisionedSchema(ctx context.Context, namespace provisionedSchemaNamespace) error {
	store, err := namespace.OpenSchemaStore(ctx)
	if err != nil {
		return fmt.Errorf("open schema metadata bucket: %w", err)
	}
	version, err := store.ReadSchemaStamp(ctx)
	if err != nil {
		return fmt.Errorf("read schema stamp: %w", err)
	}
	if _, err := postureForObservedSchema(version); err != nil {
		return err
	}
	if err := namespace.ValidateSchemaStore(ctx); err != nil {
		return fmt.Errorf("schema metadata configuration invalid: %w", err)
	}
	streams, err := namespace.ListStreamNames(ctx)
	if err != nil {
		return fmt.Errorf("enumerate provisioned schema: %w", err)
	}
	set := stringSet(streams)
	required := []string{
		"ENTITY",
		"MAVLINK_RAW",
		natsclient.KVStreamPrefix + runtimeMetadataBucket,
		natsclient.KVStreamPrefix + graph.BucketEntityStates,
		natsclient.KVStreamPrefix + graph.BucketEntitySuffixIndex,
		natsclient.KVStreamPrefix + graph.BucketGraphIngestAppliedSeq,
		natsclient.KVStreamPrefix + graph.BucketGraphStatus,
	}
	var missing []string
	for _, name := range required {
		if _, ok := set[name]; !ok {
			missing = append(missing, name)
		}
	}
	if len(missing) != 0 {
		return fmt.Errorf("beta.160 schema incomplete; missing resources=%s", strings.Join(missing, ","))
	}
	for _, expected := range []jetstream.StreamConfig{entityStreamConfig(), rawMAVLinkStreamConfig()} {
		observed, err := namespace.ReadStreamConfig(ctx, expected.Name)
		if err != nil {
			return fmt.Errorf("read %s stream configuration: %w", expected.Name, err)
		}
		if err := validateManagedStreamConfig(observed, expected); err != nil {
			return err
		}
	}
	return nil
}

func runtimeMetadataStreamConfig() jetstream.StreamConfig {
	return jetstream.StreamConfig{
		Name:              natsclient.KVStreamPrefix + runtimeMetadataBucket,
		Subjects:          []string{"$KV." + runtimeMetadataBucket + ".>"},
		Retention:         jetstream.LimitsPolicy,
		MaxMsgsPerSubject: 1,
		MaxAge:            0,
		Storage:           jetstream.FileStorage,
		Discard:           jetstream.DiscardNew,
		AllowRollup:       true,
		DenyDelete:        true,
		AllowDirect:       true,
	}
}

func validateRuntimeMetadataStreamConfig(observed jetstream.StreamConfig) error {
	expected := runtimeMetadataStreamConfig()
	if observed.Name != expected.Name {
		return ownedConfigError("metadata.name", expected.Name, observed.Name)
	}
	if !sameSubjects(observed.Subjects, expected.Subjects) {
		return ownedConfigError("metadata.subjects", strings.Join(expected.Subjects, ","), strings.Join(observed.Subjects, ","))
	}
	checks := []struct {
		field    string
		expected any
		observed any
	}{
		{"metadata.storage", expected.Storage, observed.Storage},
		{"metadata.history", expected.MaxMsgsPerSubject, observed.MaxMsgsPerSubject},
		{"metadata.ttl", expected.MaxAge, observed.MaxAge},
		{"metadata.retention", expected.Retention, observed.Retention},
		{"metadata.discard", expected.Discard, observed.Discard},
		{"metadata.allow-rollup", expected.AllowRollup, observed.AllowRollup},
		{"metadata.deny-delete", expected.DenyDelete, observed.DenyDelete},
		{"metadata.allow-direct", expected.AllowDirect, observed.AllowDirect},
	}
	for _, check := range checks {
		if fmt.Sprint(check.expected) != fmt.Sprint(check.observed) {
			return ownedConfigError(check.field, check.expected, check.observed)
		}
	}
	if observed.Mirror != nil || len(observed.Sources) != 0 || observed.MaxAge != 0 {
		return ownedConfigError("metadata.sources/mirror/ttl", "none", "configured")
	}
	return nil
}

func validateManagedStreamConfig(observed, expected jetstream.StreamConfig) error {
	if observed.Name != expected.Name {
		return ownedConfigError("stream.name", expected.Name, observed.Name)
	}
	if !sameSubjects(observed.Subjects, expected.Subjects) {
		return ownedConfigError(expected.Name+".subjects", strings.Join(expected.Subjects, ","), strings.Join(observed.Subjects, ","))
	}
	checks := []struct {
		field    string
		expected any
		observed any
	}{
		{"storage", expected.Storage, observed.Storage},
		{"retention", expected.Retention, observed.Retention},
		{"max-age", expected.MaxAge, observed.MaxAge},
		{"max-bytes", expected.MaxBytes, observed.MaxBytes},
		{"discard", expected.Discard, observed.Discard},
	}
	if expected.MaxMsgs > 0 {
		checks = append(checks, struct {
			field    string
			expected any
			observed any
		}{"max-messages", expected.MaxMsgs, observed.MaxMsgs})
	}
	if expected.MaxMsgsPerSubject > 0 {
		checks = append(checks, struct {
			field    string
			expected any
			observed any
		}{"max-messages-per-subject", expected.MaxMsgsPerSubject, observed.MaxMsgsPerSubject})
	}
	for _, check := range checks {
		if fmt.Sprint(check.expected) != fmt.Sprint(check.observed) {
			return ownedConfigError(expected.Name+"."+check.field, check.expected, check.observed)
		}
	}
	return nil
}

func ownedConfigError(field string, expected, observed any) error {
	return fmt.Errorf("state schema configuration rejected: field=%s expected=%v observed=%v; clear/reset the selected NATS store before restart; existing data was not modified", field, expected, observed)
}

func sameSubjects(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	left = append([]string(nil), left...)
	right = append([]string(nil), right...)
	sort.Strings(left)
	sort.Strings(right)
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func postureForObservedSchema(version string) (StatePosture, error) {
	if version != StateSchemaVersionBeta160 {
		return StatePosture{}, schemaStateError(version, nil)
	}
	return StatePosture{StateSchemaVersion: version, ReusedState: true}, nil
}

func schemaStateError(observed string, conflicts []string) error {
	detail := ""
	if len(conflicts) != 0 {
		detail = "; managed-resources=" + strings.Join(conflicts, ",")
	}
	return fmt.Errorf("state schema rejected: expected=%s observed=%s%s; clear/reset the selected NATS store before restart; stored data was not transformed, copied, adopted, upgraded, downgraded, overwritten, or deleted",
		StateSchemaVersionBeta160, observed, detail)
}

func managedResourceConflicts(streams []string) []string {
	managedStreams := stringSet(managedStreamNames())
	managedBuckets := stringSet(managedGraphBucketNames())
	var conflicts []string
	for _, stream := range streams {
		if _, managed := managedStreams[stream]; managed {
			conflicts = append(conflicts, "stream:"+stream)
		}
		if strings.HasPrefix(stream, natsclient.KVStreamPrefix) {
			bucket := strings.TrimPrefix(stream, natsclient.KVStreamPrefix)
			if _, managed := managedBuckets[bucket]; managed {
				conflicts = append(conflicts, "kv:"+bucket)
			}
		}
	}
	sort.Strings(conflicts)
	return conflicts
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
