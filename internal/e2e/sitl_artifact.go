package e2e

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/c360studio/semlink/internal/gcs"
)

const (
	DefaultSITLBackedReportPath   = ".artifacts/semlink-demo-sitl/report.json"
	DefaultSITLBackedArtifactPath = ".artifacts/semlink-demo-sitl/artifact.json"
	DefaultSITLSimulatorFamily    = "ardupilot"
	DefaultSITLVehicleSource      = "ArduPilot SITL without Gazebo"
	DefaultSITLGeneratorProfile   = "sitl-evidence"
)

type SITLBackedDemoConfig struct {
	Evidence          gcs.EvidenceBundle
	EvidenceProbe     EndpointProbe
	RuntimeBaseURL    string
	VehicleProfile    string
	SimulatorFamily   string
	VehicleSource     string
	Route             string
	SemLinkVersion    string
	SemLinkCommit     string
	GeneratorCommand  string
	GeneratorProfile  string
	NoTransmitPosture string
}

func FetchRuntimeEvidence(
	ctx context.Context,
	client *http.Client,
	runtimeBaseURL string,
) (EndpointProbe, gcs.EvidenceBundle, error) {
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	runtimeBaseURL = strings.TrimRight(strings.TrimSpace(runtimeBaseURL), "/")
	if runtimeBaseURL == "" {
		return EndpointProbe{Path: "/api/evidence", Detail: "runtime base URL is required"},
			gcs.EvidenceBundle{},
			fmt.Errorf("runtime base URL is required")
	}
	return probeEvidence(ctx, client, runtimeBaseURL)
}

func BuildSITLBackedSingleNodeDemoArtifact(cfg SITLBackedDemoConfig) (SingleNodeDemoReport, DemoArtifact, error) {
	report, nodeSource, err := BuildSITLBackedSingleNodeReport(cfg)
	if err != nil {
		return SingleNodeDemoReport{}, DemoArtifact{}, err
	}
	if strings.TrimSpace(cfg.SemLinkVersion) == "" && strings.TrimSpace(cfg.SemLinkCommit) == "" {
		return SingleNodeDemoReport{}, DemoArtifact{}, fmt.Errorf("SITL-backed artifact requires real semlink version or commit")
	}
	opts := DemoArtifactOptions{
		SourceFidelity:    ArtifactFidelitySITLBacked,
		SemLinkVersion:    cfg.SemLinkVersion,
		SemLinkCommit:     cfg.SemLinkCommit,
		GeneratorCommand:  defaultSITLGeneratorCommand(cfg),
		GeneratorProfile:  defaultSITLGeneratorProfile(cfg),
		SimulatorFamily:   defaultSITLSimulatorFamily(cfg),
		NoTransmitPosture: defaultSITLNoTransmitPosture(cfg),
		Nodes:             []DemoArtifactNodeSource{nodeSource},
	}
	artifact, err := BuildSingleNodeDemoArtifact(report, opts)
	if err != nil {
		return SingleNodeDemoReport{}, DemoArtifact{}, err
	}
	return report, artifact, nil
}

