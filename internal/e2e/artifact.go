package e2e

import (
	"fmt"
	"strings"
	"time"
)

const (
	DemoArtifactKind = "semlink-companion-demo-artifact-v0"

	ArtifactFidelityDeterministic    = "deterministic"
	ArtifactFidelitySITLBacked       = "sitl-backed"
	ArtifactFidelityHardwareAdjacent = "hardware-adjacent"
	DefaultSemLinkArtifactVersion    = "dev"
	DefaultArtifactGeneratorProfile  = "semlink-demo-deterministic"
	DefaultArtifactNoTransmitPosture = "SemLink companion demo artifact; " +
		"no native SemOps transmit authority; no companion hardware transmit authority; " +
		"deterministic simulator-only proof"
	defaultArtifactGeneratorCommand   = "semlink-demo"
	artifactTimestampRequiredError    = "demo report generated_at is required"
	artifactSourceReferenceError      = "artifact source requires semlink version or commit"
	artifactGeneratorReferenceError   = "artifact source requires generator command or profile"
	artifactNoTransmitPostureError    = "artifact source requires no-transmit posture"
	artifactLiveNodeMetadataErrorText = "live-source fidelity requires node source metadata"
)

type DemoArtifact struct {
	ArtifactKind string             `json:"artifact_kind"`
	GeneratedAt  time.Time          `json:"generated_at"`
	Source       DemoArtifactSource `json:"source"`
	Report       any                `json:"report"`
}

type DemoArtifactSource struct {
	SourceFidelity    string                   `json:"source_fidelity"`
	SemLinkVersion    string                   `json:"semlink_version,omitempty"`
	SemLinkCommit     string                   `json:"semlink_commit,omitempty"`
	GeneratorCommand  string                   `json:"generator_command,omitempty"`
	GeneratorProfile  string                   `json:"generator_profile,omitempty"`
	SimulatorFamily   string                   `json:"simulator_family,omitempty"`
	NoTransmitPosture string                   `json:"no_transmit_posture,omitempty"`
	Nodes             []DemoArtifactNodeSource `json:"nodes,omitempty"`
}

type DemoArtifactNodeSource struct {
	NodeID          string `json:"node_id"`
	SourceFidelity  string `json:"source_fidelity"`
	SimulatorFamily string `json:"simulator_family,omitempty"`
	VehicleSource   string `json:"vehicle_source,omitempty"`
	MAVLinkSystemID int    `json:"mavlink_system_id,omitempty"`
	Route           string `json:"route,omitempty"`
}

type DemoArtifactOptions struct {
	SourceFidelity    string
	SemLinkVersion    string
	SemLinkCommit     string
	GeneratorCommand  string
	GeneratorProfile  string
	SimulatorFamily   string
	NoTransmitPosture string
	Nodes             []DemoArtifactNodeSource
}

func BuildSingleNodeDemoArtifact(report SingleNodeDemoReport, opts DemoArtifactOptions) (DemoArtifact, error) {
	if report.Kind != SingleNodeReportKind {
		return DemoArtifact{}, fmt.Errorf(
			"single-node artifact requires report kind %q, got %q",
			SingleNodeReportKind,
			report.Kind,
		)
	}
	return buildDemoArtifact(report.GeneratedAt, []string{report.Node.NodeID}, report, opts)
}

func BuildSimpleMeshDemoArtifact(report SimpleMeshDemoReport, opts DemoArtifactOptions) (DemoArtifact, error) {
	if report.Kind != SimpleMeshReportKind {
		return DemoArtifact{}, fmt.Errorf(
			"simple-mesh artifact requires report kind %q, got %q",
			SimpleMeshReportKind,
			report.Kind,
		)
	}
	nodeIDs := make([]string, 0, len(report.Nodes))
	for _, node := range report.Nodes {
		nodeIDs = append(nodeIDs, node.NodeID)
	}
	return buildDemoArtifact(report.GeneratedAt, nodeIDs, report, opts)
}

func buildDemoArtifact(
	generatedAt time.Time,
	reportNodeIDs []string,
	report any,
	opts DemoArtifactOptions,
) (DemoArtifact, error) {
	if generatedAt.IsZero() {
		return DemoArtifact{}, fmt.Errorf(artifactTimestampRequiredError)
	}
	source, err := artifactSource(opts)
	if err != nil {
		return DemoArtifact{}, err
	}
	if err := validateArtifactSource(source, reportNodeIDs); err != nil {
		return DemoArtifact{}, err
	}
	return DemoArtifact{
		ArtifactKind: DemoArtifactKind,
		GeneratedAt:  generatedAt,
		Source:       source,
		Report:       report,
	}, nil
}

