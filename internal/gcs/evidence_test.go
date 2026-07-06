package gcs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/commandgate"
	"github.com/c360studio/semlink/internal/mesh"
	"github.com/c360studio/semlink/internal/projector"
	"github.com/c360studio/semlink/internal/rules"
)

func TestHandleEvidenceReturnsExternalConsumerContract(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	store := NewStore("nats://demo", true)
	vehicle := projector.VehicleStatePayload{
		ID:               "c360.semlink.robotics.fleet.drone.uav-003",
		Callsign:         "BOAT-003",
		SystemID:         3,
		Mode:             "guided",
		FlightStatus:     "active",
		LinkStatus:       "online",
		BatteryRemaining: 21,
		LatitudeDeg:      38.89,
		LongitudeDeg:     -77.04,
		LastSeen:         now,
	}
	store.ApplyProjectorSnapshot([]projector.VehicleStatePayload{vehicle}, nil)
	store.RecordGraphWrite(vehicle.ID, 88, time.Millisecond)

	trace, err := rules.NewTracePayload(rules.EvaluationInput{
		NodeID:      "boat-alpha",
		VehicleID:   vehicle.ID,
		EvaluatedAt: now,
		Facts: []rules.InputFact{{
			Scope:      rules.InputScopeLocal,
			Subject:    vehicle.ID,
			Predicate:  projector.PredicateBatteryRemainingPct,
			Object:     21,
			Source:     "test",
			ObservedAt: now,
			Confidence: 1,
		}},
	}, rules.EvaluationResult{
		RuleID:          "low-battery-peer-hold",
		RuleVersion:     "v1",
		TargetEntity:    vehicle.ID,
		Decision:        "suggest-hold-for-peer-coverage",
		SuggestedAction: "hold-position",
	})
	if err != nil {
		t.Fatalf("NewTracePayload() error = %v", err)
	}
	store.RecordRuleTrace(trace)

	block, _ := commandgate.DefaultHardwareTransmitBlocker().Check(commandgate.HardwareTransmitRequest{
		RuntimeMode:    commandgate.RuntimeModeHardware,
		SafetyProfile:  "hardware-unapproved",
		TargetEntity:   vehicle.ID,
		Verb:           commandgate.VerbRequestAutopilotVersion,
		RequestedBy:    "operator",
		TargetSystemID: 3,
	})
	store.RecordCommandGateResult(commandgate.TransmitResult{
		Status:        block.Status,
		StartedAt:     now,
		HardwareBlock: &block,
	})
	store.AddCommand(CommandView{
		EntityID:      "c360.semlink.robotics.fleet.command.hold-uav-003-1",
		TargetEntity:  vehicle.ID,
		Verb:          "hold",
		Status:        "requested",
		RequestedAt:   now,
		GraphRevision: 89,
	})

	index := mesh.NewSummaryIndex()
	item, err := mesh.NewItem(mesh.Envelope{
		OriginNodeID:    "boat-alpha",
		OriginVehicleID: vehicle.ID,
		EntityID:        vehicle.ID,
		PredicateGroup:  "vehicle.status",
		OriginSequence:  7,
		OperationID:     "op-7",
		ObservedAt:      now,
		ExpiresAt:       now.Add(time.Minute),
		SourceKind:      mesh.SourceKindMAVLinkProjection,
		Confidence:      0.95,
		MergePolicy:     mesh.MergePolicyLastWriterWins,
	}, map[string]any{"battery_remaining": 21})
	if err != nil {
		t.Fatalf("NewItem() error = %v", err)
	}
	if err := index.Upsert(item); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	server := NewServer(store, nil, "", ServerOptions{
		NodeID:    "boat-alpha",
		MeshIndex: index,
	})
	req := httptest.NewRequest(http.MethodGet, "/api/evidence", nil)
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rr.Code, rr.Body.String())
	}
	var body EvidenceBundle
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode evidence bundle: %v", err)
	}
	if body.Contract.Name != EvidenceContractName || body.Contract.Version != EvidenceContractVersion {
		t.Fatalf("contract = %#v", body.Contract)
	}
	if !containsString(body.Contract.Consumers, "SemOps GCS/COP") ||
		!containsString(body.Contract.Consumers, "semstreams-ui ops/debug") {
		t.Fatalf("consumers = %#v", body.Contract.Consumers)
	}
	if body.Contract.APIPaths["bundle"] != "/api/evidence" {
		t.Fatalf("api paths = %#v", body.Contract.APIPaths)
	}
	if body.Node.NodeID != "boat-alpha" || body.Node.Runtime != "embedded-semstreams" {
		t.Fatalf("node = %#v", body.Node)
	}
	if len(body.Vehicles) != 1 || body.Vehicles[0].EvidenceClass != "mavlink-current-state" {
		t.Fatalf("vehicles = %#v", body.Vehicles)
	}
	if body.Mesh.Status != "configured" || body.Mesh.SummaryCount != 1 || body.Mesh.WatermarkCount != 1 {
		t.Fatalf("mesh = %#v", body.Mesh)
	}
	if body.Mesh.RawMAVLinkReplicatesByDefault {
		t.Fatalf("raw MAVLink should not replicate by default")
	}
	if len(body.RuleTraces) != 1 || body.RuleTraces[0].InputCount != 1 {
		t.Fatalf("rule traces = %#v", body.RuleTraces)
	}
	if len(body.Commands) != 2 {
		t.Fatalf("commands = %#v", body.Commands)
	}
	if body.Commands[1].Gate == nil || body.Commands[1].Gate.HardwareBlock == nil {
		t.Fatalf("command gate evidence missing hardware block: %#v", body.Commands)
	}
}

func TestHandleEvidenceRejectsNonGET(t *testing.T) {
	server := NewServer(NewStore("nats://demo", true), nil, "", ServerOptions{})
	req := httptest.NewRequest(http.MethodPost, "/api/evidence", nil)
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
