package semstreams

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/projector"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/pkg/errs"
	"github.com/c360studio/semstreams/pkg/projection"
)

type fakeMutationPort struct {
	creates    []projection.CreateMutation
	reconciles []projection.ReconcileMutation
	reads      []string
	create     func(int, projection.CreateMutation) (projection.MutationReceipt, error)
	reconcile  func(int, projection.ReconcileMutation) (projection.MutationReceipt, error)
	read       func(int, string) (*graph.ExactEntity, error)
}

func (f *fakeMutationPort) Create(_ context.Context, request projection.CreateMutation) (projection.MutationReceipt, error) {
	f.creates = append(f.creates, request)
	if f.create != nil {
		return f.create(len(f.creates)-1, request)
	}
	return verifiedReceipt(request.Entity, 1), nil
}

func (f *fakeMutationPort) Reconcile(_ context.Context, request projection.ReconcileMutation) (projection.MutationReceipt, error) {
	f.reconciles = append(f.reconciles, request)
	if f.reconcile != nil {
		return f.reconcile(len(f.reconciles)-1, request)
	}
	return verifiedReceipt(entityWithTriples(request.EntityID, request.Desired), uint64(len(f.reconciles)+1)), nil
}

func (f *fakeMutationPort) ReadAuthoritative(_ context.Context, entityID string) (*graph.ExactEntity, error) {
	f.reads = append(f.reads, entityID)
	if f.read != nil {
		return f.read(len(f.reads)-1, entityID)
	}
	return &graph.ExactEntity{Entity: entityWithTriples(entityID, nil), KVRevision: 1}, nil
}

func TestUpsertReconcilesFirstThenCreatesEmptyEnvelopeAndReconciles(t *testing.T) {
	port := &fakeMutationPort{}
	port.reconcile = func(call int, request projection.ReconcileMutation) (projection.MutationReceipt, error) {
		if call == 0 {
			return projection.MutationReceipt{Commit: projection.CommitNotCommitted}, mutationError(
				projection.MutationOperationReadAuthoritative, projection.MutationNotFound, projection.CommitNotCommitted,
			)
		}
		return verifiedReceipt(entityWithTriples(request.EntityID, request.Desired), 7), nil
	}
	client := testGraphClient(port)

	result, err := client.UpsertProjection(context.Background(), vehicleProjection())
	if err != nil {
		t.Fatalf("UpsertProjection: %v", err)
	}
	if !result.Created || result.Revision != 7 || result.Triples != len(vehicleProjection().Triples) {
		t.Fatalf("result = %#v", result)
	}
	if len(port.reconciles) != 2 || len(port.creates) != 1 {
		t.Fatalf("reconciles=%d creates=%d", len(port.reconciles), len(port.creates))
	}
	create := port.creates[0]
	if len(create.Triples) != 0 || len(create.Entity.Triples) != 0 {
		t.Fatalf("create included facts: %#v", create)
	}
	if create.Metadata.RequestID == "" || create.Metadata.Source != createEnvelopeSource || create.Metadata.Timestamp.IsZero() {
		t.Fatalf("create metadata = %#v", create.Metadata)
	}
	if got := port.reconciles[1]; got.Contract != projector.VehicleTelemetryType.String() || got.Group != projector.VehicleTelemetryGroup {
		t.Fatalf("reconcile binding = %q/%q", got.Contract, got.Group)
	}
}

func TestUpsertConvergesCreateConflictThroughReconcile(t *testing.T) {
	port := &fakeMutationPort{}
	port.reconcile = func(call int, request projection.ReconcileMutation) (projection.MutationReceipt, error) {
		if call == 0 {
			return projection.MutationReceipt{Commit: projection.CommitNotCommitted}, mutationError(
				projection.MutationOperationReadAuthoritative, projection.MutationNotFound, projection.CommitNotCommitted,
			)
		}
		return verifiedReceipt(entityWithTriples(request.EntityID, request.Desired), 9), nil
	}
	port.create = func(_ int, _ projection.CreateMutation) (projection.MutationReceipt, error) {
		return projection.MutationReceipt{Commit: projection.CommitNotCommitted}, mutationError(
			projection.MutationOperationCreate, projection.MutationConflict, projection.CommitNotCommitted,
		)
	}

	result, err := testGraphClient(port).UpsertProjection(context.Background(), vehicleProjection())
	if err != nil {
		t.Fatalf("UpsertProjection: %v", err)
	}
	if result.Created || result.Revision != 9 || len(port.reconciles) != 2 {
		t.Fatalf("result=%#v reconciles=%d", result, len(port.reconciles))
	}
}

