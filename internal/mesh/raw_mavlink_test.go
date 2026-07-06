package mesh

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/mavlink"
)

func TestSummaryIndexRejectsRawMAVLinkFramesByDefault(t *testing.T) {
	observed := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	raw := mustRawMAVLinkItem(t, observed)
	index := NewSummaryIndex()

	err := index.Upsert(raw)
	if err == nil {
		t.Fatal("Upsert(raw MAVLink) error = nil, want default non-replication error")
	}
	if !strings.Contains(err.Error(), "does not replicate over mesh by default") {
		t.Fatalf("Upsert(raw MAVLink) error = %v", err)
	}
	if got := index.Len(); got != 0 {
		t.Fatalf("index len after rejected raw MAVLink = %d, want 0", got)
	}
	if got := len(index.Watermarks(observed).Entries); got != 0 {
		t.Fatalf("watermarks after rejected raw MAVLink = %d, want 0", got)
	}
}

func TestHTTPTransportRejectsRawMAVLinkDiffItemsByDefault(t *testing.T) {
	observed := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	raw := mustRawMAVLinkItem(t, observed)
	localIndex := NewSummaryIndex()
	localTransport := newMeshHTTPTransport(t, localIndex)
	peer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != HTTPDiffPath {
			http.NotFound(w, r)
			return
		}
		writeHTTPJSON(w, Diff{Items: []Item{raw}})
	}))
	defer peer.Close()

	_, err := localTransport.PullFrom(context.Background(), peer.URL, DiffOptions{
		Now: observed.Add(time.Second),
	})
	if err == nil {
		t.Fatal("PullFrom(raw MAVLink diff) error = nil, want default non-replication error")
	}
	if !strings.Contains(err.Error(), "does not replicate over mesh by default") {
		t.Fatalf("PullFrom(raw MAVLink diff) error = %v", err)
	}
	if got := localIndex.Len(); got != 0 {
		t.Fatalf("local index len after rejected raw MAVLink diff = %d, want 0", got)
	}
}

func mustRawMAVLinkItem(t *testing.T, observed time.Time) Item {
	t.Helper()

	frame, err := mavlink.EncodeV2(1, 42, 1, mavlink.MessageHeartbeat, mavlink.HeartbeatPayload(true, 0))
	if err != nil {
		t.Fatalf("EncodeV2() error = %v", err)
	}
	item, err := NewItem(Envelope{
		OriginNodeID:    "boat-042",
		OriginVehicleID: "sys-042",
		EntityID:        "mavlink.raw.boat-042:1",
		PredicateGroup:  "mavlink.raw",
		OriginSequence:  1,
		OperationID:     "boat-042:mavlink.raw:1",
		ObservedAt:      observed,
		ExpiresAt:       observed.Add(time.Second),
		SourceKind:      SourceKindRawMAVLink,
		Confidence:      1,
		MergePolicy:     MergePolicyAppendLimited,
	}, mavlink.RawFrame{
		Subject:   "mavlink.raw.boat-042",
		VehicleID: "boat-042",
		SystemID:  42,
		EmittedAt: observed,
		Bytes:     frame,
	})
	if err != nil {
		t.Fatalf("NewItem(raw MAVLink) error = %v", err)
	}
	return item
}
