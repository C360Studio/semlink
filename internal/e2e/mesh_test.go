package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRunSimpleMeshDemoCatchesUpOneVehiclePerCompanionAndWritesReport(t *testing.T) {
	report, err := RunSimpleMeshDemo(context.Background(), SimpleMeshDemoConfig{
		Nodes:          3,
		VehicleProfile: "ardurover",
		Start:          time.Date(2026, 7, 9, 17, 0, 0, 0, time.UTC),
		StepElapsed:    20 * time.Second,
		MaxDiffItems:   8,
	})
	if err != nil {
		t.Fatalf("RunSimpleMeshDemo() error = %v", err)
	}
	if report.Kind != SimpleMeshReportKind {
		t.Fatalf("kind = %q", report.Kind)
	}
	if report.ExpectedSummaries != 3 {
		t.Fatalf("expected summaries = %d, want 3", report.ExpectedSummaries)
	}
	if len(report.Nodes) != 3 {
		t.Fatalf("nodes = %#v", report.Nodes)
	}
	for _, node := range report.Nodes {
		if node.VehicleCount != 1 {
			t.Fatalf("node %s vehicle count = %d, want 1", node.NodeID, node.VehicleCount)
		}
		if node.PeerCount != 2 {
			t.Fatalf("node %s peer count = %d, want 2", node.NodeID, node.PeerCount)
		}
		if node.InitialSummaryCount != 1 || node.FinalSummaryCount != 3 || node.WatermarkCount != 3 {
			t.Fatalf("node %s summary counts = %#v", node.NodeID, node)
		}
		if node.AppliedDiffCount != 2 || node.DiffItemCount != 2 {
			t.Fatalf("node %s diff counts = %#v", node.NodeID, node)
		}
		if node.TTLMergePosture == "" {
			t.Fatalf("node %s missing TTL posture", node.NodeID)
		}
	}
	if !report.RawMAVLinkExclusion.RejectedBySummaryIndex {
		t.Fatalf("raw MAVLink exclusion = %#v", report.RawMAVLinkExclusion)
	}
	requireAssertion(t, report.Assertions, "local-mesh-runtimes")
	requireAssertion(t, report.Assertions, "watermark-diff-catch-up")
	requireAssertion(t, report.Assertions, "bounded-diff-counts")
	requireAssertion(t, report.Assertions, "ttl-merge-posture")
	requireAssertion(t, report.Assertions, "raw-mavlink-excluded")

	path := filepath.Join(t.TempDir(), "mesh-report.json")
	if err := WriteJSONReport(path, report); err != nil {
		t.Fatalf("WriteJSONReport() error = %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(report) error = %v", err)
	}
	var decoded SimpleMeshDemoReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("decode report: %v", err)
	}
	if decoded.Kind != SimpleMeshReportKind || decoded.ExpectedSummaries != 3 || len(decoded.Nodes) != 3 {
		t.Fatalf("decoded report = %#v", decoded)
	}
}
