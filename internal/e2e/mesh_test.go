package e2e

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/companion"
	"github.com/c360studio/semlink/internal/mesh"
	"github.com/c360studio/semlink/internal/projector"
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
		assertVisibleOriginVehicles(t, node, originVehicleRefsForReports(report.Nodes))
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

func TestSimpleMeshPerspectiveMatrix(t *testing.T) {
	if !probeRawMAVLinkExclusion(time.Date(2026, 7, 9, 17, 5, 0, 0, time.UTC)).RejectedBySummaryIndex {
		t.Fatal("raw MAVLink exclusion probe failed")
	}

	t.Run("full mesh converges from every node perspective", func(t *testing.T) {
		fixture := newMeshPerspectiveFixture(t)
		reports := fixture.pullRound(t, [][]int{
			{1, 2},
			{0, 2},
			{0, 1},
		})

		assertAllNodesSee(t, reports, fixture.allOriginVehicles)
	})

	t.Run("line topology converges after repeated adjacent pulls", func(t *testing.T) {
		fixture := newMeshPerspectiveFixture(t)
		line := [][]int{
			{1},
			{0, 2},
			{1},
		}
		fixture.pullRound(t, line)
		reports := fixture.pullRound(t, line)

		assertAllNodesSee(t, reports, fixture.allOriginVehicles)
	})

	t.Run("partition heal does not overclaim isolated visibility", func(t *testing.T) {
		fixture := newMeshPerspectiveFixture(t)
		partitioned := fixture.pullRound(t, [][]int{
			{1},
			{0},
			nil,
		})

		assertVisibleOriginVehicles(t, partitioned[0], fixture.originVehicles(0, 1))
		assertVisibleOriginVehicles(t, partitioned[1], fixture.originVehicles(0, 1))
		assertVisibleOriginVehicles(t, partitioned[2], fixture.originVehicles(2))

		healedLine := [][]int{
			{1},
			{0, 2},
			{1},
		}
		fixture.pullRound(t, healedLine)
		healed := fixture.pullRound(t, healedLine)

		assertAllNodesSee(t, healed, fixture.allOriginVehicles)
	})

	t.Run("late joiner catches up without rewriting the deployment model", func(t *testing.T) {
		fixture := newMeshPerspectiveFixture(t)
		beforeJoin := fixture.pullRound(t, [][]int{
			{1},
			{0},
			nil,
		})

		assertVisibleOriginVehicles(t, beforeJoin[0], fixture.originVehicles(0, 1))
		assertVisibleOriginVehicles(t, beforeJoin[1], fixture.originVehicles(0, 1))
		assertVisibleOriginVehicles(t, beforeJoin[2], fixture.originVehicles(2))

		joined := fixture.pullRound(t, [][]int{
			{1, 2},
			{0, 2},
			{0, 1},
		})

		assertAllNodesSee(t, joined, fixture.allOriginVehicles)
	})
}

type meshPerspectiveFixture struct {
	runtimes          []meshRuntime
	now               time.Time
	allOriginVehicles []OriginVehicleRef
}

func newMeshPerspectiveFixture(t *testing.T) meshPerspectiveFixture {
	t.Helper()

	start := time.Date(2026, 7, 9, 17, 10, 0, 0, time.UTC)
	harness, err := companion.NewHarness(companion.HarnessConfig{
		Nodes:        3,
		NodeIDPrefix: defaultNodeIDPrefix,
		Start:        start,
	})
	if err != nil {
		t.Fatalf("NewHarness() error = %v", err)
	}
	step, err := harness.Step(context.Background(), 20*time.Second)
	if err != nil {
		t.Fatalf("Harness.Step() error = %v", err)
	}
	runtimes, err := startMeshRuntimes(step.Nodes, step.At, "ardurover")
	if err != nil {
		t.Fatalf("startMeshRuntimes() error = %v", err)
	}
	t.Cleanup(func() {
		closeMeshRuntimes(runtimes)
	})

	fixture := meshPerspectiveFixture{
		runtimes: runtimes,
		now:      step.At.Add(time.Second),
	}
	fixture.allOriginVehicles = fixture.originVehicles(0, 1, 2)
	return fixture
}

func (f meshPerspectiveFixture) pullRound(t *testing.T, peerPlan [][]int) []MeshNodeReport {
	t.Helper()
	if len(peerPlan) != len(f.runtimes) {
		t.Fatalf("peer plan length = %d, want %d", len(peerPlan), len(f.runtimes))
	}

	reports := make([]MeshNodeReport, len(f.runtimes))
	for nodeIndex, peerIndexes := range peerPlan {
		report, err := pullMeshPeerIndexes(context.Background(), f.runtimes, nodeIndex, peerIndexes, mesh.DiffOptions{
			Now:      f.now,
			MaxItems: 8,
		})
		if err != nil {
			t.Fatalf("pullMeshPeerIndexes(node=%d, peers=%v) error = %v", nodeIndex, peerIndexes, err)
		}
		if report.VehicleCount != 1 {
			t.Fatalf("node %s vehicle count = %d, want 1", report.NodeID, report.VehicleCount)
		}
		if report.DiffItemCount > 8*len(peerIndexes) {
			t.Fatalf("node %s diff item count = %d, max pull bound = %d", report.NodeID, report.DiffItemCount, 8*len(peerIndexes))
		}
		reports[nodeIndex] = report
	}
	return reports
}

func (f meshPerspectiveFixture) originVehicles(indexes ...int) []OriginVehicleRef {
	refs := make([]OriginVehicleRef, 0, len(indexes))
	for _, index := range indexes {
		runtime := f.runtimes[index]
		for _, vehicle := range runtime.node.Vehicles {
			refs = append(refs, OriginVehicleRef{
				OriginNodeID:    runtime.node.NodeID,
				OriginVehicleID: vehicle.ID,
			})
		}
	}
	return refs
}

func assertAllNodesSee(t *testing.T, reports []MeshNodeReport, want []OriginVehicleRef) {
	t.Helper()
	for _, report := range reports {
		if report.FinalSummaryCount != len(want) || report.WatermarkCount != len(want) {
			t.Fatalf("node %s counts = %#v, want %d summaries", report.NodeID, report, len(want))
		}
		assertVisibleOriginVehicles(t, report, want)
	}
}

func assertVisibleOriginVehicles(t *testing.T, report MeshNodeReport, want []OriginVehicleRef) {
	t.Helper()
	if !reflect.DeepEqual(report.VisibleOriginVehicles, want) {
		t.Fatalf("node %s visible origin vehicles = %#v, want %#v", report.NodeID, report.VisibleOriginVehicles, want)
	}
}

func originVehicleRefsForReports(nodes []MeshNodeReport) []OriginVehicleRef {
	refs := make([]OriginVehicleRef, 0, len(nodes))
	for _, node := range nodes {
		refs = append(refs, OriginVehicleRef{
			OriginNodeID:    node.NodeID,
			OriginVehicleID: projector.VehicleEntityID(1),
		})
	}
	return refs
}
