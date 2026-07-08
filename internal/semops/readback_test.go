package semops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/handoff"
	"github.com/c360studio/semlink/internal/mavlink"
)

func TestNewAutopilotVersionRequestMatchesContractFixture(t *testing.T) {
	requestedAt := mustParseTime(t, "2026-07-07T18:30:00Z")
	got := NewAutopilotVersionRequest(AutopilotVersionRequestInput{
		Profile: handoff.Profile{
			NodeID: "blue-boat-01",
		},
		TargetSystemID: 42,
		CorrelationID:  "corr-blue-boat-01-autopilot-version-001",
		IdempotencyKey: "idem-blue-boat-01-autopilot-version-001",
		RequestedAt:    requestedAt,
		TTL:            30 * time.Second,
	})
	want := loadFixture[ReadbackRequest](t, "request.accepted.json")

	if got != want {
		t.Fatalf("request mismatch\ngot:  %#v\nwant: %#v", got, want)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}
}

func TestReadbackRequestRejectsNonV0MAVLinkShape(t *testing.T) {
	request := loadFixture[ReadbackRequest](t, "request.rejected-unsupported-message.json")

	if err := request.Validate(); err == nil {
		t.Fatal("Validate accepted unsupported requested_message_id")
	}
}

func TestClientPostsReadbackRequestWithoutMintingTrustedHeaders(t *testing.T) {
	expectedRequest := loadFixture[ReadbackRequest](t, "request.accepted.json")
	expectedResponse := loadFixture[ReadbackResponse](t, "response.accepted.json")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != ReadbackPathV0 {
			t.Fatalf("path = %s, want %s", r.URL.Path, ReadbackPathV0)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Fatalf("content type = %q, want application/json", got)
		}
		if got := r.Header.Get("X-SemOps-Operator-Authenticated"); got != "" {
			t.Fatalf("trusted SemOps auth header was minted: %q", got)
		}
		var gotRequest ReadbackRequest
		decoder := json.NewDecoder(r.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&gotRequest); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if gotRequest != expectedRequest {
			t.Fatalf("posted request mismatch\ngot:  %#v\nwant: %#v", gotRequest, expectedRequest)
		}
		writeJSON(t, w, expectedResponse)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	got, err := client.SubmitReadback(context.Background(), expectedRequest)
	if err != nil {
		t.Fatalf("SubmitReadback: %v", err)
	}
	if !reflect.DeepEqual(got, expectedResponse) {
		t.Fatalf("response mismatch\ngot:  %#v\nwant: %#v", got, expectedResponse)
	}
	if got.NativeExecutionAllowed || got.CompanionTransmitAllowed {
		t.Fatalf(
			"transmit posture = native %v companion %v, want both false",
			got.NativeExecutionAllowed,
			got.CompanionTransmitAllowed,
		)
	}
}

func TestClientCarriesCallerSuppliedHeadersForGatewayOrTestHarness(t *testing.T) {
	request := loadFixture[ReadbackRequest](t, "request.accepted.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test-Gateway"); got != "local-dev" {
			t.Fatalf("X-Test-Gateway = %q, want local-dev", got)
		}
		writeJSON(t, w, loadFixture[ReadbackResponse](t, "response.accepted.json"))
	}))
	defer server.Close()

	headers := http.Header{}
	headers.Set("X-Test-Gateway", "local-dev")
	client, err := NewClient(ClientConfig{BaseURL: server.URL, Headers: headers})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	if _, err := client.SubmitReadback(context.Background(), request); err != nil {
		t.Fatalf("SubmitReadback: %v", err)
	}
}

func TestClientDecodesDuplicateAdmission(t *testing.T) {
	request := loadFixture[ReadbackRequest](t, "request.accepted.json")
	expected := loadFixture[ReadbackResponse](t, "response.duplicate.json")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, expected)
	}))
	defer server.Close()

	client, err := NewClient(ClientConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	got, err := client.SubmitReadback(context.Background(), request)
	if err != nil {
		t.Fatalf("SubmitReadback: %v", err)
	}
	if !reflect.DeepEqual(got, expected) || !got.Duplicate || got.Mutations != 0 {
		t.Fatalf("duplicate response = %#v, want %#v", got, expected)
	}
}

func TestRequestUsesMAVLinkConstants(t *testing.T) {
	request := loadFixture[ReadbackRequest](t, "request.accepted.json")

	if request.CommandID != mavlink.MAVCmdRequestMessage {
		t.Fatalf("command_id = %d, want %d", request.CommandID, mavlink.MAVCmdRequestMessage)
	}
	if request.RequestedMessageID != uint32(mavlink.MessageAutopilotVersion) {
		t.Fatalf("requested_message_id = %d, want %d", request.RequestedMessageID, mavlink.MessageAutopilotVersion)
	}
}

