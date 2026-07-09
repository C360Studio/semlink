package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/gcs"
)

func TestRunSingleNodeDemoProbesRuntimeAndWritesReport(t *testing.T) {
	report, err := RunSingleNodeDemo(context.Background(), SingleNodeDemoConfig{
		VehicleProfile: "ardurover",
		Start:          time.Date(2026, 7, 9, 16, 0, 0, 0, time.UTC),
		StepElapsed:    20 * time.Second,
	})
	if err != nil {
		t.Fatalf("RunSingleNodeDemo() error = %v", err)
	}
	if report.Kind != SingleNodeReportKind {
		t.Fatalf("kind = %q", report.Kind)
	}
	if report.VehicleProfile != "ardurover" {
		t.Fatalf("vehicle profile = %q", report.VehicleProfile)
	}
	if report.Node.VehicleCount != 1 || report.Node.GraphEntities == 0 {
		t.Fatalf("node = %#v", report.Node)
	}
	requireProbe(t, report.Probes, "/api/health")
	requireProbe(t, report.Probes, "/register_service")
	requireProbe(t, report.Probes, "/api/evidence")
	requireAssertion(t, report.Assertions, "runtime-readiness")
	requireAssertion(t, report.Assertions, "register-service")
	requireAssertion(t, report.Assertions, "api-evidence")
	requireAssertion(t, report.Assertions, "companion-profile")
	requireAssertion(t, report.Assertions, "downstream-dependency-posture")
	requireAssertion(t, report.Assertions, "hardware-transmit-blocked")
	requireAssertion(t, report.Assertions, "simulator-command-evidence-distinct")

	if report.Evidence.Contract.Name != gcs.EvidenceContractName {
		t.Fatalf("evidence contract = %#v", report.Evidence.Contract)
	}
	if report.CommandPosture.HTTPStatus != 409 || !report.CommandPosture.HardwareBlocked {
		t.Fatalf("command posture = %#v", report.CommandPosture)
	}
	if !report.SimulatorCommand.Accepted ||
		!report.SimulatorCommand.SimulatorOnly ||
		report.SimulatorCommand.HardwareTransmitAuthorized ||
		!report.SimulatorCommand.PostStateObserved {
		t.Fatalf("simulator command = %#v", report.SimulatorCommand)
	}

	path := filepath.Join(t.TempDir(), "report.json")
	if err := WriteJSONReport(path, report); err != nil {
		t.Fatalf("WriteJSONReport() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(report) error = %v", err)
	}
	var decoded SingleNodeDemoReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if decoded.Kind != SingleNodeReportKind || decoded.Node.NodeID != report.Node.NodeID {
		t.Fatalf("decoded report = %#v", decoded)
	}
}

func requireProbe(t *testing.T, probes []EndpointProbe, path string) {
	t.Helper()
	for _, probe := range probes {
		if probe.Path == path {
			if !probe.OK || probe.HTTPStatus != 200 {
				t.Fatalf("probe %s = %#v", path, probe)
			}
			return
		}
	}
	t.Fatalf("probe %s missing from %#v", path, probes)
}