func BuildSITLBackedSingleNodeReport(cfg SITLBackedDemoConfig) (SingleNodeDemoReport, DemoArtifactNodeSource, error) {
	evidence := cfg.Evidence
	if err := validateSITLEvidence(evidence); err != nil {
		return SingleNodeDemoReport{}, DemoArtifactNodeSource{}, err
	}
	vehicle := evidence.Vehicles[0]
	probe := cfg.EvidenceProbe
	if probe.Path == "" {
		probe = EndpointProbe{
			Path:       "/api/evidence",
			HTTPStatus: http.StatusOK,
			OK:         true,
			Detail:     gcs.EvidenceContractName,
		}
	}
	report := SingleNodeDemoReport{
		Kind:           SingleNodeReportKind,
		GeneratedAt:    evidence.GeneratedAt,
		VehicleProfile: defaultSITLVehicleProfile(cfg),
		Node: NodeReport{
			NodeID:        evidence.Node.NodeID,
			VehicleCount:  len(evidence.Vehicles),
			FrameCount:    clampInt64(evidence.Node.RawFrames),
			DecodedCount:  clampInt64(evidence.Node.DecodedFrames),
			GraphEntities: clampInt64(evidence.Node.GraphWrites),
		},
		RuntimeBaseURL: strings.TrimRight(strings.TrimSpace(cfg.RuntimeBaseURL), "/"),
		Probes:         []EndpointProbe{probe},
		Evidence:       evidence,
		CommandPosture: CommandPostureReport{
			HTTPStatus: http.StatusConflict,
			Status:     evidence.Profile.Command.HardwareTransmitStatus,
			HardwareBlocked: !evidence.Profile.Command.HardwareTransmitEnabled &&
				evidence.Profile.Command.HardwareTransmitStatus == "blocked",
		},
	}
	report.addAssertion(
		"api-evidence",
		probe.OK && evidence.Contract.Name == gcs.EvidenceContractName,
		gcs.EvidenceContractName,
	)
	report.addAssertion(
		"sitl-external-mavlink-input",
		evidence.Profile.MAVLink.ExternalInputConfigured &&
			!evidence.Profile.Simulator.Enabled &&
			evidence.Profile.Simulator.Source == string(gcs.TelemetrySourceExternalMAVLinkUDP),
		evidence.Profile.Simulator.Source,
	)
	report.addAssertion(
		"sitl-vehicle-state",
		evidence.Node.RawFrames > 0 && evidence.Node.DecodedFrames > 0 &&
			len(evidence.Vehicles) > 0 && vehicle.SystemID > 0,
		fmt.Sprintf("vehicles=%d system_id=%d", len(evidence.Vehicles), vehicle.SystemID),
	)
	report.addAssertion(
		"hardware-transmit-blocked",
		report.CommandPosture.HardwareBlocked,
		evidence.Profile.Command.HardwareTransmitStatus,
	)
	report.addAssertion(
		"raw-mavlink-excluded",
		rawMAVLinkExcluded(evidence),
		evidence.Mesh.RawMAVLinkReplicationPolicy,
	)
	report.addAssertion(
		"artifact-source-metadata",
		strings.TrimSpace(defaultSITLSimulatorFamily(cfg)) != "" &&
			strings.TrimSpace(defaultSITLVehicleSource(cfg)) != "" &&
			vehicle.SystemID > 0,
		"SITL node metadata derived from evidence",
	)
	nodeSource := DemoArtifactNodeSource{
		NodeID:          evidence.Node.NodeID,
		SourceFidelity:  ArtifactFidelitySITLBacked,
		SimulatorFamily: defaultSITLSimulatorFamily(cfg),
		VehicleSource:   defaultSITLVehicleSource(cfg),
		MAVLinkSystemID: int(vehicle.SystemID),
		Route:           defaultSITLRoute(cfg, evidence),
	}
	return report, nodeSource, nil
}

