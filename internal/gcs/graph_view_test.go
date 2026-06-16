package gcs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/cop"
	"github.com/c360studio/semlink/internal/projector"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
)

type fakeEntityQuerier struct {
	entities map[string]*graph.EntityState
}

func TestHandleGraphReturnsCOPLens(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	store := NewStore("nats://demo", true)
	marker := cop.View{
		Kind:            cop.KindMarker,
		EntityID:        "c360.semlink.cop.marker.poi.marker1",
		UID:             "MARKER-1",
		Label:           "North Gate",
		Description:     "checkpoint",
		LatitudeDeg:     38.894,
		LongitudeDeg:    -77.038,
		HasPosition:     true,
		LastSeen:        now,
		IndexingProfile: "content",
	}
	store.ApplyCOPView(marker)
	store.RecordGraphWrite(marker.EntityID, 21, time.Millisecond)

	graphClient := fakeEntityQuerier{entities: map[string]*graph.EntityState{
		marker.EntityID: {
			ID: marker.EntityID,
			Triples: []message.Triple{
				{Subject: marker.EntityID, Predicate: cop.PredicateKind, Object: string(cop.KindMarker), Source: cop.SourceCOP},
				{Subject: marker.EntityID, Predicate: cop.PredicateLabel, Object: "North Gate", Source: cop.SourceCoT},
			},
		},
	}}

	server := NewServer(store, nil, "", ServerOptions{Graph: graphClient})
	req := httptest.NewRequest(http.MethodGet, "/api/graph?entity_id="+marker.EntityID, nil)
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var view GraphView
	if err := json.Unmarshal(rr.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode graph view: %v", err)
	}
	if view.EntityID != marker.EntityID || view.EntityKind != string(cop.KindMarker) {
		t.Fatalf("entity = %q/%q, want %q/%q", view.EntityID, view.EntityKind, marker.EntityID, cop.KindMarker)
	}
	if len(view.Lenses) != 2 || view.Lenses[0].Status != "live" || view.Lenses[1].Status != "disabled" {
		t.Fatalf("lenses = %#v, want live COP lens", view.Lenses)
	}
	if len(view.Lenses[0].Facts) == 0 || view.Lenses[0].Facts[0].Predicate != cop.PredicateKind {
		t.Fatalf("facts = %#v, want COP graph facts", view.Lenses[0].Facts)
	}
}

func TestHandleGraphReturnsCOPCSAPILens(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	operatorID, err := cop.EntityID(cop.KindOperator, "ANDROID-1")
	if err != nil {
		t.Fatalf("operator id: %v", err)
	}
	messageID, err := cop.EntityID(cop.KindMessage, "chat-1")
	if err != nil {
		t.Fatalf("message id: %v", err)
	}
	store := NewStore("nats://demo", true)
	store.ApplyCOPView(cop.View{
		Kind:            cop.KindOperator,
		EntityID:        operatorID,
		UID:             "ANDROID-1",
		Callsign:        "ALPHA",
		LatitudeDeg:     38.892,
		LongitudeDeg:    -77.035,
		HasPosition:     true,
		LastSeen:        now,
		IndexingProfile: "signal",
	})
	chat := cop.View{
		Kind:            cop.KindMessage,
		EntityID:        messageID,
		UID:             "chat-1",
		Callsign:        "ALPHA",
		Text:            "hold at checkpoint",
		SenderUID:       "ANDROID-1",
		SenderEntity:    operatorID,
		HasPosition:     true,
		LatitudeDeg:     38.892,
		LongitudeDeg:    -77.035,
		LastSeen:        now,
		IndexingProfile: "content",
	}
	store.ApplyCOPView(chat)
	store.RecordGraphWrite(chat.EntityID, 42, time.Millisecond)

	csapi := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/systems":
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{{
				"id":   "c360.semconnect.systems.csapi.system.android-1",
				"name": "ALPHA",
				"properties": map[string]any{
					"uid":  "ANDROID-1",
					"name": "ALPHA",
				},
			}}})
		case "/systemEvents":
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{{
				"id":        "c360.semlink.cop.csapi.event.chat-1",
				"system@id": "c360.semconnect.systems.csapi.system.android-1",
				"eventType": "GeoChat",
				"message":   "hold at checkpoint",
				"payload": map[string]any{
					"sender_uid":  "ANDROID-1",
					"semlink_cop": messageID,
				},
			}}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
		}
	}))
	defer csapi.Close()

	graphClient := fakeEntityQuerier{entities: map[string]*graph.EntityState{
		chat.EntityID: {
			ID: chat.EntityID,
			Triples: []message.Triple{
				{Subject: chat.EntityID, Predicate: cop.PredicateKind, Object: string(cop.KindMessage), Source: cop.SourceCOP},
				{Subject: chat.EntityID, Predicate: cop.PredicateMessageText, Object: "hold at checkpoint", Source: cop.SourceCoT},
				{Subject: chat.EntityID, Predicate: cop.PredicateMessageSenderUID, Object: "ANDROID-1", Source: cop.SourceCoT},
			},
		},
	}}

	server := NewServer(store, nil, "", ServerOptions{Graph: graphClient, CSAPIURL: csapi.URL})
	req := httptest.NewRequest(http.MethodGet, "/api/graph?entity_id="+chat.EntityID, nil)
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var view GraphView
	if err := json.Unmarshal(rr.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode graph view: %v", err)
	}
	if len(view.Lenses) != 2 || view.Lenses[1].Source != "csapi" || view.Lenses[1].Status != "live" {
		t.Fatalf("lenses = %#v, want live CS API COP lens", view.Lenses)
	}
	if len(view.Lenses[1].Nodes) < 2 || len(view.Lenses[1].Edges) == 0 {
		t.Fatalf("csapi lens graph = %#v, want sender system and GeoChat event", view.Lenses[1])
	}
}

