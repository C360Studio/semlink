package semstreams

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nats-io/nats.go/jetstream"
)

type fakeSchemaNamespace struct {
	streams       []string
	listErr       error
	storeExists   bool
	stamp         string
	readErr       error
	createErr     error
	stampAfterCAS string
	operations    []string
	configErr     error
}

func (f *fakeSchemaNamespace) ListStreamNames(context.Context) ([]string, error) {
	f.operations = append(f.operations, "list")
	return append([]string(nil), f.streams...), f.listErr
}

func (f *fakeSchemaNamespace) OpenSchemaStore(context.Context) (schemaStampStore, error) {
	f.operations = append(f.operations, "open-meta")
	if !f.storeExists {
		return nil, errSchemaStoreNotFound
	}
	return f, nil
}

func (f *fakeSchemaNamespace) EnsureSchemaStore(context.Context) (schemaStampStore, error) {
	f.operations = append(f.operations, "ensure-meta")
	f.storeExists = true
	return f, nil
}

func (f *fakeSchemaNamespace) ValidateSchemaStore(context.Context) error {
	f.operations = append(f.operations, "validate-meta")
	return f.configErr
}

func (f *fakeSchemaNamespace) ReadSchemaStamp(context.Context) (string, error) {
	f.operations = append(f.operations, "read-stamp")
	if f.readErr != nil {
		return "", f.readErr
	}
	if f.stamp == "" {
		return "", errSchemaStampNotFound
	}
	return f.stamp, nil
}

func TestRuntimeMetadataConfigRejectsEveryOwnedInvariantDivergence(t *testing.T) {
	valid := runtimeMetadataStreamConfig()
	tests := []struct {
		name   string
		mutate func(*jetstream.StreamConfig)
	}{
		{name: "memory storage", mutate: func(c *jetstream.StreamConfig) { c.Storage = jetstream.MemoryStorage }},
		{name: "wrong subject", mutate: func(c *jetstream.StreamConfig) { c.Subjects = []string{"other.>"} }},
		{name: "history two", mutate: func(c *jetstream.StreamConfig) { c.MaxMsgsPerSubject = 2 }},
		{name: "ttl", mutate: func(c *jetstream.StreamConfig) { c.MaxAge = time.Hour }},
		{name: "wrong retention", mutate: func(c *jetstream.StreamConfig) { c.Retention = jetstream.InterestPolicy }},
		{name: "wrong discard", mutate: func(c *jetstream.StreamConfig) { c.Discard = jetstream.DiscardOld }},
		{name: "rollup disabled", mutate: func(c *jetstream.StreamConfig) { c.AllowRollup = false }},
		{name: "delete allowed", mutate: func(c *jetstream.StreamConfig) { c.DenyDelete = false }},
		{name: "direct disabled", mutate: func(c *jetstream.StreamConfig) { c.AllowDirect = false }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			observed := valid
			observed.Subjects = append([]string(nil), valid.Subjects...)
			tt.mutate(&observed)
			err := validateRuntimeMetadataStreamConfig(observed)
			if err == nil || !strings.Contains(err.Error(), "clear/reset") {
				t.Fatalf("metadata config error = %v", err)
			}
		})
	}
}

func TestManagedStreamConfigRejectsStorageSubjectAndBoundDivergence(t *testing.T) {
	for _, expected := range []jetstream.StreamConfig{entityStreamConfig(), rawMAVLinkStreamConfig()} {
		tests := []struct {
			name   string
			mutate func(*jetstream.StreamConfig)
		}{
			{name: "memory storage", mutate: func(c *jetstream.StreamConfig) { c.Storage = jetstream.MemoryStorage }},
			{name: "wrong subject", mutate: func(c *jetstream.StreamConfig) { c.Subjects = []string{"wrong.>"} }},
			{name: "unbounded age", mutate: func(c *jetstream.StreamConfig) { c.MaxAge = 0 }},
			{name: "unbounded bytes", mutate: func(c *jetstream.StreamConfig) { c.MaxBytes = -1 }},
			{name: "wrong retention", mutate: func(c *jetstream.StreamConfig) { c.Retention = jetstream.InterestPolicy }},
			{name: "wrong discard", mutate: func(c *jetstream.StreamConfig) { c.Discard = jetstream.DiscardNew }},
		}
		for _, tt := range tests {
			t.Run(expected.Name+"/"+tt.name, func(t *testing.T) {
				observed := expected
				observed.Subjects = append([]string(nil), expected.Subjects...)
				tt.mutate(&observed)
				if err := validateManagedStreamConfig(observed, expected); err == nil || !strings.Contains(err.Error(), "clear/reset") {
					t.Fatalf("managed stream config error = %v", err)
				}
			})
		}
	}
}

