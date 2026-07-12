package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/c360studio/semlink/internal/commandgate"
	"github.com/c360studio/semlink/internal/companion"
	"github.com/c360studio/semlink/internal/gcs"
	"github.com/c360studio/semlink/internal/handoff"
	"github.com/c360studio/semlink/internal/mavlink"
	"github.com/c360studio/semlink/internal/mesh"
	"github.com/c360studio/semlink/internal/semops"
)

const (
	FastCompanionReportKind = "fast-companion-e2e"
	defaultVehicleProfile   = "mavlink-generic"
	defaultNodeIDPrefix     = "vehicle-node"
)

type FastCompanionConfig struct {
	Nodes          int
	VehicleProfile string
	Start          time.Time
	StepElapsed    time.Duration
}

type FastCompanionReport struct {
	Kind             string                 `json:"kind"`
	GeneratedAt      time.Time              `json:"generated_at"`
	VehicleProfile   string                 `json:"vehicle_profile"`
	Nodes            []NodeReport           `json:"nodes"`
	EvidencePresent  bool                   `json:"evidence_present"`
	Evidence         gcs.EvidenceBundle     `json:"evidence"`
	CommandPosture   CommandPostureReport   `json:"command_posture"`
	SimulatorCommand SimulatorCommandReport `json:"simulator_command"`
	SemOpsReadback   SemOpsReadbackReport   `json:"semops_readback"`
	Assertions       []Assertion            `json:"assertions"`
}

type NodeReport struct {
	NodeID        string `json:"node_id"`
	VehicleCount  int    `json:"vehicle_count"`
	AlertCount    int    `json:"alert_count"`
	FrameCount    int    `json:"frame_count"`
	DecodedCount  int    `json:"decoded_count"`
	GraphEntities int    `json:"graph_entities"`
}

type CommandPostureReport struct {
	HTTPStatus      int    `json:"http_status"`
	Status          string `json:"status"`
	HardwareBlocked bool   `json:"hardware_blocked"`
}

type SimulatorCommandReport struct {
	Accepted                   bool   `json:"accepted"`
	Status                     string `json:"status"`
	PreflightAccepted          bool   `json:"preflight_accepted"`
	SimulatorOnly              bool   `json:"simulator_only"`
	FrameCount                 int    `json:"frame_count"`
	ACKCount                   int    `json:"ack_count"`
	ACKAccepted                bool   `json:"ack_accepted"`
	PostStateObserved          bool   `json:"post_state_observed"`
	HardwareTransmitAuthorized bool   `json:"hardware_transmit_authorized"`
}

type SemOpsReadbackReport struct {
	Request          semops.ReadbackRequest  `json:"request"`
	Response         semops.ReadbackResponse `json:"response"`
	ReceivedRequests int                     `json:"received_requests"`
}

type Assertion struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail,omitempty"`
}

func RunFastCompanionE2E(ctx context.Context, cfg FastCompanionConfig) (FastCompanionReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg = cfg.withDefaults()

	harness, err := companion.NewHarness(companion.HarnessConfig{
		Nodes:        cfg.Nodes,
		NodeIDPrefix: defaultNodeIDPrefix,
		Start:        cfg.Start,
	})
	if err != nil {
		return FastCompanionReport{}, err
	}
	step, err := harness.Step(ctx, cfg.StepElapsed)
	if err != nil {
		return FastCompanionReport{}, err
	}

	report := FastCompanionReport{
		Kind:           FastCompanionReportKind,
		GeneratedAt:    step.At,
		VehicleProfile: cfg.VehicleProfile,
		Nodes:          nodeReports(step.Nodes),
	}
	report.addAssertion("deterministic-companion-nodes", len(step.Nodes) == cfg.Nodes, fmt.Sprintf("nodes=%d", len(step.Nodes)))
	report.addAssertion("projected-local-state", allNodesHaveProjectedState(step.Nodes), "each node has vehicle state and graph entities")

	if len(step.Nodes) == 0 || len(step.Nodes[0].Vehicles) == 0 {
		report.addAssertion("evidence-compatible-state", false, "first node has no vehicle state")
		return report, nil
	}

	first := step.Nodes[0]
	profile := fastProfile(first, cfg)
	store := evidenceStore(first)
	index, err := evidenceMeshIndex(first, step.At, cfg.VehicleProfile)
	if err != nil {
		return FastCompanionReport{}, err
	}
	server := gcs.NewServer(store, nil, "", gcs.ServerOptions{
		NodeID:         first.NodeID,
		MeshIndex:      index,
		HandoffProfile: &profile,
	})
	report.Evidence = server.EvidenceBundle(step.At)
	report.EvidencePresent = report.Evidence.Contract.Name == gcs.EvidenceContractName
	report.addAssertion("evidence-compatible-state", report.EvidencePresent && len(report.Evidence.Vehicles) > 0, "evidence contract and vehicle state are present")
	report.addAssertion("downstream-optional-posture", downstreamOptional(report.Evidence.Downstream), "downstream consumers are optional")
	report.addAssertion("no-csapi-hot-path", semconnectDisabled(report.Evidence), "CS API URL is absent and SemConnect remains disabled")

	commandPosture, err := probeHardwareCommandBlock(server, first.Vehicles[0].ID)
	if err != nil {
		return FastCompanionReport{}, err
	}
	report.CommandPosture = commandPosture
	report.addAssertion("hardware-transmit-blocked", commandPosture.HardwareBlocked, commandPosture.Status)

	simulatorCommand, err := probeSimulatorCommandEvidence(ctx, first.Vehicles[0].ID, first.Vehicles[0].SystemID, step.At)
	if err != nil {
		return FastCompanionReport{}, err
	}
	report.SimulatorCommand = simulatorCommand
	report.addAssertion("simulator-command-evidence-distinct", simulatorCommand.SimulatorOnly && !simulatorCommand.HardwareTransmitAuthorized, simulatorCommand.Status)

	readback, err := probeSemOpsReadback(ctx, profile, first, step.At)
	if err != nil {
		return FastCompanionReport{}, err
	}
	report.SemOpsReadback = readback
	report.addAssertion("semops-readback-v0", readback.ReceivedRequests == 1 && readback.Response.Accepted, readback.Response.Status)

	return report, nil
}

