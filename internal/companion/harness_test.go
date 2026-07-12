package companion

import (
	"context"
	"testing"
	"time"
)

func TestHarnessStepsDeterministicNodesWithIsolatedLocalGraphs(t *testing.T) {
	start := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	cfg := HarnessConfig{
		Nodes: 2,
		Start: start,
	}

	harness, err := NewHarness(cfg)
	if err != nil {
		t.Fatalf("NewHarness() error = %v", err)
	}
	if harness.nodes[0].graph == harness.nodes[1].graph {
		t.Fatal("expected each companion node to own an isolated local graph")
	}

	got, err := harness.Step(context.Background(), time.Second)
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}

	again, err := NewHarness(cfg)
	if err != nil {
		t.Fatalf("NewHarness() second run error = %v", err)
	}
	want, err := again.Step(context.Background(), time.Second)
	if err != nil {
		t.Fatalf("Step() second run error = %v", err)
	}

	if len(got.Nodes) != 2 {
		t.Fatalf("Step() returned %d node snapshots, want 2", len(got.Nodes))
	}
	if len(want.Nodes) != len(got.Nodes) {
		t.Fatalf("second Step() returned %d node snapshots, want %d", len(want.Nodes), len(got.Nodes))
	}

	for i, node := range got.Nodes {
		if node.NodeID == "" {
			t.Fatalf("node %d has empty NodeID", i)
		}
		if node.GraphEntities != 1 {
			t.Fatalf("node %s graph entity count = %d, want 1", node.NodeID, node.GraphEntities)
		}
		if len(node.Vehicles) != 1 {
			t.Fatalf("node %s vehicle count = %d, want 1", node.NodeID, len(node.Vehicles))
		}
		if len(node.EntityIDs) != 1 {
			t.Fatalf("node %s entity ID count = %d, want 1", node.NodeID, len(node.EntityIDs))
		}

		repeated := want.Nodes[i]
		if repeated.NodeID != node.NodeID {
			t.Fatalf("repeat node %d ID = %q, want %q", i, repeated.NodeID, node.NodeID)
		}
		assertVehicleEqual(t, repeated.Vehicles[0], node.Vehicles[0])
	}
}

func TestHarnessDerivesNodeLocalAlerts(t *testing.T) {
	start := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	harness, err := NewHarness(HarnessConfig{
		Nodes: 2,
		Start: start,
	})
	if err != nil {
		t.Fatalf("NewHarness() error = %v", err)
	}

	got, err := harness.Step(context.Background(), 20*time.Second)
	if err != nil {
		t.Fatalf("Step() error = %v", err)
	}

	for _, node := range got.Nodes {
		if len(node.Vehicles) != 1 {
			t.Fatalf("node %s vehicle count = %d, want 1", node.NodeID, len(node.Vehicles))
		}
		if len(node.Alerts) != 1 {
			t.Fatalf("node %s alert count = %d, want 1", node.NodeID, len(node.Alerts))
		}
		if node.Alerts[0].Severity != "warning" {
			t.Fatalf("node %s alert severity = %q, want warning", node.NodeID, node.Alerts[0].Severity)
		}
		if node.GraphEntities != 2 {
			t.Fatalf("node %s graph entity count = %d, want 2", node.NodeID, node.GraphEntities)
		}
	}
}

func assertVehicleEqual(t *testing.T, got, want VehicleSnapshot) {
	t.Helper()

	if got.ID != want.ID {
		t.Fatalf("ID = %q, want %q", got.ID, want.ID)
	}
	if got.LatitudeDeg != want.LatitudeDeg {
		t.Fatalf("LatitudeDeg = %f, want %f", got.LatitudeDeg, want.LatitudeDeg)
	}
	if got.LongitudeDeg != want.LongitudeDeg {
		t.Fatalf("LongitudeDeg = %f, want %f", got.LongitudeDeg, want.LongitudeDeg)
	}
	if got.BatteryRemaining != want.BatteryRemaining {
		t.Fatalf("BatteryRemaining = %d, want %d", got.BatteryRemaining, want.BatteryRemaining)
	}
	if !got.LastSeen.Equal(want.LastSeen) {
		t.Fatalf("LastSeen = %s, want %s", got.LastSeen, want.LastSeen)
	}
}