func validateSITLEvidence(evidence gcs.EvidenceBundle) error {
	if evidence.GeneratedAt.IsZero() {
		return fmt.Errorf("SITL-backed artifact requires non-zero generated_at evidence")
	}
	if evidence.Contract.Name != gcs.EvidenceContractName {
		return fmt.Errorf("SITL-backed artifact requires %s evidence contract", gcs.EvidenceContractName)
	}
	if strings.TrimSpace(evidence.Node.NodeID) == "" {
		return fmt.Errorf("SITL-backed artifact requires companion node evidence")
	}
	if !evidence.Profile.MAVLink.ExternalInputConfigured ||
		evidence.Profile.Simulator.Enabled ||
		evidence.Profile.Simulator.Source != string(gcs.TelemetrySourceExternalMAVLinkUDP) {
		return fmt.Errorf("SITL-backed artifact requires external MAVLink input evidence")
	}
	if evidence.Node.RawFrames <= 0 || evidence.Node.DecodedFrames <= 0 {
		return fmt.Errorf("SITL-backed artifact requires observed raw and decoded MAVLink frames")
	}
	if len(evidence.Vehicles) == 0 {
		return fmt.Errorf("SITL-backed artifact requires decoded vehicle evidence")
	}
	if evidence.Vehicles[0].SystemID == 0 {
		return fmt.Errorf("SITL-backed artifact requires observed MAVLink system ID")
	}
	if evidence.Profile.Command.HardwareTransmitEnabled ||
		evidence.Profile.Command.HardwareTransmitStatus != "blocked" {
		return fmt.Errorf("SITL-backed artifact requires blocked hardware transmit posture")
	}
	if !rawMAVLinkExcluded(evidence) {
		return fmt.Errorf("SITL-backed artifact requires raw MAVLink mesh exclusion evidence")
	}
	return nil
}

func rawMAVLinkExcluded(evidence gcs.EvidenceBundle) bool {
	return !evidence.Mesh.RawMAVLinkReplicatesByDefault &&
		evidence.Mesh.RawMAVLinkReplicationPolicy == gcs.RawMAVLinkReplicationPolicyExclude
}

func defaultSITLVehicleProfile(cfg SITLBackedDemoConfig) string {
	if value := strings.TrimSpace(cfg.VehicleProfile); value != "" {
		return value
	}
	return "ardurover"
}

func defaultSITLSimulatorFamily(cfg SITLBackedDemoConfig) string {
	if value := strings.TrimSpace(cfg.SimulatorFamily); value != "" {
		return value
	}
	return DefaultSITLSimulatorFamily
}

func defaultSITLVehicleSource(cfg SITLBackedDemoConfig) string {
	if value := strings.TrimSpace(cfg.VehicleSource); value != "" {
		return value
	}
	return DefaultSITLVehicleSource
}

func defaultSITLGeneratorProfile(cfg SITLBackedDemoConfig) string {
	if value := strings.TrimSpace(cfg.GeneratorProfile); value != "" {
		return value
	}
	return DefaultSITLGeneratorProfile
}

func defaultSITLGeneratorCommand(cfg SITLBackedDemoConfig) string {
	if value := strings.TrimSpace(cfg.GeneratorCommand); value != "" {
		return value
	}
	return "semlink-demo -mode sitl-artifact"
}

func defaultSITLNoTransmitPosture(cfg SITLBackedDemoConfig) string {
	if value := strings.TrimSpace(cfg.NoTransmitPosture); value != "" {
		return value
	}
	command := cfg.Evidence.Profile.Command
	return fmt.Sprintf(
		"SITL evidence only; SemLink hardware transmit status=%s runtime_mode=%s; no native SemOps transmit authority",
		command.HardwareTransmitStatus,
		command.RuntimeMode,
	)
}

func defaultSITLRoute(cfg SITLBackedDemoConfig, evidence gcs.EvidenceBundle) string {
	if value := strings.TrimSpace(cfg.Route); value != "" {
		return value
	}
	if value := strings.TrimSpace(evidence.Profile.MAVLink.UDPListen); value != "" {
		return "mavlink-udp-listen=" + value
	}
	if evidence.Profile.MAVLink.UDPHost != "" && evidence.Profile.MAVLink.UDPPort > 0 {
		return fmt.Sprintf(
			"mavlink-udp=%s:%d",
			evidence.Profile.MAVLink.UDPHost,
			evidence.Profile.MAVLink.UDPPort,
		)
	}
	return ""
}

func clampInt64(value int64) int {
	if value <= 0 {
		return 0
	}
	maxInt := int64(^uint(0) >> 1)
	if value > maxInt {
		return int(maxInt)
	}
	return int(value)
}