func (cfg FastCompanionConfig) withDefaults() FastCompanionConfig {
	if cfg.Nodes <= 0 {
		cfg.Nodes = 2
	}
	if cfg.VehicleProfile == "" {
		cfg.VehicleProfile = defaultVehicleProfile
	}
	if cfg.Start.IsZero() {
		cfg.Start = time.Unix(0, 0).UTC()
	}
	if cfg.StepElapsed <= 0 {
		cfg.StepElapsed = 20 * time.Second
	}
	return cfg
}

func nodeReports(nodes []companion.NodeSnapshot) []NodeReport {
	out := make([]NodeReport, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, NodeReport{
			NodeID:        node.NodeID,
			VehicleCount:  len(node.Vehicles),
			AlertCount:    len(node.Alerts),
			FrameCount:    node.Frames,
			DecodedCount:  node.Decoded,
			GraphEntities: node.GraphEntities,
		})
	}
	return out
}

func allNodesHaveProjectedState(nodes []companion.NodeSnapshot) bool {
	if len(nodes) == 0 {
		return false
	}
	for _, node := range nodes {
		if node.NodeID == "" || len(node.Vehicles) == 0 || node.GraphEntities == 0 {
			return false
		}
	}
	return true
}

func fastProfile(node companion.NodeSnapshot, cfg FastCompanionConfig) handoff.Profile {
	vehicle := node.Vehicles[0]
	profile := handoff.DefaultProfile()
	profile.NodeID = node.NodeID
	profile.VehicleID = vehicle.ID
	profile.Callsign = vehicle.Callsign
	profile.HTTPListen = "127.0.0.1:0"
	profile.NATSURL = "nats://fast-e2e"
	profile.MAVLinkUDPListen = ""
	profile.MAVLinkUDPHost = ""
	profile.MAVLinkUDPPort = 0
	profile.Vehicles = 1
	profile.Hz = 1
	profile.CSAPIURL = ""
	profile.CommandRuntimeMode = handoff.CommandRuntimeHardwareReadonly
	profile.HardwareTransmitEnabled = false
	return profile
}

func evidenceStore(node companion.NodeSnapshot) *gcs.Store {
	store := gcs.NewStore("nats://fast-e2e", true)
	for range node.Frames {
		store.RecordRawFrame()
	}
	for range node.Decoded {
		store.RecordDecodedFrame()
	}
	for range node.Projected {
		store.RecordProjectedWrite()
	}
	store.ApplyProjectorSnapshot(node.Vehicles, node.Alerts)
	for i, entityID := range node.EntityIDs {
		store.RecordGraphWrite(entityID, uint64(i+1), time.Millisecond)
	}
	return store
}