func artifactSource(opts DemoArtifactOptions) (DemoArtifactSource, error) {
	source := DemoArtifactSource{
		SourceFidelity:    normalizeArtifactFidelity(opts.SourceFidelity),
		SemLinkVersion:    strings.TrimSpace(opts.SemLinkVersion),
		SemLinkCommit:     strings.TrimSpace(opts.SemLinkCommit),
		GeneratorCommand:  strings.TrimSpace(opts.GeneratorCommand),
		GeneratorProfile:  strings.TrimSpace(opts.GeneratorProfile),
		SimulatorFamily:   strings.TrimSpace(opts.SimulatorFamily),
		NoTransmitPosture: strings.TrimSpace(opts.NoTransmitPosture),
		Nodes:             append([]DemoArtifactNodeSource(nil), opts.Nodes...),
	}
	if source.SourceFidelity == "" {
		source.SourceFidelity = ArtifactFidelityDeterministic
	}
	if source.SemLinkVersion == "" && source.SemLinkCommit == "" {
		source.SemLinkVersion = DefaultSemLinkArtifactVersion
	}
	if source.GeneratorCommand == "" && source.GeneratorProfile == "" {
		source.GeneratorCommand = defaultArtifactGeneratorCommand
		source.GeneratorProfile = DefaultArtifactGeneratorProfile
	}
	if source.NoTransmitPosture == "" {
		source.NoTransmitPosture = DefaultArtifactNoTransmitPosture
	}
	if !isKnownArtifactFidelity(source.SourceFidelity) {
		return DemoArtifactSource{}, fmt.Errorf("unsupported artifact source_fidelity %q", source.SourceFidelity)
	}
	if source.SemLinkVersion == "" && source.SemLinkCommit == "" {
		return DemoArtifactSource{}, fmt.Errorf(artifactSourceReferenceError)
	}
	if source.GeneratorCommand == "" && source.GeneratorProfile == "" {
		return DemoArtifactSource{}, fmt.Errorf(artifactGeneratorReferenceError)
	}
	if source.NoTransmitPosture == "" {
		return DemoArtifactSource{}, fmt.Errorf(artifactNoTransmitPostureError)
	}
	return source, nil
}

func validateArtifactSource(source DemoArtifactSource, reportNodeIDs []string) error {
	reportNodes := make(map[string]struct{}, len(reportNodeIDs))
	for _, nodeID := range reportNodeIDs {
		nodeID = strings.TrimSpace(nodeID)
		if nodeID != "" {
			reportNodes[nodeID] = struct{}{}
		}
	}
	if len(reportNodes) == 0 {
		return fmt.Errorf("artifact source requires at least one report node")
	}
	if isLiveArtifactFidelity(source.SourceFidelity) && len(source.Nodes) == 0 {
		return fmt.Errorf("%s for source_fidelity %q", artifactLiveNodeMetadataErrorText, source.SourceFidelity)
	}
	seen := make(map[string]struct{}, len(source.Nodes))
	for _, node := range source.Nodes {
		node.NodeID = strings.TrimSpace(node.NodeID)
		if node.NodeID == "" {
			return fmt.Errorf("artifact node source requires node_id")
		}
		if _, ok := reportNodes[node.NodeID]; !ok {
			return fmt.Errorf("artifact node source %q is not present in report", node.NodeID)
		}
		if _, ok := seen[node.NodeID]; ok {
			return fmt.Errorf("artifact node source has duplicate node_id %q", node.NodeID)
		}
		seen[node.NodeID] = struct{}{}
		nodeFidelity := normalizeArtifactFidelity(node.SourceFidelity)
		if nodeFidelity == "" {
			nodeFidelity = source.SourceFidelity
		}
		if !isKnownArtifactFidelity(nodeFidelity) {
			return fmt.Errorf("unsupported artifact node source_fidelity %q", node.SourceFidelity)
		}
		if err := validateArtifactNodeSource(node, nodeFidelity); err != nil {
			return err
		}
	}
	if isLiveArtifactFidelity(source.SourceFidelity) && len(seen) != len(reportNodes) {
		return fmt.Errorf(
			"%s for every report node when source_fidelity is %q",
			artifactLiveNodeMetadataErrorText,
			source.SourceFidelity,
		)
	}
	return nil
}

func validateArtifactNodeSource(node DemoArtifactNodeSource, fidelity string) error {
	switch fidelity {
	case ArtifactFidelitySITLBacked:
		if strings.TrimSpace(node.SimulatorFamily) == "" {
			return fmt.Errorf("SITL-backed artifact node %q requires simulator_family", node.NodeID)
		}
		if strings.TrimSpace(node.VehicleSource) == "" {
			return fmt.Errorf("SITL-backed artifact node %q requires vehicle_source", node.NodeID)
		}
		if node.MAVLinkSystemID <= 0 || node.MAVLinkSystemID > 255 {
			return fmt.Errorf("SITL-backed artifact node %q requires mavlink_system_id 1..255", node.NodeID)
		}
	case ArtifactFidelityHardwareAdjacent:
		if strings.TrimSpace(node.VehicleSource) == "" {
			return fmt.Errorf("hardware-adjacent artifact node %q requires vehicle_source", node.NodeID)
		}
		if node.MAVLinkSystemID <= 0 || node.MAVLinkSystemID > 255 {
			return fmt.Errorf("hardware-adjacent artifact node %q requires mavlink_system_id 1..255", node.NodeID)
		}
	}
	return nil
}

func normalizeArtifactFidelity(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func isKnownArtifactFidelity(value string) bool {
	switch normalizeArtifactFidelity(value) {
	case ArtifactFidelityDeterministic, ArtifactFidelitySITLBacked, ArtifactFidelityHardwareAdjacent:
		return true
	default:
		return false
	}
}

func isLiveArtifactFidelity(value string) bool {
	switch normalizeArtifactFidelity(value) {
	case ArtifactFidelitySITLBacked, ArtifactFidelityHardwareAdjacent:
		return true
	default:
		return false
	}
}
