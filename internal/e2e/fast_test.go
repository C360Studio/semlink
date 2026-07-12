package e2e

import (
	"context"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/gcs"
	"github.com/c360studio/semlink/internal/semops"
)

func TestRunFastCompanionE2EProducesRepoOwnedReport(t *testing.T) {
	start := time.Date(2026, 7, 9, 15, 0, 0, 0, time.UTC)

	report, err := RunFastCompanionE2E(context.Background(), FastCompanionConfig{
		Nodes:          2,
		VehicleProfile: "mavlink-generic",
		Start:          start,
		StepElapsed:    20 * time.Second,
	})
	if err != nil {
		t.Fatalf("RunFastCompanionE2E() error = %v", err)
	}

	if report.Kind != "fast-companion-e2e" {
		t.Fatalf("kind = %q", report.Kind)
	}
	if report.VehicleProfile != "mavlink-generic" {
		t.Fatalf("vehicle profile = %q", report.VehicleProfile)
	}
	if len(report.Nodes) != 2 {
		t.Fatalf("node count = %d, want 2", len(report.Nodes))
	}
	for _, node := range report.Nodes {
		if node.NodeID == "" {
			t.Fatalf("node has empty ID: %#v", node)
		}
		if node.VehicleCount != 1 || node.GraphEntities == 0 {
			t.Fatalf("node missing projected state: %#v", node)
		}
	}

	if !report.EvidencePresent || report.Evidence.Node.NodeID == "" {
		t.Fatalf("evidence missing: %#v", report.Evidence)
	}
	if len(report.Evidence.Vehicles) == 0 {
		t.Fatalf("evidence has no vehicles: %#v", report.Evidence)
	}
	semconnect, ok := downstreamByName(report.Evidence.Downstream, "semconnect-csapi")
	if !ok {
		t.Fatalf("semconnect downstream missing: %#v", report.Evidence.Downstream)
	}
	if semconnect.DependencyMode != gcs.DownstreamDependencyModeOptional ||
		semconnect.RuntimeDependency ||
		semconnect.RequiredForReadiness ||
		semconnect.Enabled ||
		semconnect.Status != "disabled" {
		t.Fatalf("semconnect posture = %#v", semconnect)
	}

	if report.CommandPosture.Status != "hardware-transmit-blocked" ||
		report.CommandPosture.HTTPStatus != 409 {
		t.Fatalf("command posture = %#v", report.CommandPosture)
	}
	if !report.SimulatorCommand.Accepted ||
		!report.SimulatorCommand.SimulatorOnly ||
		report.SimulatorCommand.HardwareTransmitAuthorized ||
		report.SimulatorCommand.FrameCount != 1 ||
		report.SimulatorCommand.ACKCount != 1 ||
		!report.SimulatorCommand.PostStateObserved {
		t.Fatalf("simulator command = %#v", report.SimulatorCommand)
	}
	if report.SemOpsReadback.Request.Contract != semops.ReadbackContractV0 {
		t.Fatalf("readback request = %#v", report.SemOpsReadback.Request)
	}
	if !report.SemOpsReadback.Response.Accepted ||
		report.SemOpsReadback.Response.NativeExecutionAllowed ||
		report.SemOpsReadback.Response.CompanionTransmitAllowed {
		t.Fatalf("readback response = %#v", report.SemOpsReadback.Response)
	}
	if report.SemOpsReadback.ReceivedRequests != 1 {
		t.Fatalf("fake SemOps received %d requests, want 1", report.SemOpsReadback.ReceivedRequests)
	}

	requireAssertion(t, report.Assertions, "deterministic-companion-nodes")
	requireAssertion(t, report.Assertions, "projected-local-state")
	requireAssertion(t, report.Assertions, "evidence-compatible-state")
	requireAssertion(t, report.Assertions, "downstream-optional-posture")
	requireAssertion(t, report.Assertions, "no-csapi-hot-path")
	requireAssertion(t, report.Assertions, "hardware-transmit-blocked")
	requireAssertion(t, report.Assertions, "simulator-command-evidence-distinct")
	requireAssertion(t, report.Assertions, "semops-readback-v0")
}

func downstreamByName(values []gcs.DownstreamView, name string) (gcs.DownstreamView, bool) {
	for _, value := range values {
		if value.Name == name {
			return value, true
		}
	}
	return gcs.DownstreamView{}, false
}

func requireAssertion(t *testing.T, assertions []Assertion, name string) {
	t.Helper()
	for _, assertion := range assertions {
		if assertion.Name == name {
			if !assertion.Passed {
				t.Fatalf("assertion %q failed: %#v", name, assertion)
			}
			return
		}
	}
	t.Fatalf("assertion %q missing from %#v", name, assertions)
}