func TestUpsertPreservesMixedTripleProvenance(t *testing.T) {
	projectionValue := vehicleProjection()
	port := &fakeMutationPort{}
	port.reconcile = func(call int, request projection.ReconcileMutation) (projection.MutationReceipt, error) {
		if call == 0 {
			return projection.MutationReceipt{Commit: projection.CommitNotCommitted}, mutationError(
				projection.MutationOperationReadAuthoritative, projection.MutationNotFound, projection.CommitNotCommitted,
			)
		}
		return verifiedReceipt(entityWithTriples(request.EntityID, request.Desired), 4), nil
	}

	if _, err := testGraphClient(port).UpsertProjection(context.Background(), projectionValue); err != nil {
		t.Fatalf("UpsertProjection: %v", err)
	}
	request := port.reconciles[1]
	if request.Metadata.Source != "" || !request.Metadata.Timestamp.IsZero() {
		t.Fatalf("reconcile metadata collapsed provenance: %#v", request.Metadata)
	}
	for index, triple := range request.Desired {
		want := projectionValue.Triples[index]
		if triple.Source != want.Source || !triple.Timestamp.Equal(want.Timestamp) {
			t.Fatalf("triple[%d] provenance = %q/%s, want %q/%s", index, triple.Source, triple.Timestamp, want.Source, want.Timestamp)
		}
	}
}

func TestUpsertRetriesRevisionConflictWithinBound(t *testing.T) {
	port := &fakeMutationPort{}
	port.reconcile = func(_ int, _ projection.ReconcileMutation) (projection.MutationReceipt, error) {
		return projection.MutationReceipt{Commit: projection.CommitNotCommitted}, mutationError(
			projection.MutationOperationReconcile, projection.MutationRevisionConflict, projection.CommitNotCommitted,
		)
	}
	client := newGraphClient(port, projector.Contracts(), 3, func() string { return "request-1" })

	_, err := client.UpsertProjection(context.Background(), vehicleProjection())
	var mutationErr *projection.MutationError
	if !errors.As(err, &mutationErr) || mutationErr.Kind != projection.MutationRevisionConflict {
		t.Fatalf("error = %#v", err)
	}
	if len(port.reconciles) != 3 {
		t.Fatalf("reconcile calls = %d, want 3", len(port.reconciles))
	}
}

func TestUpsertReusesCreateMetadataAcrossSafeRetry(t *testing.T) {
	port := &fakeMutationPort{}
	port.reconcile = func(call int, request projection.ReconcileMutation) (projection.MutationReceipt, error) {
		if call < 2 {
			return projection.MutationReceipt{Commit: projection.CommitNotCommitted}, mutationError(
				projection.MutationOperationReadAuthoritative, projection.MutationNotFound, projection.CommitNotCommitted,
			)
		}
		return verifiedReceipt(entityWithTriples(request.EntityID, request.Desired), 12), nil
	}
	port.create = func(call int, request projection.CreateMutation) (projection.MutationReceipt, error) {
		if call == 0 {
			return projection.MutationReceipt{Commit: projection.CommitUnknown}, mutationError(
				projection.MutationOperationCreate, projection.MutationCommitUnknown, projection.CommitUnknown,
			)
		}
		return verifiedReceipt(request.Entity, 8), nil
	}
	port.read = func(_ int, _ string) (*graph.ExactEntity, error) {
		return nil, mutationError(projection.MutationOperationReadAuthoritative, projection.MutationNotFound, projection.CommitNotCommitted)
	}

	if _, err := testGraphClient(port).UpsertProjection(context.Background(), vehicleProjection()); err != nil {
		t.Fatalf("UpsertProjection: %v", err)
	}
	if len(port.creates) != 2 {
		t.Fatalf("create calls = %d", len(port.creates))
	}
	if first, second := port.creates[0].Metadata, port.creates[1].Metadata; first != second {
		t.Fatalf("create metadata changed across retry: %#v != %#v", first, second)
	}
}

func TestUpsertFinalBudgetCreateStillGetsImmediateReconcile(t *testing.T) {
	port := &fakeMutationPort{}
	port.reconcile = func(call int, request projection.ReconcileMutation) (projection.MutationReceipt, error) {
		if call == 0 {
			return projection.MutationReceipt{Commit: projection.CommitNotCommitted}, mutationError(
				projection.MutationOperationReadAuthoritative, projection.MutationNotFound, projection.CommitNotCommitted,
			)
		}
		return verifiedReceipt(entityWithTriples(request.EntityID, request.Desired), 21), nil
	}
	client := newGraphClient(port, projector.Contracts(), 1, func() string { return "request-final" })

	result, err := client.UpsertProjection(context.Background(), vehicleProjection())
	if err != nil {
		t.Fatalf("UpsertProjection: %v", err)
	}
	if !result.Created || result.Revision != 21 || len(port.creates) != 1 || len(port.reconciles) != 2 {
		t.Fatalf("result=%#v creates=%d reconciles=%d", result, len(port.creates), len(port.reconciles))
	}
}

