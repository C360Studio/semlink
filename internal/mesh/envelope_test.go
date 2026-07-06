package mesh

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewItemAddsEnvelopeMetadataAndPayloadHash(t *testing.T) {
	observed := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	payload := map[string]any{
		"battery_remaining": 87,
		"link_status":       "online",
	}
	item, err := NewItem(Envelope{
		OriginNodeID:    "boat-001",
		OriginVehicleID: "sys-042",
		EntityID:        "c360.semlink.robotics.fleet.drone.uav-042",
		PredicateGroup:  "vehicle.telemetry.current",
		OriginSequence:  17,
		OperationID:     "boat-001:17",
		ObservedAt:      observed,
		ExpiresAt:       observed.Add(5 * time.Second),
		SourceKind:      SourceKindMAVLinkProjection,
		Confidence:      0.97,
		MergePolicy:     MergePolicyLastWriterWins,
	}, payload)
	if err != nil {
		t.Fatalf("NewItem() error = %v", err)
	}

	if item.Envelope.PayloadHash == "" || !strings.HasPrefix(item.Envelope.PayloadHash, "sha256:") {
		t.Fatalf("PayloadHash = %q", item.Envelope.PayloadHash)
	}
	if item.Envelope.OriginSequence != 17 {
		t.Fatalf("OriginSequence = %d", item.Envelope.OriginSequence)
	}
	if item.Envelope.MergePolicy != MergePolicyLastWriterWins {
		t.Fatalf("MergePolicy = %q", item.Envelope.MergePolicy)
	}

	raw, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	for _, field := range []string{
		"origin_node_id",
		"origin_vehicle_id",
		"entity_id",
		"predicate_group",
		"origin_sequence",
		"operation_id",
		"observed_at",
		"expires_at",
		"source_kind",
		"confidence",
		"merge_policy",
		"payload_hash",
	} {
		if !strings.Contains(string(raw), field) {
			t.Fatalf("marshaled item missing %q: %s", field, raw)
		}
	}
}

func TestEnvelopeAllowsHybridLogicalTimeOrdering(t *testing.T) {
	observed := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	item, err := NewItem(Envelope{
		OriginNodeID:    "boat-001",
		OriginVehicleID: "sys-042",
		EntityID:        "c360.semlink.robotics.fleet.alert.low-battery-uav-042",
		PredicateGroup:  "alert.low-battery",
		HybridTime:      &HybridLogicalTime{WallTime: observed, Counter: 3},
		OperationID:     "boat-001:hlc:3",
		ObservedAt:      observed,
		ExpiresAt:       observed.Add(time.Minute),
		SourceKind:      SourceKindPeerSummary,
		Confidence:      1,
		MergePolicy:     MergePolicySetUnion,
	}, map[string]string{"active": "true"})
	if err != nil {
		t.Fatalf("NewItem() error = %v", err)
	}
	if item.Envelope.HybridTime == nil || item.Envelope.HybridTime.Counter != 3 {
		t.Fatalf("HybridTime = %#v", item.Envelope.HybridTime)
	}
}

func TestSemStreamsEvidenceDoesNotSatisfyDistributedOrdering(t *testing.T) {
	observed := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	_, err := NewItem(Envelope{
		OriginNodeID:    "boat-001",
		OriginVehicleID: "sys-042",
		EntityID:        "c360.semlink.robotics.fleet.drone.uav-042",
		PredicateGroup:  "vehicle.telemetry.current",
		OperationID:     "entity-revision-44",
		ObservedAt:      observed,
		ExpiresAt:       observed.Add(5 * time.Second),
		SourceKind:      SourceKindMAVLinkProjection,
		Confidence:      1,
		MergePolicy:     MergePolicyLastWriterWins,
		SemStreams: &SemStreamsEvidence{
			KVRevision:      44,
			EntityVersion:   7,
			EntityUpdatedAt: observed,
			TripleTimestamp: observed,
		},
	}, map[string]string{"link_status": "online"})
	if err == nil {
		t.Fatal("NewItem() error = nil, want missing distributed ordering error")
	}
	if !strings.Contains(err.Error(), "origin sequence or hybrid logical time") {
		t.Fatalf("NewItem() error = %v", err)
	}
}

func TestItemRejectsPayloadHashMismatch(t *testing.T) {
	observed := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	_, err := NewItem(Envelope{
		OriginNodeID:    "boat-001",
		OriginVehicleID: "sys-042",
		EntityID:        "c360.semlink.robotics.fleet.drone.uav-042",
		PredicateGroup:  "vehicle.telemetry.current",
		OriginSequence:  18,
		OperationID:     "boat-001:18",
		ObservedAt:      observed,
		ExpiresAt:       observed.Add(5 * time.Second),
		SourceKind:      SourceKindMAVLinkProjection,
		Confidence:      1,
		MergePolicy:     MergePolicyLastWriterWins,
		PayloadHash:     "sha256:0000000000000000000000000000000000000000000000000000000000000000",
	}, map[string]string{"link_status": "online"})
	if err == nil {
		t.Fatal("NewItem() error = nil, want payload hash mismatch")
	}
	if !strings.Contains(err.Error(), "payload hash mismatch") {
		t.Fatalf("NewItem() error = %v", err)
	}
}