func evidenceMeshIndex(node companion.NodeSnapshot, at time.Time, vehicleProfile string) (*mesh.SummaryIndex, error) {
	index := mesh.NewSummaryIndex()
	for seq, vehicle := range node.Vehicles {
		item, err := mesh.NewItem(mesh.Envelope{
			OriginNodeID:    node.NodeID,
			OriginVehicleID: vehicle.ID,
			EntityID:        vehicle.ID,
			PredicateGroup:  "vehicle.telemetry.current",
			OriginSequence:  uint64(seq + 1),
			OperationID:     fmt.Sprintf("%s:%s:%d", node.NodeID, vehicle.ID, seq+1),
			ObservedAt:      at,
			ExpiresAt:       at.Add(time.Minute),
			SourceKind:      mesh.SourceKindMAVLinkProjection,
			Confidence:      1,
			MergePolicy:     mesh.MergePolicyLastWriterWins,
		}, map[string]any{
			"battery_remaining": vehicle.BatteryRemaining,
			"lat":               vehicle.LatitudeDeg,
			"lon":               vehicle.LongitudeDeg,
			"vehicle_profile":   vehicleProfile,
		})
		if err != nil {
			return nil, err
		}
		if err := index.Upsert(item); err != nil {
			return nil, err
		}
	}
	return index, nil
}

func probeHardwareCommandBlock(server *gcs.Server, targetEntity string) (CommandPostureReport, error) {
	body := bytes.NewBufferString(fmt.Sprintf(`{"vehicle_id":%q,"verb":%q}`, targetEntity, commandgate.VerbRequestAutopilotVersion))
	req := httptest.NewRequest(http.MethodPost, "/api/commands", body)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	server.Handler().ServeHTTP(rr, req)

	var result commandgate.TransmitResult
	if err := json.Unmarshal(rr.Body.Bytes(), &result); err != nil {
		return CommandPostureReport{}, fmt.Errorf("decode hardware command block: %w", err)
	}
	return CommandPostureReport{
		HTTPStatus:      rr.Code,
		Status:          result.Status,
		HardwareBlocked: rr.Code == http.StatusConflict && result.HardwareBlock != nil && result.HardwareBlock.Status == commandgate.HardwareTransmitBlockStatus,
	}, nil
}

func probeSimulatorCommandEvidence(ctx context.Context, targetEntity string, targetSystemID uint8, at time.Time) (SimulatorCommandReport, error) {
	transmitter := &recordingTransmitter{}
	gate := commandgate.SimulatorTransmitGate{
		Preflight:       commandgate.DefaultSimulatorPreflightGate(),
		Transmitter:     transmitter,
		ACKObserver:     acceptedACKObserver{at: at},
		PostStatePoller: observedPostStatePoller{at: at},
		Now: func() time.Time {
			return at
		},
	}
	result, err := gate.Transmit(ctx, commandgate.TransmitRequest{
		Preflight: commandgate.PreflightRequest{
			RuntimeMode:              commandgate.RuntimeModeSimulator,
			SafetyProfile:            commandgate.DefaultSafetyProfile,
			TargetEntity:             targetEntity,
			Verb:                     commandgate.VerbRequestAutopilotVersion,
			RequestedBy:              "fast-companion-e2e",
			SenderSystemID:           250,
			SenderComponentID:        1,
			Attempts:                 1,
			LocalOverride:            true,
			ACKRequired:              true,
			PostStatePollingRequired: true,
			SimulatorConfirmed:       true,
			AbortReady:               true,
		},
		Sequence:           42,
		TargetSystemID:     targetSystemID,
		TargetComponentID:  semops.DefaultTargetComponentID,
		RequestedMessageID: mavlink.MessageAutopilotVersion,
	})
	if err != nil {
		return SimulatorCommandReport{}, err
	}
	ackAccepted := false
	if len(result.ACKs) > 0 {
		ackAccepted = result.ACKs[len(result.ACKs)-1].Accepted()
	}
	return SimulatorCommandReport{
		Accepted:                   result.Accepted,
		Status:                     result.Status,
		PreflightAccepted:          result.Preflight.Accepted,
		SimulatorOnly:              result.Preflight.Evidence.RuntimeMode == commandgate.RuntimeModeSimulator,
		FrameCount:                 len(result.Frames),
		ACKCount:                   len(result.ACKs),
		ACKAccepted:                ackAccepted,
		PostStateObserved:          result.PostState.Observed,
		HardwareTransmitAuthorized: result.HardwareBlock != nil,
	}, nil
}

func probeSemOpsReadback(ctx context.Context, profile handoff.Profile, node companion.NodeSnapshot, at time.Time) (SemOpsReadbackReport, error) {
	receiver := NewFakeSemOpsReceiver(FakeSemOpsConfig{})
	server := httptest.NewServer(receiver.Handler())
	defer server.Close()

	client, err := semops.NewClient(semops.ClientConfig{BaseURL: server.URL})
	if err != nil {
		return SemOpsReadbackReport{}, err
	}
	request := semops.NewAutopilotVersionRequest(semops.AutopilotVersionRequestInput{
		Profile:        profile,
		TargetSystemID: node.Vehicles[0].SystemID,
		CorrelationID:  profile.NodeID + "-autopilot-version-fast-e2e",
		IdempotencyKey: profile.NodeID + "-autopilot-version-fast-e2e",
		RequestedAt:    at,
		TTL:            semops.DefaultReadbackTTL,
	})
	response, err := client.SubmitReadback(ctx, request)
	if err != nil {
		return SemOpsReadbackReport{}, err
	}
	return SemOpsReadbackReport{
		Request:          request,
		Response:         response,
		ReceivedRequests: len(receiver.Requests()),
	}, nil
}