func TestHandleGraphReturnsVehicleCSAPIDirectDatastreams(t *testing.T) {
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	store := NewStore("nats://demo", true)
	vehicle := projector.VehicleStatePayload{
		ID:               "c360.semlink.robotics.fleet.drone.uav-006",
		Callsign:         "UAV-006",
		SystemID:         6,
		Mode:             "guided",
		FlightStatus:     "active",
		LinkStatus:       "online",
		LastSeen:         now,
		BatteryRemaining: 58,
		LatitudeDeg:      38.89,
		LongitudeDeg:     -77.04,
		AltitudeM:        98,
		GroundSpeedMS:    5.5,
		HeadingDeg:       80,
	}
	store.ApplyProjectorSnapshot([]projector.VehicleStatePayload{vehicle}, nil)
	store.RecordGraphWrite(vehicle.ID, 66, time.Millisecond)
	vehicleView := store.Snapshot().Vehicles[0]

	csapi := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/systems/" + csapiVehicleSystemID(vehicleView):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":   csapiVehicleSystemID(vehicleView),
				"name": vehicleView.Callsign,
			})
		case "/datastreams/" + csapiVehicleDatastreamID(vehicleView, "battery"):
			writeDatastream(w, vehicleView, "battery", "Battery remaining", "https://c360.studio/def/robot/power/batteryRemaining")
		case "/datastreams/" + csapiVehicleDatastreamID(vehicleView, "altitude"):
			writeDatastream(w, vehicleView, "altitude", "Altitude", "https://c360.studio/def/robot/position/altitude")
		case "/datastreams/" + csapiVehicleDatastreamID(vehicleView, "ground-speed"):
			writeDatastream(w, vehicleView, "ground-speed", "Ground speed", "https://c360.studio/def/robot/motion/groundSpeed")
		case "/datastreams/" + csapiVehicleDatastreamID(vehicleView, "battery") + "/observations",
			"/datastreams/" + csapiVehicleDatastreamID(vehicleView, "altitude") + "/observations",
			"/datastreams/" + csapiVehicleDatastreamID(vehicleView, "ground-speed") + "/observations":
			_ = json.NewEncoder(w).Encode(map[string]any{"numberMatched": 7, "items": []map[string]any{}})
		default:
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]any{}})
		}
	}))
	defer csapi.Close()

	server := NewServer(store, nil, "", ServerOptions{Graph: fakeEntityQuerier{}, CSAPIURL: csapi.URL})
	req := httptest.NewRequest(http.MethodGet, "/api/graph?entity_id="+vehicle.ID, nil)
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var view GraphView
	if err := json.Unmarshal(rr.Body.Bytes(), &view); err != nil {
		t.Fatalf("decode graph view: %v", err)
	}
	projection := view.Lenses[1]
	if projection.Source != "csapi" || projection.Status != "live" {
		t.Fatalf("projection source/status = %q/%q", projection.Source, projection.Status)
	}
	if statValue(projection.Stats, "datastreams") != "3" {
		t.Fatalf("projection stats = %#v, want 3 datastreams", projection.Stats)
	}
	expectedHistoryLabels := map[string]string{
		"battery":      "Battery history",
		"altitude":     "Altitude history",
		"ground-speed": "Ground speed history",
	}
	for _, key := range csapiVehicleTelemetryKeys {
		streamID := csapiVehicleDatastreamID(vehicleView, key)
		observationsID := streamID + "#observations"
		if !containsGraphNode(projection.Nodes, streamID) {
			t.Fatalf("projection nodes missing %s: %#v", streamID, projection.Nodes)
		}
		if !containsGraphNode(projection.Nodes, observationsID) {
			t.Fatalf("projection nodes missing observations for %s: %#v", streamID, projection.Nodes)
		}
		if !containsGraphEdge(projection.Edges, streamID, observationsID) {
			t.Fatalf("projection edges missing %s -> %s: %#v", streamID, observationsID, projection.Edges)
		}
		if label := graphNodeLabel(projection.Nodes, observationsID); label != expectedHistoryLabels[key] {
			t.Fatalf("history node label for %s = %q, want %q", key, label, expectedHistoryLabels[key])
		}
	}
}

func writeDatastream(w http.ResponseWriter, vehicle VehicleView, key, label, observedProperty string) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"id":               csapiVehicleDatastreamID(vehicle, key),
		"name":             vehicle.Callsign + " " + label,
		"system@id":        csapiVehicleSystemID(vehicle),
		"observedProperty": observedProperty,
	})
}

func containsGraphNode(nodes []GraphNode, id string) bool {
	for _, node := range nodes {
		if node.ID == id {
			return true
		}
	}
	return false
}

func containsGraphEdge(edges []GraphEdge, from, to string) bool {
	for _, edge := range edges {
		if edge.From == from && edge.To == to {
			return true
		}
	}
	return false
}

func graphNodeLabel(nodes []GraphNode, id string) string {
	for _, node := range nodes {
		if node.ID == id {
			return node.Label
		}
	}
	return ""
}

func statValue(stats []GraphStat, label string) string {
	for _, stat := range stats {
		if stat.Label == label {
			return stat.Value
		}
	}
	return ""
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
