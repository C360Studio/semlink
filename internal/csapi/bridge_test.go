package csapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/cop"
	"github.com/c360studio/semlink/internal/gcs"
)

type staticStore struct {
	snapshot gcs.Snapshot
}

func (s staticStore) Snapshot() gcs.Snapshot {
	return s.snapshot
}

type recordedRequest struct {
	Path        string
	ContentType string
	Body        map[string]any
}

type recordingCSAPI struct {
	mu       sync.Mutex
	requests []recordedRequest
}

func (r *recordingCSAPI) handler(w http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()
	var body map[string]any
	_ = json.NewDecoder(req.Body).Decode(&body)

	r.mu.Lock()
	r.requests = append(r.requests, recordedRequest{
		Path:        req.URL.Path,
		ContentType: req.Header.Get("Content-Type"),
		Body:        body,
	})
	r.mu.Unlock()

	id := idForRequest(req.URL.Path, body)
	w.Header().Set("Content-Type", string(mediaJSON))
	w.Header().Set("Location", req.URL.Path+"/"+id)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(map[string]any{"id": id})
}

func (r *recordingCSAPI) countPrefix(prefix string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, req := range r.requests {
		if strings.HasPrefix(req.Path, prefix) {
			count++
		}
	}
	return count
}

func (r *recordingCSAPI) countPath(path string) int {
	r.mu.Lock()
	defer r.mu.Unlock()
	count := 0
	for _, req := range r.requests {
		if req.Path == path {
			count++
		}
	}
	return count
}

func (r *recordingCSAPI) total() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.requests)
}

func idForRequest(path string, body map[string]any) string {
	if id, _ := body["id"].(string); id != "" {
		return id
	}
	if path == "/systems" {
		if props, _ := body["properties"].(map[string]any); props != nil {
			if uid, _ := props["uid"].(string); uid != "" {
				return "c360.semconnect.systems.csapi.system." + safeToken(uid)
			}
		}
		return "c360.semconnect.systems.csapi.system.unknown"
	}
	if path == "/samplingFeatures" {
		if props, _ := body["properties"].(map[string]any); props != nil {
			if uid, _ := props["uid"].(string); uid != "" {
				return "c360.semconnect.systems.csapi.samplingfeature." + safeToken(uid)
			}
		}
		return "c360.semconnect.systems.csapi.samplingfeature.unknown"
	}
	if strings.Contains(path, "/observations") {
		return "observation-001"
	}
	return "resource-001"
}

