package e2e

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildSingleNodeDemoArtifactDefaultsToDeterministicEnvelope(t *testing.T) {
	report := SingleNodeDemoReport{
		Kind:        SingleNodeReportKind,
		GeneratedAt: time.Date(2026, 7, 9, 18, 0, 0, 0, time.UTC),
		Node: NodeReport{
			NodeID:       "vehicle-node-1",
			VehicleCount: 1,
		},
	}

	artifact, err := BuildSingleNodeDemoArtifact(report, DemoArtifactOptions{
		GeneratorProfile: "single-deterministic",
	})
	if err != nil {
		t.Fatalf("BuildSingleNodeDemoArtifact() error = %v", err)
	}
	if artifact.ArtifactKind != DemoArtifactKind {
		t.Fatalf("artifact kind = %q", artifact.ArtifactKind)
	}
	if !artifact.GeneratedAt.Equal(report.GeneratedAt) {
		t.Fatalf("artifact generated_at = %s, want %s", artifact.GeneratedAt, report.GeneratedAt)
	}
	if artifact.Source.SourceFidelity != ArtifactFidelityDeterministic {
		t.Fatalf("source fidelity = %q", artifact.Source.SourceFidelity)
	}
	if artifact.Source.SemLinkVersion != DefaultSemLinkArtifactVersion {
		t.Fatalf("semlink version = %q", artifact.Source.SemLinkVersion)
	}
	if artifact.Source.GeneratorProfile != "single-deterministic" {
		t.Fatalf("generator profile = %q", artifact.Source.GeneratorProfile)
	}
	if artifact.Source.NoTransmitPosture == "" {
		t.Fatalf("missing no-transmit posture")
	}
	embedded, ok := artifact.Report.(SingleNodeDemoReport)
	if !ok {
		t.Fatalf("embedded report type = %T", artifact.Report)
	}
	if embedded.Kind != SingleNodeReportKind || embedded.Node.NodeID != report.Node.NodeID {
		t.Fatalf("embedded report = %#v", embedded)
	}

	data, err := json.Marshal(artifact)
	if err != nil {
		t.Fatalf("marshal artifact: %v", err)
	}
	var decoded struct {
		ArtifactKind string `json:"artifact_kind"`
		GeneratedAt  string `json:"generated_at"`
		Source       struct {
			SourceFidelity    string `json:"source_fidelity"`
			SemLinkVersion    string `json:"semlink_version"`
			GeneratorProfile  string `json:"generator_profile"`
			NoTransmitPosture string `json:"no_transmit_posture"`
		} `json:"source"`
		Report struct {
			Kind        string `json:"kind"`
			GeneratedAt string `json:"generated_at"`
		} `json:"report"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal artifact JSON: %v", err)
	}
	if decoded.ArtifactKind != DemoArtifactKind || decoded.Report.Kind != SingleNodeReportKind {
		t.Fatalf("decoded artifact = %#v", decoded)
	}
	if decoded.GeneratedAt != decoded.Report.GeneratedAt {
		t.Fatalf("artifact generated_at %q != report generated_at %q", decoded.GeneratedAt, decoded.Report.GeneratedAt)
	}
}

func TestBuildSimpleMeshDemoArtifactPreservesRawMAVLinkExclusion(t *testing.T) {
	report := SimpleMeshDemoReport{
		Kind:              SimpleMeshReportKind,
		GeneratedAt:       time.Date(2026, 7, 9, 18, 10, 0, 0, time.UTC),
		VehicleProfile:    "ardurover",
		ExpectedSummaries: 2,
		Nodes: []MeshNodeReport{
			{NodeID: "vehicle-node-1", VehicleCount: 1},
			{NodeID: "vehicle-node-2", VehicleCount: 1},
		},
		RawMAVLinkExclusion: RawMAVLinkExclusionReport{
			RejectedBySummaryIndex: true,
			Policy:                 "raw MAVLink frames are local-only by default; selected summaries replicate",
		},
	}

	artifact, err := BuildSimpleMeshDemoArtifact(report, DemoArtifactOptions{
		SemLinkCommit:    "abc123",
		GeneratorCommand: "semlink-demo -mode mesh",
	})
	if err != nil {
		t.Fatalf("BuildSimpleMeshDemoArtifact() error = %v", err)
	}
	if artifact.Source.SemLinkCommit != "abc123" {
		t.Fatalf("semlink commit = %q", artifact.Source.SemLinkCommit)
	}
	embedded, ok := artifact.Report.(SimpleMeshDemoReport)
	if !ok {
		t.Fatalf("embedded report type = %T", artifact.Report)
	}
	if !embedded.RawMAVLinkExclusion.RejectedBySummaryIndex {
		t.Fatalf("raw MAVLink exclusion not preserved: %#v", embedded.RawMAVLinkExclusion)
	}
}

func TestBuildDemoArtifactRejectsUnderEvidencedLiveFidelity(t *testing.T) {
	report := SimpleMeshDemoReport{
		Kind:        SimpleMeshReportKind,
		GeneratedAt: time.Date(2026, 7, 9, 18, 20, 0, 0, time.UTC),
		Nodes:       []MeshNodeReport{{NodeID: "vehicle-node-1", VehicleCount: 1}},
	}

	_, err := BuildSimpleMeshDemoArtifact(report, DemoArtifactOptions{
		SourceFidelity: ArtifactFidelitySITLBacked,
	})
	if err == nil || !strings.Contains(err.Error(), artifactLiveNodeMetadataErrorText) {
		t.Fatalf("error = %v, want live-source metadata rejection", err)
	}
}

func TestBuildDemoArtifactRejectsLiveNodeWithoutMAVLinkSystemID(t *testing.T) {
	report := SimpleMeshDemoReport{
		Kind:        SimpleMeshReportKind,
		GeneratedAt: time.Date(2026, 7, 9, 18, 30, 0, 0, time.UTC),
		Nodes:       []MeshNodeReport{{NodeID: "vehicle-node-1", VehicleCount: 1}},
	}

	_, err := BuildSimpleMeshDemoArtifact(report, DemoArtifactOptions{
		Nodes: []DemoArtifactNodeSource{
			{
				NodeID:          "vehicle-node-1",
				SourceFidelity:  ArtifactFidelitySITLBacked,
				SimulatorFamily: "ardupilot",
				VehicleSource:   "ArduRover SITL without Gazebo",
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "requires mavlink_system_id") {
		t.Fatalf("error = %v, want mavlink_system_id rejection", err)
	}
}

func TestBuildDemoArtifactRejectsPartialTopLevelLiveSourceMetadata(t *testing.T) {
	report := SimpleMeshDemoReport{
		Kind:        SimpleMeshReportKind,
		GeneratedAt: time.Date(2026, 7, 9, 18, 35, 0, 0, time.UTC),
		Nodes: []MeshNodeReport{
			{NodeID: "vehicle-node-1", VehicleCount: 1},
			{NodeID: "vehicle-node-2", VehicleCount: 1},
		},
	}

	_, err := BuildSimpleMeshDemoArtifact(report, DemoArtifactOptions{
		SourceFidelity: ArtifactFidelitySITLBacked,
		Nodes: []DemoArtifactNodeSource{
			{
				NodeID:          "vehicle-node-1",
				SourceFidelity:  ArtifactFidelitySITLBacked,
				SimulatorFamily: "ardupilot",
				VehicleSource:   "ArduRover SITL without Gazebo",
				MAVLinkSystemID: 42,
			},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "for every report node") {
		t.Fatalf("error = %v, want every-node source metadata rejection", err)
	}
}

func TestBuildDemoArtifactRejectsUnknownNodeSource(t *testing.T) {
	report := SimpleMeshDemoReport{
		Kind:        SimpleMeshReportKind,
		GeneratedAt: time.Date(2026, 7, 9, 18, 40, 0, 0, time.UTC),
		Nodes:       []MeshNodeReport{{NodeID: "vehicle-node-1", VehicleCount: 1}},
	}

	_, err := BuildSimpleMeshDemoArtifact(report, DemoArtifactOptions{
		Nodes: []DemoArtifactNodeSource{
			{NodeID: "vehicle-node-missing", SourceFidelity: ArtifactFidelityDeterministic},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "is not present in report") {
		t.Fatalf("error = %v, want unknown node rejection", err)
	}
}
