package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"time"

	"github.com/c360studio/semlink/internal/blueos"
	"github.com/c360studio/semlink/internal/companion"
	"github.com/c360studio/semlink/internal/gcs"
)

const (
	SingleNodeReportKind        = "single-node-companion-demo"
	DefaultSingleNodeReportPath = ".artifacts/semlink-demo-single/report.json"
)

type SingleNodeDemoConfig struct {
	VehicleProfile string
	Start          time.Time
	StepElapsed    time.Duration
}

type SingleNodeDemoReport struct {
	Kind             string                 `json:"kind"`
	GeneratedAt      time.Time              `json:"generated_at"`
	VehicleProfile   string                 `json:"vehicle_profile"`
	Node             NodeReport             `json:"node"`
	RuntimeBaseURL   string                 `json:"runtime_base_url"`
	Probes           []EndpointProbe        `json:"probes"`
	Evidence         gcs.EvidenceBundle     `json:"evidence"`
	CommandPosture   CommandPostureReport   `json:"command_posture"`
	SimulatorCommand SimulatorCommandReport `json:"simulator_command"`
	SemOpsReadback   SemOpsReadbackReport   `json:"semops_readback"`
	Assertions       []Assertion            `json:"assertions"`
}

type EndpointProbe struct {
	Path       string `json:"path"`
	HTTPStatus int    `json:"http_status"`
	OK         bool   `json:"ok"`
	Detail     string `json:"detail,omitempty"`
}

func RunSingleNodeDemo(ctx context.Context, cfg SingleNodeDemoConfig) (SingleNodeDemoReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	fastCfg := FastCompanionConfig{
		Nodes:           1,
		VehiclesPerNode: 1,
		VehicleProfile:  cfg.VehicleProfile,
		Start:           cfg.Start,
		StepElapsed:     cfg.StepElapsed,
	}.withDefaults()

	harness, err := companion.NewHarness(companion.HarnessConfig{
		Nodes:           1,
		NodeIDPrefix:    defaultNodeIDPrefix,
		VehiclesPerNode: 1,
		Start:           fastCfg.Start,
	})
	if err != nil {
		return SingleNodeDemoReport{}, err
	}
	step, err := harness.Step(ctx, fastCfg.StepElapsed)
	if err != nil {
		return SingleNodeDemoReport{}, err
	}
	if len(step.Nodes) != 1 || len(step.Nodes[0].Vehicles) == 0 {
		return SingleNodeDemoReport{}, fmt.Errorf("single-node demo expected one node with vehicle state, got %#v", step.Nodes)
	}

	node := step.Nodes[0]
	profile := fastProfile(node, fastCfg)
	store := evidenceStore(node)
	index, err := evidenceMeshIndex(node, step.At, fastCfg.VehicleProfile)
	if err != nil {
		return SingleNodeDemoReport{}, err
	}
	server := gcs.NewServer(store, nil, "", gcs.ServerOptions{
		NodeID:         node.NodeID,
		MeshIndex:      index,
		HandoffProfile: &profile,
	})
	runtime := httptest.NewServer(server.Handler())
	defer runtime.Close()

	report := SingleNodeDemoReport{
		Kind:           SingleNodeReportKind,
		GeneratedAt:    step.At,
		VehicleProfile: fastCfg.VehicleProfile,
		Node:           nodeReports([]companion.NodeSnapshot{node})[0],
		RuntimeBaseURL: runtime.URL,
	}
	report.addAssertion("single-node-runtime", report.Node.NodeID != "" && report.Node.VehicleCount == 1, "one deterministic node launched")

	client := &http.Client{Timeout: 2 * time.Second}
	healthProbe, err := probeHealth(ctx, client, runtime.URL)
	if err != nil {
		return SingleNodeDemoReport{}, err
	}
	report.Probes = append(report.Probes, healthProbe)
	report.addAssertion("runtime-readiness", healthProbe.OK, healthProbe.Detail)

	registrationProbe, err := probeRegistration(ctx, client, runtime.URL)
	if err != nil {
		return SingleNodeDemoReport{}, err
	}
	report.Probes = append(report.Probes, registrationProbe)
	report.addAssertion("register-service", registrationProbe.OK, registrationProbe.Detail)

	evidenceProbe, evidence, err := probeEvidence(ctx, client, runtime.URL)
	if err != nil {
		return SingleNodeDemoReport{}, err
	}
	report.Probes = append(report.Probes, evidenceProbe)
	report.Evidence = evidence
	report.addAssertion("api-evidence", evidenceProbe.OK && evidence.Contract.Name == gcs.EvidenceContractName, evidenceProbe.Detail)
	report.addAssertion("companion-profile", evidence.Profile.Status == "configured" && evidence.Profile.Simulator.Enabled, evidence.Profile.Status)
	report.addAssertion("downstream-dependency-posture", downstreamOptional(evidence.Downstream), "downstream consumers are optional")

	commandPosture, err := probeHardwareCommandBlock(server, node.Vehicles[0].ID)
	if err != nil {
		return SingleNodeDemoReport{}, err
	}
	report.CommandPosture = commandPosture
	report.addAssertion("hardware-transmit-blocked", commandPosture.HardwareBlocked, commandPosture.Status)

	simulatorCommand, err := probeSimulatorCommandEvidence(ctx, node.Vehicles[0].ID, node.Vehicles[0].SystemID, step.At)
	if err != nil {
		return SingleNodeDemoReport{}, err
	}
	report.SimulatorCommand = simulatorCommand
	report.addAssertion("simulator-command-evidence-distinct", simulatorCommand.SimulatorOnly && !simulatorCommand.HardwareTransmitAuthorized, simulatorCommand.Status)

	readback, err := probeSemOpsReadback(ctx, profile, node, step.At)
	if err != nil {
		return SingleNodeDemoReport{}, err
	}
	report.SemOpsReadback = readback
	report.addAssertion("semops-readback-v0", readback.ReceivedRequests == 1 && readback.Response.Accepted, readback.Response.Status)
	return report, nil
}

