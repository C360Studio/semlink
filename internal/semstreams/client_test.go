package semstreams

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/projector"
	"github.com/c360studio/semstreams/graph"
	semerrs "github.com/c360studio/semstreams/pkg/errs"
	graphingest "github.com/c360studio/semstreams/processor/graph-ingest"
)

type fakeRequester struct {
	calls []string
}

func (f *fakeRequester) Request(_ context.Context, subject string, _ []byte, _ time.Duration) ([]byte, error) {
	return f.RequestClassified(context.Background(), subject, nil, 0)
}

func (f *fakeRequester) RequestClassified(_ context.Context, subject string, _ []byte, _ time.Duration) ([]byte, error) {
	f.calls = append(f.calls, subject)
	switch subject {
	case graphingest.SubjectEntityCreateWithTriples:
		return nil, semerrs.ClassifiedCode(semerrs.ErrorInvalid, graph.ErrorCodeEntityExists, errors.New("entity already exists"))
	case graphingest.SubjectEntityUpdateWithTriples:
		return json.Marshal(graph.UpdateEntityWithTriplesResponse{
			MutationResponse: graph.MutationResponse{KVRevision: 9},
			TriplesAdded:     2,
		})
	default:
		return []byte(`{}`), nil
	}
}

func TestUpsertFallsBackToMustExistUpdateOnCreateConflict(t *testing.T) {
	req := &fakeRequester{}
	client := NewGraphClient(req, time.Second)
	payload := projector.VehicleStatePayload{
		ID:                  projector.VehicleEntityID(1),
		Callsign:            "UAV-001",
		SystemID:            1,
		LinkStatus:          "online",
		TelemetrySampleTime: time.Unix(10, 0),
	}
	result, err := client.UpsertProjection(context.Background(), projector.Projection{
		Entity:          &graph.EntityState{ID: payload.ID, MessageType: projector.VehicleTelemetryType},
		Triples:         payload.Triples(),
		IndexingProfile: payload.IndexingProfile(),
	})
	if err != nil {
		t.Fatalf("UpsertProjection: %v", err)
	}
	if result.Created {
		t.Fatal("result marked created after create conflict")
	}
	if result.Revision != 9 {
		t.Fatalf("revision = %d", result.Revision)
	}
	if len(req.calls) != 2 || req.calls[0] != graphingest.SubjectEntityCreateWithTriples || req.calls[1] != graphingest.SubjectEntityUpdateWithTriples {
		t.Fatalf("calls = %#v", req.calls)
	}

	result, err = client.UpsertProjection(context.Background(), projector.Projection{
		Entity:          &graph.EntityState{ID: payload.ID, MessageType: projector.VehicleTelemetryType},
		Triples:         payload.Triples(),
		IndexingProfile: payload.IndexingProfile(),
	})
	if err != nil {
		t.Fatalf("second UpsertProjection: %v", err)
	}
	if result.Created {
		t.Fatal("second result marked created after entity was cached as existing")
	}
	if len(req.calls) != 3 || req.calls[2] != graphingest.SubjectEntityUpdateWithTriples {
		t.Fatalf("second write did not use direct update, calls = %#v", req.calls)
	}
}