func TestBridgeSyncPublishesCuratedCSAPIProjection(t *testing.T) {
	now := time.Date(2026, 6, 14, 15, 4, 5, 0, time.UTC)
	operatorID, err := cop.EntityID(cop.KindOperator, "ANDROID-1")
	if err != nil {
		t.Fatalf("operator id: %v", err)
	}
	markerID, err := cop.EntityID(cop.KindMarker, "marker-1")
	if err != nil {
		t.Fatalf("marker id: %v", err)
	}
	messageID, err := cop.EntityID(cop.KindMessage, "chat-1")
	if err != nil {
		t.Fatalf("message id: %v", err)
	}
	store := staticStore{snapshot: gcs.Snapshot{
		GeneratedAt: now,
		Vehicles: []gcs.VehicleView{{
			EntityID:         "c360.semlink.robotics.fleet.drone.uav-001",
			Callsign:         "UAV-001",
			SystemID:         1,
			BatteryRemaining: 18,
			LatitudeDeg:      38.9,
			LongitudeDeg:     -77.03,
			AltitudeM:        121.5,
			GroundSpeedMS:    6.2,
			LastSeen:         now,
			GraphRevision:    42,
			IndexingProfile:  "signal",
		}},
		Alerts: []gcs.AlertView{{
			EntityID:      "c360.semlink.robotics.fleet.alert.low-battery-uav-001",
			Kind:          "low-battery",
			Severity:      "warning",
			Active:        true,
			SubjectEntity: "c360.semlink.robotics.fleet.drone.uav-001",
			Message:       "UAV-001 battery is below 20%",
			RaisedAt:      now,
			GraphRevision: 43,
		}},
		Commands: []gcs.CommandView{{
			EntityID:      "c360.semlink.robotics.fleet.command.return-uav-001-1781468645000",
			TargetEntity:  "c360.semlink.robotics.fleet.drone.uav-001",
			Verb:          "return",
			Status:        "requested",
			RequestedAt:   now,
			GraphRevision: 44,
		}},
		Operators: []cop.View{{
			Kind:            cop.KindOperator,
			EntityID:        operatorID,
			UID:             "ANDROID-1",
			Callsign:        "ALPHA",
			LatitudeDeg:     38.91,
			LongitudeDeg:    -77.04,
			AltitudeM:       20,
			HasPosition:     true,
			LastSeen:        now,
			GraphRevision:   45,
			IndexingProfile: "signal",
		}},
		Markers: []cop.View{{
			Kind:            cop.KindMarker,
			EntityID:        markerID,
			UID:             "marker-1",
			Label:           "Checkpoint",
			Description:     "north gate",
			LatitudeDeg:     38.92,
			LongitudeDeg:    -77.05,
			HasPosition:     true,
			LastSeen:        now,
			GraphRevision:   46,
			IndexingProfile: "content",
		}},
		Messages: []cop.View{{
			Kind:            cop.KindMessage,
			EntityID:        messageID,
			UID:             "chat-1",
			Callsign:        "ALPHA",
			Text:            "hold at checkpoint",
			SenderUID:       "ANDROID-1",
			SenderEntity:    operatorID,
			LastSeen:        now,
			GraphRevision:   47,
			IndexingProfile: "content",
		}},
	}}

	recorder := &recordingCSAPI{}
	server := httptest.NewServer(http.HandlerFunc(recorder.handler))
	defer server.Close()

	bridge, err := NewBridge(Config{
		BaseURL:             server.URL,
		ObservationInterval: 5 * time.Second,
		HTTPClient:          server.Client(),
	}, store)
	if err != nil {
		t.Fatalf("NewBridge: %v", err)
	}

	if err := bridge.Sync(context.Background()); err != nil {
		t.Fatalf("Sync: %v", err)
	}

	if got := recorder.countPath("/systems"); got != 2 {
		t.Fatalf("POST /systems count = %d, want 2", got)
	}
	if got := recorder.countPath("/datastreams"); got != 4 {
		t.Fatalf("POST /datastreams count = %d, want 4", got)
	}
	if got := recorder.countPrefix("/datastreams/c360.semlink.robotics.csapi.datastream."); got != 3 {
		t.Fatalf("observation count = %d, want 3", got)
	}
	if got := recorder.countPrefix("/datastreams/c360.semlink.cop.csapi.datastream."); got != 1 {
		t.Fatalf("operator observation count = %d, want 1", got)
	}
	if got := recorder.countPath("/systems/c360.semconnect.systems.csapi.system.uav-001/events"); got != 1 {
		t.Fatalf("system event count = %d, want 1", got)
	}
	if got := recorder.countPath("/samplingFeatures"); got != 1 {
		t.Fatalf("sampling feature count = %d, want 1", got)
	}
	if got := recorder.countPath("/systems/c360.semconnect.systems.csapi.system.android-1/events"); got != 1 {
		t.Fatalf("geochat system event count = %d, want 1", got)
	}
	if got := recorder.countPath("/controlstreams"); got != 1 {
		t.Fatalf("controlstream count = %d, want 1", got)
	}
	if got := recorder.countPath("/commands"); got != 1 {
		t.Fatalf("command count = %d, want 1", got)
	}

	firstTotal := recorder.total()
	if err := bridge.Sync(context.Background()); err != nil {
		t.Fatalf("second Sync: %v", err)
	}
	if got := recorder.total(); got != firstTotal {
		t.Fatalf("second Sync posted duplicates: got total %d want %d", got, firstTotal)
	}
}

func TestDecodePostResultAcceptsConflictAttemptedID(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusConflict,
		Header:     http.Header{"X-Cs-Attempted-Id": []string{"c360.semconnect.systems.csapi.system.uav-001"}},
		Body:       http.NoBody,
	}
	result, err := decodePostResult(resp, "", nil)
	if err != nil {
		t.Fatalf("decodePostResult: %v", err)
	}
	if result.ID != "c360.semconnect.systems.csapi.system.uav-001" {
		t.Fatalf("ID = %q", result.ID)
	}
}

func TestDecodePostResultInfersConflictIDFromRequestBody(t *testing.T) {
	resp := &http.Response{
		StatusCode: http.StatusConflict,
		Body:       http.NoBody,
	}
	result, err := decodePostResult(resp, "/systems", map[string]any{
		"properties": map[string]any{"uid": "ANDROID-ALPHA"},
	})
	if err != nil {
		t.Fatalf("decodePostResult: %v", err)
	}
	if result.ID != "c360.semconnect.systems.csapi.system.ANDROID-ALPHA" {
		t.Fatalf("ID = %q", result.ID)
	}
}
