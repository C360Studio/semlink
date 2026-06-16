package tak

import (
	"context"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/cop"
	"github.com/c360studio/semlink/internal/cot"
	"github.com/c360studio/semlink/internal/gcs"
	"github.com/c360studio/semlink/internal/graphprojection"
	semruntime "github.com/c360studio/semlink/internal/semstreams"
)

type fakeGraph struct {
	projections []graphprojection.Projection
}

func (f *fakeGraph) UpsertProjection(_ context.Context, p graphprojection.Projection) (*semruntime.WriteResult, error) {
	f.projections = append(f.projections, p)
	return &semruntime.WriteResult{EntityID: p.Entity.ID, Revision: 17, Triples: len(p.Triples), Profile: p.IndexingProfile}, nil
}

func TestEventsFromSnapshotEmitsVehicleTracksAndAlerts(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 30, 0, 0, time.UTC)
	snapshot := gcs.Snapshot{
		Vehicles: []gcs.VehicleView{{
			EntityID:        "c360.semlink.robotics.fleet.drone.uav-001",
			Callsign:        "UAV-001",
			SystemID:        1,
			LatitudeDeg:     38.9,
			LongitudeDeg:    -77.03,
			AltitudeM:       120,
			GroundSpeedMS:   6.2,
			HeadingDeg:      90,
			LastSeen:        now,
			IndexingProfile: "signal",
		}},
		Alerts: []gcs.AlertView{{
			EntityID:      "c360.semlink.robotics.fleet.alert.low-battery-uav-001",
			Kind:          "low-battery",
			Severity:      "warning",
			Active:        true,
			SubjectEntity: "c360.semlink.robotics.fleet.drone.uav-001",
			Message:       "UAV-001 battery is 18%",
			RaisedAt:      now,
		}},
	}

	events := EventsFromSnapshot(snapshot, now)
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2", len(events))
	}
	if events[0].UID != "uav-001" || events[0].Type != cot.TypeAirTrack || !events[0].HasTrack {
		t.Fatalf("track event = %#v", events[0])
	}
	if events[1].Type != cot.TypeAlert || events[1].Remarks == "" {
		t.Fatalf("alert event = %#v", events[1])
	}
}

func TestInboundCoTProjectsCOPIntoStoreAndGraph(t *testing.T) {
	store := gcs.NewStore("nats://example", false)
	graph := &fakeGraph{}
	bridge, err := NewBridge(Config{}, store, graph)
	if err != nil {
		t.Fatalf("NewBridge: %v", err)
	}
	raw, err := cot.Marshal(cot.Event{
		UID:      "ANDROID-1",
		Type:     cot.TypeOperatorPosition,
		Time:     time.Date(2026, 6, 16, 12, 30, 0, 0, time.UTC),
		Point:    &cot.Point{Lat: 38.9, Lon: -77.03, HAE: 22},
		Callsign: "ALPHA",
	})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	bridge.handleInbound(context.Background(), raw)

	snapshot := store.Snapshot()
	if len(snapshot.Operators) != 1 {
		t.Fatalf("operators = %#v", snapshot.Operators)
	}
	if snapshot.Operators[0].Kind != cop.KindOperator || snapshot.Operators[0].GraphRevision != 17 {
		t.Fatalf("operator view = %#v", snapshot.Operators[0])
	}
	if len(graph.projections) != 1 || graph.projections[0].IndexingProfile != "signal" {
		t.Fatalf("graph projections = %#v", graph.projections)
	}
}