func (f *fakeSchemaNamespace) CreateSchemaStamp(context.Context, string) error {
	f.operations = append(f.operations, "create-stamp")
	if f.stampAfterCAS != "" {
		f.stamp = f.stampAfterCAS
	}
	if f.createErr != nil {
		return f.createErr
	}
	f.stamp = StateSchemaVersionBeta160
	return nil
}

func TestBootstrapStateSchemaPosturesAndRejections(t *testing.T) {
	tests := []struct {
		name       string
		ns         *fakeSchemaNamespace
		wantFresh  bool
		wantReused bool
		wantErr    []string
		noMutation bool
	}{
		{name: "empty first initialization", ns: &fakeSchemaNamespace{}, wantFresh: true},
		{name: "exact beta160 restart", ns: &fakeSchemaNamespace{storeExists: true, stamp: StateSchemaVersionBeta160, streams: []string{"KV_SEMLINK_RUNTIME_META", "ENTITY"}}, wantReused: true},
		{name: "metadata bucket without key", ns: &fakeSchemaNamespace{storeExists: true, streams: []string{"KV_SEMLINK_RUNTIME_META"}}, wantFresh: true},
		{name: "divergent metadata bucket without key", ns: &fakeSchemaNamespace{storeExists: true, streams: []string{"KV_SEMLINK_RUNTIME_META"}, configErr: errors.New("metadata config rejected; clear/reset")}, wantErr: []string{"metadata config rejected", "clear/reset"}, noMutation: true},
		{name: "missing stamp beside managed resources", ns: &fakeSchemaNamespace{streams: []string{"ENTITY"}}, wantErr: []string{"expected=beta.160", "observed=missing", "stream:ENTITY", "clear/reset"}, noMutation: true},
		{name: "older stamp", ns: &fakeSchemaNamespace{storeExists: true, stamp: "beta.141", streams: []string{"KV_SEMLINK_RUNTIME_META"}}, wantErr: []string{"expected=beta.160", "observed=beta.141", "clear/reset"}, noMutation: true},
		{name: "newer stamp", ns: &fakeSchemaNamespace{storeExists: true, stamp: "beta.161", streams: []string{"KV_SEMLINK_RUNTIME_META"}}, wantErr: []string{"expected=beta.160", "observed=beta.161", "clear/reset"}, noMutation: true},
		{name: "CAS loser converges exact", ns: &fakeSchemaNamespace{createErr: errSchemaCASLost, stampAfterCAS: StateSchemaVersionBeta160}, wantReused: true},
		{name: "CAS loser rejects other", ns: &fakeSchemaNamespace{createErr: errSchemaCASLost, stampAfterCAS: "beta.141"}, wantErr: []string{"expected=beta.160", "observed=beta.141", "clear/reset"}},
		{name: "enumeration failure", ns: &fakeSchemaNamespace{listErr: errors.New("names unavailable")}, wantErr: []string{"enumerate", "names unavailable"}, noMutation: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := bootstrapStateSchema(context.Background(), tt.ns)
			if len(tt.wantErr) == 0 {
				if err != nil {
					t.Fatalf("bootstrapStateSchema: %v", err)
				}
				if got.StateSchemaVersion != StateSchemaVersionBeta160 || got.FreshState != tt.wantFresh || got.ReusedState != tt.wantReused || got.FreshState == got.ReusedState {
					t.Fatalf("posture = %#v", got)
				}
				return
			}
			if err == nil {
				t.Fatalf("bootstrapStateSchema accepted invalid state: %#v", got)
			}
			for _, want := range tt.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("error %q lacks %q", err, want)
				}
			}
			if tt.noMutation && (strings.Contains(strings.Join(tt.ns.operations, ","), "ensure-meta") || strings.Contains(strings.Join(tt.ns.operations, ","), "create-stamp")) {
				t.Fatalf("rejected state was mutated: %v", tt.ns.operations)
			}
		})
	}
}

func TestBootstrapStateSchemaRejectsEveryUnstampedManagedResource(t *testing.T) {
	resources := append([]string(nil), managedStreamNames()...)
	for _, bucket := range managedGraphBucketNames() {
		resources = append(resources, "KV_"+bucket)
	}
	for _, resource := range resources {
		t.Run(resource, func(t *testing.T) {
			ns := &fakeSchemaNamespace{streams: []string{resource}}
			_, err := bootstrapStateSchema(context.Background(), ns)
			if err == nil || !strings.Contains(err.Error(), "observed=missing") || !strings.Contains(err.Error(), "clear/reset") {
				t.Fatalf("unstamped managed resource error = %v", err)
			}
			if strings.Contains(strings.Join(ns.operations, ","), "ensure-meta") || strings.Contains(strings.Join(ns.operations, ","), "create-stamp") {
				t.Fatalf("unstamped resource was adopted: %v", ns.operations)
			}
		})
	}
}
