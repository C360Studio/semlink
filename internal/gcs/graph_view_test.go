package gcs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/projector"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
)

type fakeEntityQuerier struct {
	entities map[string]*graph.EntityState
}

func (f fakeEntityQuerier) QueryEntity(_ context.Context, id string) (*graph.EntityState, error) {
	return f.entities[id], nil
}

func TestHandleGraphReturnsSourceAwareVehicleLens(t *testing.T) {
	now := time.Date(2026, 6, 15, 12, 0, 0, 0, time.UTC)
	store := NewStore("nats://demo", true)
	vehicle := projector.VehicleStatePayload{
		ID:               "c360.semlink.robotics.fleet.drone.uav-001",
		Callsign:         "UAV-001",
		SystemID:         1,
		Mode:             "guided",
		FlightStatus:     "active",
		LinkStatus:       "online",
		LastSeen:         now,
		BatteryRemaining: 18,
		LatitudeDeg:      38.88,
		LongitudeDeg:     -77.03,
		AltitudeM:        120,
		GroundSpeedMS:    5.2,
		HeadingDeg:       90,
	}
	alert := projector.AlertPayload{
		ID:            "c360.semlink.robotics.fleet.alert.low-battery-uav-001",
		Kind:          "low-battery",
		Severity:      "warning",
		Active:        true,
		SubjectEntity: vehicle.ID,
		Message:       "UAV-001 battery is 18%",
		RaisedAt:      now,
	}
	store.ApplyProjectorSnapshot([]projector.VehicleStatePayload{vehicle}, []projector.AlertPayload{alert})
	store.RecordGraphWrite(vehicle.ID, 12, time.Millisecond)
	store.RecordGraphWrite(alert.ID, 13, time.Millisecond)

	graphClient := fakeEntityQuerier{entities: map[string]*graph.EntityState{
		vehicle.ID: {
			ID: vehicle.ID,
			Triples: []message.Triple{
				{Subject: vehicle.ID, Predicate: projector.PredicateVehicleCallsign, Object: "UAV-001", Source: projector.SourceProjector},
				{Subject: vehicle.ID, Predicate: projector.PredicateBatteryRemainingPct, Object: 18, Source: projector.SourceMAVLink},
				{Subject: vehicle.ID, Predicate: projector.PredicateLinkStatus, Object: "online", Source: projector.SourceProjector},
			},
		},
	}}

	server := NewServer(store, nil, "", ServerOptions{Graph: graphClient})
	req := httptest.NewRequest(http.MethodGet, "/api/graph?vehicle_id="+vehicle.ID, nil)
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var view GraphView
	if err := json.Unmarshal(rr.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode graph view: %v", err)
	}
	if view.VehicleID != vehicle.ID {
		t.Fatalf("vehicle id = %q, want %q", view.VehicleID, vehicle.ID)
	}
	if len(view.Lenses) != 2 {
		t.Fatalf("lenses = %d, want 2", len(view.Lenses))
	}
	local := view.Lenses[0]
	if local.Source != "semlink" || local.Status != "live" {
		t.Fatalf("local lens source/status = %q/%q", local.Source, local.Status)
	}
	if len(local.Nodes) < 5 {
		t.Fatalf("local nodes = %d, want at least vehicle, fact groups, and alert", len(local.Nodes))
	}
	if len(local.Facts) == 0 || local.Facts[0].Predicate != projector.PredicateVehicleCallsign {
		t.Fatalf("first fact = %#v, want callsign fact", local.Facts)
	}
	projection := view.Lenses[1]
	if projection.Source != "csapi" || projection.Status != "disabled" {
		t.Fatalf("projection lens source/status = %q/%q", projection.Source, projection.Status)
	}
}
