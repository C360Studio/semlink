package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/gcs"
)

func TestBuildSITLBackedSingleNodeDemoArtifactUsesEvidenceMetadata(t *testing.T) {
	evidence := validSITLArtifactEvidence()

	report, artifact, err := BuildSITLBackedSingleNodeDemoArtifact(SITLBackedDemoConfig{
		Evidence:         evidence,
		RuntimeBaseURL:   "http://127.0.0.1:8081/",
		VehicleProfile:   "ardurover",
		SemLinkCommit:    "abc123",
		GeneratorCommand: "semlink-demo -mode sitl-artifact -runtime-url http://127.0.0.1:8081",
	})
	if err != nil {
		t.Fatalf("BuildSITLBackedSingleNodeDemoArtifact() error = %v", err)
	}
	if report.Kind != SingleNodeReportKind {
		t.Fatalf("report kind = %q", report.Kind)
	}
	if report.RuntimeBaseURL != "http://127.0.0.1:8081" {
		t.Fatalf("runtime base URL = %q", report.RuntimeBaseURL)
	}
	if report.Node.NodeID != evidence.Node.NodeID ||
		report.Node.VehicleCount != 1 ||
		report.Node.FrameCount != int(evidence.Node.RawFrames) ||
		report.Node.DecodedCount != int(evidence.Node.DecodedFrames) {
		t.Fatalf("report node = %#v", report.Node)
	}
	requireAssertion(t, report.Assertions, "sitl-external-mavlink-input")
	requireAssertion(t, report.Assertions, "sitl-vehicle-state")
	requireAssertion(t, report.Assertions, "hardware-transmit-blocked")
	requireAssertion(t, report.Assertions, "raw-mavlink-excluded")
	requireAssertion(t, report.Assertions, "artifact-source-metadata")

	if artifact.ArtifactKind != DemoArtifactKind {
		t.Fatalf("artifact kind = %q", artifact.ArtifactKind)
	}
	if artifact.Source.SourceFidelity != ArtifactFidelitySITLBacked {
		t.Fatalf("source fidelity = %q", artifact.Source.SourceFidelity)
	}
	if artifact.Source.SemLinkCommit != "abc123" {
		t.Fatalf("semlink commit = %q", artifact.Source.SemLinkCommit)
	}
	if artifact.Source.SemLinkVersion != "" {
		t.Fatalf("unexpected semlink version = %q", artifact.Source.SemLinkVersion)
	}
	if artifact.Source.SimulatorFamily != DefaultSITLSimulatorFamily {
		t.Fatalf("simulator family = %q", artifact.Source.SimulatorFamily)
	}
	if len(artifact.Source.Nodes) != 1 {
		t.Fatalf("artifact nodes = %#v", artifact.Source.Nodes)
	}
	node := artifact.Source.Nodes[0]
	if node.NodeID != evidence.Node.NodeID ||
		node.SourceFidelity != ArtifactFidelitySITLBacked ||
		node.SimulatorFamily != DefaultSITLSimulatorFamily ||
		node.VehicleSource != DefaultSITLVehicleSource ||
		node.MAVLinkSystemID != 42 ||
		node.Route != "mavlink-udp-listen=:14550" {
		t.Fatalf("artifact node source = %#v", node)
	}
	if !strings.Contains(artifact.Source.NoTransmitPosture, "blocked") {
		t.Fatalf("no-transmit posture = %q", artifact.Source.NoTransmitPosture)
	}
}

func TestBuildSITLBackedSingleNodeDemoArtifactRejectsMissingExternalInput(t *testing.T) {
	evidence := validSITLArtifactEvidence()
	evidence.Profile.MAVLink.ExternalInputConfigured = false
	evidence.Profile.Simulator.Enabled = true
	evidence.Profile.Simulator.Source = "internal-simulator"

	_, _, err := BuildSITLBackedSingleNodeDemoArtifact(SITLBackedDemoConfig{
		Evidence:      evidence,
		SemLinkCommit: "abc123",
	})
	if err == nil || !strings.Contains(err.Error(), "external MAVLink input") {
		t.Fatalf("error = %v, want external MAVLink rejection", err)
	}
}

func TestBuildSITLBackedSingleNodeDemoArtifactRejectsMissingSourceRef(t *testing.T) {
	_, _, err := BuildSITLBackedSingleNodeDemoArtifact(SITLBackedDemoConfig{
		Evidence: validSITLArtifactEvidence(),
	})
	if err == nil || !strings.Contains(err.Error(), "version or commit") {
		t.Fatalf("error = %v, want source ref rejection", err)
	}
}

func TestBuildSITLBackedSingleNodeDemoArtifactRejectsMissingMAVLinkSystemID(t *testing.T) {
	evidence := validSITLArtifactEvidence()
	evidence.Vehicles[0].SystemID = 0

	_, _, err := BuildSITLBackedSingleNodeDemoArtifact(SITLBackedDemoConfig{
		Evidence:      evidence,
		SemLinkCommit: "abc123",
	})
	if err == nil || !strings.Contains(err.Error(), "MAVLink system ID") {
		t.Fatalf("error = %v, want MAVLink system ID rejection", err)
	}
}

func TestBuildSITLBackedSingleNodeDemoArtifactRejectsReplicatedRawMAVLink(t *testing.T) {
	evidence := validSITLArtifactEvidence()
	evidence.Mesh.RawMAVLinkReplicatesByDefault = true

	_, _, err := BuildSITLBackedSingleNodeDemoArtifact(SITLBackedDemoConfig{
		Evidence:      evidence,
		SemLinkCommit: "abc123",
	})
	if err == nil || !strings.Contains(err.Error(), "raw MAVLink mesh exclusion") {
		t.Fatalf("error = %v, want raw MAVLink exclusion rejection", err)
	}
}

func validSITLArtifactEvidence() gcs.EvidenceBundle {
	generatedAt := time.Date(2026, 7, 11, 15, 0, 0, 0, time.UTC)
	return gcs.EvidenceBundle{
		Contract: gcs.EvidenceContract{
			Name:    gcs.EvidenceContractName,
			Version: gcs.EvidenceContractVersion,
		},
		GeneratedAt: generatedAt,
		Node: gcs.NodeEvidence{
			NodeID:        "vehicle-node-sitl-1",
			Status:        "ok",
			RawFrames:     5,
			DecodedFrames: 5,
			GraphWrites:   3,
		},
		Profile: gcs.ProfileEvidence{
			Status: "configured",
			MAVLink: gcs.MAVLinkProfileEvidence{
				UDPListen:               ":14550",
				ExternalInputConfigured: true,
			},
			Simulator: gcs.SimulatorProfileEvidence{
				Enabled: false,
				Source:  string(gcs.TelemetrySourceExternalMAVLinkUDP),
			},
			Command: gcs.CommandProfileEvidence{
				RuntimeMode:             "hardware-readonly",
				HardwareTransmitEnabled: false,
				HardwareTransmitStatus:  "blocked",
			},
		},
		Vehicles: []gcs.VehicleEvidence{
			{
				EntityID:      "vehicle:42",
				Callsign:      "ROVER42",
				SystemID:      42,
				VehicleType:   "ground-rover",
				LinkStatus:    "online",
				LastSeen:      generatedAt,
				EvidenceClass: "mavlink-current-state",
			},
		},
		Mesh: gcs.MeshEvidence{
			RawMAVLinkReplicatesByDefault: false,
			RawMAVLinkReplicationPolicy:   gcs.RawMAVLinkReplicationPolicyExclude,
		},
	}
}