func TestCSAPIProjectionFixturePreservesNativeContractFacts(t *testing.T) {
	request := loadFixture[ReadbackRequest](t, "request.accepted.json")
	projection := loadFixture[csapiProjectionFixture](t, "csapi-projection.accepted.json")

	if projection.Contract != "c360.semops.semlink.csapi-projection.v0" ||
		projection.SourceContract != ReadbackContractV0 ||
		projection.ClaimScope != "csapi-projection-fixture-only" ||
		projection.CSAPIScore != "amber" {
		t.Fatalf("projection header = %+v", projection)
	}
	companion := projection.systemByRole("companion")
	if companion.ID == "" || companion.SourceRef != "semlink://blue-boat-01" {
		t.Fatalf("companion system = %+v", companion)
	}
	target := projection.systemByRole("target")
	if target.ID != "c360.edge.cop.mavlink.asset.system-42" ||
		target.MAVLink.TargetSystemID != int(request.TargetSystemID) ||
		target.MAVLink.TargetComponentID != int(request.TargetComponentID) {
		t.Fatalf("target system = %+v", target)
	}
	if projection.Command.Status != "accepted" ||
		projection.Command.CorrelationID != request.CorrelationID ||
		projection.Command.IdempotencyKey != request.IdempotencyKey ||
		projection.Command.SourceRef != request.SourceRef {
		t.Fatalf("command projection = %+v", projection.Command)
	}
	desired := projection.Command.Desired
	if desired.CommandID != request.CommandID ||
		desired.RequestedMessageID != request.RequestedMessageID ||
		desired.TargetSystemID != int(request.TargetSystemID) ||
		desired.TargetComponentID != int(request.TargetComponentID) ||
		desired.Command != "MAV_CMD_REQUEST_MESSAGE" ||
		desired.Message != "AUTOPILOT_VERSION" {
		t.Fatalf("desired projection = %+v", desired)
	}
	governance := projection.Command.Governance
	if governance.AuthorityScope != "semlink.readback.intent" ||
		governance.ClaimScope != "semlink-companion-command-intent-only" ||
		governance.NativeExecutionAllowed ||
		governance.CompanionTransmitAllowed ||
		governance.Duplicate ||
		governance.Mutations != 1 {
		t.Fatalf("governance projection = %+v", governance)
	}
	if len(projection.Observations) != 1 ||
		projection.Observations[0].ObservedProperty != "AUTOPILOT_VERSION" ||
		!projection.Observations[0].Deferred {
		t.Fatalf("observation projection = %+v", projection.Observations)
	}
}

type csapiProjectionFixture struct {
	Contract       string                 `json:"contract"`
	SourceContract string                 `json:"source_contract"`
	ClaimScope     string                 `json:"claim_scope"`
	CSAPIScore     string                 `json:"csapi_score"`
	Systems        []projectedSystem      `json:"systems"`
	Command        projectedCommand       `json:"command"`
	Observations   []projectedObservation `json:"observations"`
}

func (p csapiProjectionFixture) systemByRole(role string) projectedSystem {
	for _, system := range p.Systems {
		if system.Role == role {
			return system
		}
	}
	return projectedSystem{}
}

type projectedSystem struct {
	Role      string `json:"role"`
	ID        string `json:"id"`
	SourceRef string `json:"source_ref"`
	MAVLink   struct {
		TargetSystemID    int `json:"target_system_id"`
		TargetComponentID int `json:"target_component_id"`
	} `json:"mavlink"`
}

type projectedCommand struct {
	Status         string `json:"status"`
	CorrelationID  string `json:"correlation_id"`
	IdempotencyKey string `json:"idempotency_key"`
	SourceRef      string `json:"source_ref"`
	Desired        struct {
		Command            string `json:"command"`
		Message            string `json:"message"`
		CommandID          uint16 `json:"command_id"`
		RequestedMessageID uint32 `json:"requested_message_id"`
		TargetSystemID     int    `json:"target_system_id"`
		TargetComponentID  int    `json:"target_component_id"`
	} `json:"desired"`
	Governance struct {
		AuthorityScope           string `json:"authority_scope"`
		ClaimScope               string `json:"claim_scope"`
		NativeExecutionAllowed   bool   `json:"native_execution_allowed"`
		CompanionTransmitAllowed bool   `json:"companion_transmit_allowed"`
		Duplicate                bool   `json:"duplicate"`
		Mutations                int    `json:"mutations"`
	} `json:"governance"`
}

type projectedObservation struct {
	ObservedProperty string `json:"observed_property"`
	Deferred         bool   `json:"deferred"`
}

func loadFixture[T any](t *testing.T, name string) T {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "contracts", "semlink-companion-readback-v0", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	var out T
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode fixture %s: %v", name, err)
	}
	return out
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write response: %v", err)
	}
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time %q: %v", value, err)
	}
	return parsed
}