func WriteJSONReport(path string, value any) error {
	if path == "" {
		return fmt.Errorf("report path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func probeHealth(ctx context.Context, client *http.Client, baseURL string) (EndpointProbe, error) {
	var body struct {
		OK bool `json:"ok"`
	}
	probe, err := getJSON(ctx, client, baseURL, "/api/health", &body)
	probe.OK = probe.OK && body.OK
	if body.OK {
		probe.Detail = "health ok"
	}
	return probe, err
}

func probeRegistration(ctx context.Context, client *http.Client, baseURL string) (EndpointProbe, error) {
	var body blueos.Registration
	probe, err := getJSON(ctx, client, baseURL, "/register_service", &body)
	probe.OK = probe.OK && body.Name != ""
	if body.Name != "" {
		probe.Detail = body.Name
	}
	return probe, err
}

func probeEvidence(ctx context.Context, client *http.Client, baseURL string) (EndpointProbe, gcs.EvidenceBundle, error) {
	var body gcs.EvidenceBundle
	probe, err := getJSON(ctx, client, baseURL, "/api/evidence", &body)
	probe.OK = probe.OK && body.Contract.Name == gcs.EvidenceContractName
	if body.Contract.Name != "" {
		probe.Detail = body.Contract.Name
	}
	return probe, body, err
}

func getJSON(ctx context.Context, client *http.Client, baseURL, path string, out any) (EndpointProbe, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+path, nil)
	if err != nil {
		return EndpointProbe{Path: path, Detail: err.Error()}, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return EndpointProbe{Path: path, Detail: err.Error()}, err
	}
	defer resp.Body.Close()
	probe := EndpointProbe{
		Path:       path,
		HTTPStatus: resp.StatusCode,
		OK:         resp.StatusCode == http.StatusOK,
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		probe.OK = false
		probe.Detail = err.Error()
		return probe, err
	}
	return probe, nil
}

func (r *SingleNodeDemoReport) addAssertion(name string, passed bool, detail string) {
	r.Assertions = append(r.Assertions, Assertion{Name: name, Passed: passed, Detail: detail})
}