func TestUpsertRejectsVerifiedReceiptWithZeroRevision(t *testing.T) {
	port := &fakeMutationPort{
		reconcile: func(_ int, request projection.ReconcileMutation) (projection.MutationReceipt, error) {
			return verifiedReceipt(entityWithTriples(request.EntityID, request.Desired), 0), nil
		},
	}
	result, err := testGraphClient(port).UpsertProjection(context.Background(), vehicleProjection())
	var mutationErr *projection.MutationError
	if result != nil || !errors.As(err, &mutationErr) || mutationErr.Kind != projection.MutationInternal || mutationErr.Commit != projection.CommitUnknown {
		t.Fatalf("result=%#v error=%#v", result, err)
	}
}

func TestUpsertRejectsInvalidProjectionInputsAndUnverifiedSuccess(t *testing.T) {
	valid := vehicleProjection()
	port := &fakeMutationPort{}
	client := testGraphClient(port)
	for _, test := range []struct {
		name   string
		client *GraphClient
		value  projector.Projection
	}{
		{name: "nil client", client: nil, value: valid},
		{name: "nil entity", client: client, value: projector.Projection{}},
		{name: "no triples", client: client, value: projector.Projection{Entity: valid.Entity}},
		{name: "no binding", client: client, value: projector.Projection{Entity: valid.Entity, Triples: valid.Triples}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if result, err := test.client.UpsertProjection(context.Background(), test.value); result != nil || err == nil {
				t.Fatalf("result=%#v err=%v", result, err)
			}
		})
	}

	port.reconcile = func(_ int, request projection.ReconcileMutation) (projection.MutationReceipt, error) {
		return projection.MutationReceipt{Entity: entityWithTriples(request.EntityID, request.Desired), KVRevision: 2, Commit: projection.CommitUnknown}, nil
	}
	result, err := client.UpsertProjection(context.Background(), valid)
	var mutationErr *projection.MutationError
	if result != nil || !errors.As(err, &mutationErr) || mutationErr.Kind != projection.MutationCommitUnknown {
		t.Fatalf("result=%#v err=%#v", result, err)
	}
}

