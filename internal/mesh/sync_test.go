package mesh

import (
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSummaryIndexWatermarksAreCompactByOriginAndStateClass(t *testing.T) {
	observed := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	index := NewSummaryIndex()
	for _, item := range []Item{
		mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-1", 1, observed, map[string]any{"battery": 91}),
		mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-2", 2, observed.Add(time.Second), map[string]any{"battery": 84}),
		mustMeshItem(t, "boat-001", "sys-001", "alert.low-battery", "alert-1", 3, observed.Add(2*time.Second), map[string]any{"severity": "warning"}),
	} {
		if err := index.Upsert(item); err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}
	}

	watermarks := index.Watermarks(observed.Add(3 * time.Second))
	if got, want := len(watermarks.Entries), 2; got != want {
		t.Fatalf("Watermarks() entry count = %d, want %d", got, want)
	}

	telemetry, ok := watermarks.Get(CellID{
		OriginNodeID:    "boat-001",
		OriginVehicleID: "sys-001",
		PredicateGroup:  "vehicle.telemetry.current",
	})
	if !ok {
		t.Fatal("Watermarks() missing telemetry cell")
	}
	if telemetry.OriginSequence != 2 {
		t.Fatalf("telemetry OriginSequence = %d, want 2", telemetry.OriginSequence)
	}
	if telemetry.ItemCount != 2 {
		t.Fatalf("telemetry ItemCount = %d, want 2", telemetry.ItemCount)
	}
	if !strings.HasPrefix(telemetry.CellHash, "sha256:") {
		t.Fatalf("telemetry CellHash = %q", telemetry.CellHash)
	}
}

func TestSummaryIndexReplacesCurrentStateInsteadOfKeepingLedger(t *testing.T) {
	observed := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	index := NewSummaryIndex()
	first := mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-1", 1, observed, map[string]any{"battery": 91})
	second := mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-1", 2, observed.Add(time.Second), map[string]any{"battery": 90})

	if err := index.Upsert(first); err != nil {
		t.Fatalf("Upsert(first) error = %v", err)
	}
	if err := index.Upsert(second); err != nil {
		t.Fatalf("Upsert(second) error = %v", err)
	}
	if got, want := index.Len(), 1; got != want {
		t.Fatalf("Len() = %d, want %d", got, want)
	}

	diff, err := index.Diff(WatermarkSet{Entries: []CellWatermark{watermarkFromItem(first)}}, DiffOptions{
		Now: observed.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}
	if got, want := len(diff.Items), 1; got != want {
		t.Fatalf("Diff() item count = %d, want %d", got, want)
	}
	if diff.Items[0].Envelope.OriginSequence != 2 {
		t.Fatalf("diff sequence = %d, want 2", diff.Items[0].Envelope.OriginSequence)
	}
}

func TestSummaryIndexDiffSkipsCellsCoveredByPeerWatermarks(t *testing.T) {
	observed := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	index := NewSummaryIndex()
	boatOne := mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-1", 1, observed, map[string]any{"battery": 91})
	boatTwo := mustMeshItem(t, "boat-002", "sys-002", "vehicle.telemetry.current", "vehicle-2", 1, observed, map[string]any{"battery": 87})
	for _, item := range []Item{boatOne, boatTwo} {
		if err := index.Upsert(item); err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}
	}

	localWatermarks := index.Watermarks(observed.Add(time.Second))
	boatOneWatermark, ok := localWatermarks.Get(boatOne.CellID())
	if !ok {
		t.Fatal("missing boat one watermark")
	}

	diff, err := index.Diff(WatermarkSet{Entries: []CellWatermark{boatOneWatermark}}, DiffOptions{
		Now: observed.Add(time.Second),
	})
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}
	if got, want := len(diff.Items), 1; got != want {
		t.Fatalf("Diff() item count = %d, want %d", got, want)
	}
	if diff.Items[0].Envelope.OriginNodeID != "boat-002" {
		t.Fatalf("Diff() sent origin %q, want boat-002", diff.Items[0].Envelope.OriginNodeID)
	}
}

func TestSummaryIndexDiffRepairsDigestMismatchWithinBound(t *testing.T) {
	observed := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	index := NewSummaryIndex()
	first := mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-1", 1, observed, map[string]any{"battery": 91})
	second := mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-2", 2, observed.Add(time.Second), map[string]any{"battery": 90})
	for _, item := range []Item{first, second} {
		if err := index.Upsert(item); err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}
	}

	localWatermark, ok := index.Watermarks(observed.Add(2 * time.Second)).Get(first.CellID())
	if !ok {
		t.Fatal("missing local watermark")
	}
	peerWatermark := localWatermark
	peerWatermark.ItemCount = 1
	peerWatermark.CellHash = "sha256:0000000000000000000000000000000000000000000000000000000000000000"

	diff, err := index.Diff(WatermarkSet{Entries: []CellWatermark{peerWatermark}}, DiffOptions{
		Now:      observed.Add(2 * time.Second),
		MaxItems: 1,
	})
	if err != nil {
		t.Fatalf("Diff() error = %v", err)
	}
	if got, want := len(diff.Items), 1; got != want {
		t.Fatalf("Diff() item count = %d, want bounded %d", got, want)
	}
	if !diff.Truncated {
		t.Fatal("Diff() Truncated = false, want true")
	}
}

func TestSummaryIndexWatermarksDropExpiredItemsWhenNowProvided(t *testing.T) {
	observed := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	index := NewSummaryIndex()
	item := mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-1", 1, observed, map[string]any{"battery": 91})
	if err := index.Upsert(item); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}

	watermarks := index.Watermarks(observed.Add(2 * time.Minute))
	if got, want := len(watermarks.Entries), 0; got != want {
		t.Fatalf("Watermarks() entry count after expiry = %d, want %d", got, want)
	}
}

func mustMeshItem(t *testing.T, originNode, originVehicle, predicateGroup, entityID string, sequence uint64, observed time.Time, payload any) Item {
	t.Helper()

	item, err := NewItem(Envelope{
		OriginNodeID:    originNode,
		OriginVehicleID: originVehicle,
		EntityID:        entityID,
		PredicateGroup:  predicateGroup,
		OriginSequence:  sequence,
		OperationID:     originNode + ":" + predicateGroup + ":" + entityID + ":" + strconv.FormatUint(sequence, 10),
		ObservedAt:      observed,
		ExpiresAt:       observed.Add(time.Minute),
		SourceKind:      SourceKindMAVLinkProjection,
		Confidence:      1,
		MergePolicy:     MergePolicyLastWriterWins,
	}, payload)
	if err != nil {
		t.Fatalf("NewItem() error = %v", err)
	}
	return item
}
