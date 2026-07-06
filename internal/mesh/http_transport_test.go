package mesh

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHTTPTransportPullFromPeerAppliesOnlyMissingSummaries(t *testing.T) {
	observed := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	peerIndex := NewSummaryIndex()
	localIndex := NewSummaryIndex()

	known := mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-1", 1, observed, map[string]any{"battery": 91})
	missing := mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-2", 2, observed.Add(time.Second), map[string]any{"battery": 84})
	upsertItems(t, peerIndex, known, missing)
	upsertItems(t, localIndex, known)

	server := newMeshHTTPServer(t, peerIndex)
	defer server.Close()
	localTransport := newMeshHTTPTransport(t, localIndex)

	result, err := localTransport.PullFrom(context.Background(), server.URL, DiffOptions{
		Now: observed.Add(2 * time.Second),
	})
	if err != nil {
		t.Fatalf("PullFrom() error = %v", err)
	}
	if got, want := result.Applied, 1; got != want {
		t.Fatalf("PullFrom() applied = %d, want %d", got, want)
	}
	if got, want := len(result.Diff.Items), 1; got != want {
		t.Fatalf("PullFrom() diff item count = %d, want %d", got, want)
	}
	if result.Diff.Items[0].Envelope.EntityID != "vehicle-2" {
		t.Fatalf("PullFrom() entity = %q, want vehicle-2", result.Diff.Items[0].Envelope.EntityID)
	}
	if got, want := localIndex.Len(), 2; got != want {
		t.Fatalf("local index len = %d, want %d", got, want)
	}
}

func TestHTTPTransportPullFromPeerProgressesBoundedCatchUp(t *testing.T) {
	observed := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	peerIndex := NewSummaryIndex()
	localIndex := NewSummaryIndex()

	for seq := uint64(1); seq <= 3; seq++ {
		entityID := "vehicle-" + strconv.FormatUint(seq, 10)
		item := mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", entityID, seq, observed.Add(time.Duration(seq)*time.Second), map[string]any{"seq": seq})
		upsertItems(t, peerIndex, item)
	}

	server := newMeshHTTPServer(t, peerIndex)
	defer server.Close()
	localTransport := newMeshHTTPTransport(t, localIndex)

	first, err := localTransport.PullFrom(context.Background(), server.URL, DiffOptions{
		Now:      observed.Add(10 * time.Second),
		MaxItems: 2,
	})
	if err != nil {
		t.Fatalf("first PullFrom() error = %v", err)
	}
	if got, want := first.Applied, 2; got != want || !first.Diff.Truncated {
		t.Fatalf("first PullFrom() applied/truncated = %d/%v, want %d/true", got, first.Diff.Truncated, want)
	}

	second, err := localTransport.PullFrom(context.Background(), server.URL, DiffOptions{
		Now:      observed.Add(10 * time.Second),
		MaxItems: 2,
	})
	if err != nil {
		t.Fatalf("second PullFrom() error = %v", err)
	}
	if got, want := second.Applied, 1; got != want || second.Diff.Truncated {
		t.Fatalf("second PullFrom() applied/truncated = %d/%v, want %d/false", got, second.Diff.Truncated, want)
	}
	if got, want := localIndex.Len(), 3; got != want {
		t.Fatalf("local index len = %d, want %d", got, want)
	}
}

func TestHTTPTransportWatermarksEndpointIsInspectable(t *testing.T) {
	observed := time.Date(2026, 7, 6, 12, 0, 0, 0, time.UTC)
	peerIndex := NewSummaryIndex()
	upsertItems(t, peerIndex, mustMeshItem(t, "boat-001", "sys-001", "vehicle.telemetry.current", "vehicle-1", 1, observed, map[string]any{"battery": 91}))

	server := newMeshHTTPServer(t, peerIndex)
	defer server.Close()

	resp, err := http.Get(server.URL + HTTPWatermarksPath + "?now=" + observed.Add(time.Second).Format(time.RFC3339Nano))
	if err != nil {
		t.Fatalf("GET watermarks error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET watermarks status = %s, want 200", resp.Status)
	}
	var watermarks WatermarkSet
	if err := json.NewDecoder(resp.Body).Decode(&watermarks); err != nil {
		t.Fatalf("Decode watermarks error = %v", err)
	}
	if got, want := len(watermarks.Entries), 1; got != want {
		t.Fatalf("watermark count = %d, want %d", got, want)
	}
}

func TestHTTPTransportRejectsInvalidDiffRequest(t *testing.T) {
	peerIndex := NewSummaryIndex()
	server := newMeshHTTPServer(t, peerIndex)
	defer server.Close()

	resp, err := http.Post(server.URL+HTTPDiffPath, "application/json", bytes.NewBufferString(`{"watermarks":{"entries":[{"origin_node_id":"boat-001"}]}}`))
	if err != nil {
		t.Fatalf("POST invalid diff error = %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST invalid diff status = %s, want 400", resp.Status)
	}
}

func TestHTTPTransportRequiresAbsolutePeerURL(t *testing.T) {
	localTransport := newMeshHTTPTransport(t, NewSummaryIndex())
	_, err := localTransport.PullFrom(context.Background(), "localhost:8080", DiffOptions{})
	if err == nil {
		t.Fatal("PullFrom() error = nil, want invalid peer URL error")
	}
	if !strings.Contains(err.Error(), "absolute") {
		t.Fatalf("PullFrom() error = %v", err)
	}
}

func newMeshHTTPTransport(t *testing.T, index *SummaryIndex) *HTTPTransport {
	t.Helper()

	transport, err := NewHTTPTransport(index, HTTPTransportConfig{})
	if err != nil {
		t.Fatalf("NewHTTPTransport() error = %v", err)
	}
	return transport
}

func newMeshHTTPServer(t *testing.T, index *SummaryIndex) *httptest.Server {
	t.Helper()

	return httptest.NewServer(newMeshHTTPTransport(t, index).Handler())
}

func upsertItems(t *testing.T, index *SummaryIndex, items ...Item) {
	t.Helper()

	for _, item := range items {
		if err := index.Upsert(item); err != nil {
			t.Fatalf("Upsert() error = %v", err)
		}
	}
}