func TestUpsertResolvesUnknownCreateThatEstablishedEnvelope(t *testing.T) {
	port := &fakeMutationPort{}
	port.reconcile = func(call int, request projection.ReconcileMutation) (projection.MutationReceipt, error) {
		if call == 0 {
			return projection.MutationReceipt{Commit: projection.CommitNotCommitted}, mutationError(
				projection.MutationOperationReadAuthoritative, projection.MutationNotFound, projection.CommitNotCommitted,
			)
		}
		return verifiedReceipt(entityWithTriples(request.EntityID, request.Desired), 14), nil
	}
	port.create = func(_ int, _ projection.CreateMutation) (projection.MutationReceipt, error) {
		return projection.MutationReceipt{Commit: projection.CommitUnknown}, mutationError(
			projection.MutationOperationCreate, projection.MutationCommitUnknown, projection.CommitUnknown,
		)
	}
	port.read = func(_ int, entityID string) (*graph.ExactEntity, error) {
		return &graph.ExactEntity{Entity: entityWithTriples(entityID, nil), KVRevision: 4}, nil
	}

	result, err := testGraphClient(port).UpsertProjection(context.Background(), vehicleProjection())
	if err != nil || result.Revision != 14 || result.Created {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestUpsertResolvesCommitUnknownByAuthoritativeComparison(t *testing.T) {
	want := vehicleProjection()
	port := &fakeMutationPort{}
	port.reconcile = func(_ int, _ projection.ReconcileMutation) (projection.MutationReceipt, error) {
		return projection.MutationReceipt{Commit: projection.CommitUnknown}, mutationError(
			projection.MutationOperationReconcile, projection.MutationCommitUnknown, projection.CommitUnknown,
		)
	}
	port.read = func(_ int, entityID string) (*graph.ExactEntity, error) {
		triples := cloneTriples(want.Triples)
		for index := range triples {
			triples[index].Context = "request-1"
		}
		return &graph.ExactEntity{Entity: entityWithTriples(entityID, triples), KVRevision: 33}, nil
	}
	client := newGraphClient(port, projector.Contracts(), 3, func() string { return "request-1" })

	result, err := client.UpsertProjection(context.Background(), want)
	if err != nil {
		t.Fatalf("UpsertProjection: %v", err)
	}
	if result.Revision != 33 || len(port.reads) != 1 || len(port.reconciles) != 1 {
		t.Fatalf("result=%#v reads=%d reconciles=%d", result, len(port.reads), len(port.reconciles))
	}
}

func TestUpsertPreservesDefiniteTypedOutcomes(t *testing.T) {
	for _, kind := range []projection.MutationErrorKind{
		projection.MutationInvalid,
		projection.MutationUnavailable,
		projection.MutationInternal,
	} {
		t.Run(string(kind), func(t *testing.T) {
			port := &fakeMutationPort{}
			port.reconcile = func(_ int, _ projection.ReconcileMutation) (projection.MutationReceipt, error) {
				return projection.MutationReceipt{Commit: projection.CommitNotCommitted}, mutationError(
					projection.MutationOperationReconcile, kind, projection.CommitNotCommitted,
				)
			}
			_, err := testGraphClient(port).UpsertProjection(context.Background(), vehicleProjection())
			var mutationErr *projection.MutationError
			if !errors.As(err, &mutationErr) || mutationErr.Kind != kind || mutationErr.Commit != projection.CommitNotCommitted {
				t.Fatalf("error = %#v", err)
			}
			if len(port.reconciles) != 1 {
				t.Fatalf("reconcile calls = %d", len(port.reconciles))
			}
		})
	}
}

func TestQueryExactEntityRetainsSameEntryRevision(t *testing.T) {
	port := &fakeMutationPort{
		read: func(_ int, entityID string) (*graph.ExactEntity, error) {
			entity := entityWithTriples(entityID, vehicleProjection().Triples)
			entity.Version = 91
			return &graph.ExactEntity{Entity: entity, KVRevision: 17}, nil
		},
	}
	client := testGraphClient(port)

	exact, err := client.QueryExactEntity(context.Background(), projector.VehicleEntityID(1))
	if err != nil {
		t.Fatalf("QueryExactEntity: %v", err)
	}
	if exact.KVRevision != 17 || exact.KVRevision == exact.Entity.Version {
		t.Fatalf("exact = %#v", exact)
	}
	entity, err := client.QueryEntity(context.Background(), projector.VehicleEntityID(1))
	if err != nil || entity.Version != 91 {
		t.Fatalf("QueryEntity entity=%#v err=%v", entity, err)
	}
}

func TestQueryExactEntityRejectsInvalidAuthoritativeResult(t *testing.T) {
	port := &fakeMutationPort{read: func(_ int, _ string) (*graph.ExactEntity, error) { return nil, nil }}
	if exact, err := testGraphClient(port).QueryExactEntity(context.Background(), projector.VehicleEntityID(1)); exact != nil || err == nil {
		t.Fatalf("exact=%#v err=%v", exact, err)
	}
}

func testGraphClient(port mutationPort) *GraphClient {
	return newGraphClient(port, projector.Contracts(), 3, func() string { return "request-1" })
}

func vehicleProjection() projector.Projection {
	payload := projector.VehicleStatePayload{
		ID:                  projector.VehicleEntityID(1),
		Callsign:            "UAV-001",
		SystemID:            1,
		LinkStatus:          "online",
		TelemetrySampleTime: time.Unix(10, 0).UTC(),
	}
	return projector.Projection{
		Entity:          &graph.EntityState{ID: payload.ID, MessageType: projector.VehicleTelemetryType},
		Triples:         payload.Triples(),
		IndexingProfile: payload.IndexingProfile(),
		Contract:        projector.VehicleTelemetryType.String(),
		Group:           projector.VehicleTelemetryGroup,
	}
}

func entityWithTriples(entityID string, triples []message.Triple) *graph.EntityState {
	return &graph.EntityState{
		ID:          entityID,
		MessageType: projector.VehicleTelemetryType,
		Triples:     cloneTriples(triples),
	}
}

func verifiedReceipt(entity *graph.EntityState, revision uint64) projection.MutationReceipt {
	return projection.MutationReceipt{Entity: entity, KVRevision: revision, Commit: projection.CommitVerified}
}

func mutationError(operation projection.MutationOperation, kind projection.MutationErrorKind, commit projection.CommitState) error {
	return &projection.MutationError{
		Operation: operation,
		Kind:      kind,
		Class:     errs.ErrorInvalid,
		Commit:    commit,
		Err:       errors.New(string(kind)),
	}
}