type recordingTransmitter struct {
	frames [][]byte
}

func (t *recordingTransmitter) TransmitMAVLink(_ context.Context, frame []byte) error {
	t.frames = append(t.frames, append([]byte(nil), frame...))
	return nil
}

type acceptedACKObserver struct {
	at time.Time
}

func (o acceptedACKObserver) AwaitCommandACK(_ context.Context, expected commandgate.ACKExpectation) (commandgate.ACKEvidence, error) {
	return commandgate.ACKEvidence{
		Attempt:           expected.Attempt,
		CommandID:         expected.CommandID,
		Result:            mavlink.MAVResultAccepted,
		SourceSystemID:    expected.TargetSystemID,
		SourceComponentID: expected.TargetComponentID,
		ObservedAt:        o.at,
	}, nil
}

type observedPostStatePoller struct {
	at time.Time
}

func (p observedPostStatePoller) PollPostState(_ context.Context, expected commandgate.PostStateExpectation) (commandgate.PostStateEvidence, error) {
	return commandgate.PostStateEvidence{
		Observed:     true,
		TargetEntity: expected.TargetEntity,
		Summary:      "AUTOPILOT_VERSION readback observed in simulator-only proof",
		ObservedAt:   p.at,
	}, nil
}

func downstreamOptional(values []gcs.DownstreamView) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if !value.Optional ||
			value.DependencyMode != gcs.DownstreamDependencyModeOptional ||
			value.RuntimeDependency ||
			value.RequiredForReadiness {
			return false
		}
	}
	return true
}

func semconnectDisabled(evidence gcs.EvidenceBundle) bool {
	if evidence.Profile.Downstream.CSAPIConfigured {
		return false
	}
	for _, value := range evidence.Downstream {
		if value.Name == "semconnect-csapi" {
			return !value.Enabled && value.Status == "disabled"
		}
	}
	return false
}

func (r *FastCompanionReport) addAssertion(name string, passed bool, detail string) {
	r.Assertions = append(r.Assertions, Assertion{Name: name, Passed: passed, Detail: detail})
}

type FakeSemOpsConfig struct {
	Response semops.ReadbackResponse
}

type FakeSemOpsReceiver struct {
	mu       sync.Mutex
	response semops.ReadbackResponse
	requests []semops.ReadbackRequest
}

func NewFakeSemOpsReceiver(cfg FakeSemOpsConfig) *FakeSemOpsReceiver {
	return &FakeSemOpsReceiver{response: cfg.Response}
}

func (r *FakeSemOpsReceiver) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if req.URL.Path != semops.ReadbackPathV0 {
			http.NotFound(w, req)
			return
		}
		if strings.TrimSpace(req.Header.Get("X-SemOps-Operator-Authenticated")) != "" {
			http.Error(w, "trusted SemOps operator headers must not be minted by SemLink", http.StatusBadRequest)
			return
		}
		defer req.Body.Close()
		var readback semops.ReadbackRequest
		decoder := json.NewDecoder(req.Body)
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&readback); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := readback.Validate(); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		r.mu.Lock()
		r.requests = append(r.requests, readback)
		r.mu.Unlock()

		response := r.responseFor(readback)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	})
}

func (r *FakeSemOpsReceiver) Requests() []semops.ReadbackRequest {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]semops.ReadbackRequest(nil), r.requests...)
}

func (r *FakeSemOpsReceiver) responseFor(request semops.ReadbackRequest) semops.ReadbackResponse {
	if r.response.Contract != "" {
		response := r.response
		return response
	}
	expiresAt := request.RequestedAt.Add(time.Duration(request.TTLSeconds) * time.Second)
	return semops.ReadbackResponse{
		Contract:                  semops.ReadbackContractV0,
		Accepted:                  true,
		Status:                    "accepted",
		CorrelationID:             request.CorrelationID,
		IdempotencyKey:            request.IdempotencyKey,
		CompanionNodeID:           request.CompanionNodeID,
		AuthorizedCompanionNodeID: request.CompanionNodeID,
		AuthorityScope:            "semlink.readback.intent",
		SourceRef:                 request.SourceRef,
		RequestedAt:               &request.RequestedAt,
		ExpiresAt:                 &expiresAt,
		NativeExecutionAllowed:    false,
		CompanionTransmitAllowed:  false,
		Mutations:                 1,
	}
}
