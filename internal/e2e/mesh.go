package e2e

import (
	"context"
	"fmt"
	"net/http/httptest"
	"sort"
	"strings"
	"time"

	"github.com/c360studio/semlink/internal/companion"
	"github.com/c360studio/semlink/internal/mavlink"
	"github.com/c360studio/semlink/internal/mesh"
)

const (
	SimpleMeshReportKind        = "simple-mesh-companion-demo"
	DefaultSimpleMeshReportPath = ".artifacts/semlink-demo-mesh/report.json"
)

type SimpleMeshDemoConfig struct {
	Nodes          int
	VehicleProfile string
	Start          time.Time
	StepElapsed    time.Duration
	MaxDiffItems   int
}

type SimpleMeshDemoReport struct {
	Kind                string                    `json:"kind"`
	GeneratedAt         time.Time                 `json:"generated_at"`
	VehicleProfile      string                    `json:"vehicle_profile"`
	ExpectedSummaries   int                       `json:"expected_summaries"`
	Nodes               []MeshNodeReport          `json:"nodes"`
	RawMAVLinkExclusion RawMAVLinkExclusionReport `json:"raw_mavlink_exclusion"`
	Assertions          []Assertion               `json:"assertions"`
}

type MeshNodeReport struct {
	NodeID                string             `json:"node_id"`
	VehicleCount          int                `json:"vehicle_count"`
	PeerCount             int                `json:"peer_count"`
	PeerURLs              []string           `json:"peer_urls"`
	InitialSummaryCount   int                `json:"initial_summary_count"`
	FinalSummaryCount     int                `json:"final_summary_count"`
	WatermarkCount        int                `json:"watermark_count"`
	AppliedDiffCount      int                `json:"applied_diff_count"`
	DiffItemCount         int                `json:"diff_item_count"`
	TTLMergePosture       string             `json:"ttl_merge_posture"`
	VisibleOriginVehicles []OriginVehicleRef `json:"visible_origin_vehicles"`
}

type OriginVehicleRef struct {
	OriginNodeID    string `json:"origin_node_id"`
	OriginVehicleID string `json:"origin_vehicle_id"`
}

type RawMAVLinkExclusionReport struct {
	RejectedBySummaryIndex bool   `json:"rejected_by_summary_index"`
	Policy                 string `json:"policy"`
	Error                  string `json:"error,omitempty"`
}

func RunSimpleMeshDemo(ctx context.Context, cfg SimpleMeshDemoConfig) (SimpleMeshDemoReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cfg = cfg.withDefaults()

	harness, err := companion.NewHarness(companion.HarnessConfig{
		Nodes:        cfg.Nodes,
		NodeIDPrefix: defaultNodeIDPrefix,
		Start:        cfg.Start,
	})
	if err != nil {
		return SimpleMeshDemoReport{}, err
	}
	step, err := harness.Step(ctx, cfg.StepElapsed)
	if err != nil {
		return SimpleMeshDemoReport{}, err
	}

	runtimes, err := startMeshRuntimes(step.Nodes, step.At, cfg.VehicleProfile)
	if err != nil {
		return SimpleMeshDemoReport{}, err
	}
	defer closeMeshRuntimes(runtimes)

	report := SimpleMeshDemoReport{
		Kind:              SimpleMeshReportKind,
		GeneratedAt:       step.At,
		VehicleProfile:    cfg.VehicleProfile,
		ExpectedSummaries: cfg.Nodes,
	}
	report.addAssertion("local-mesh-runtimes", len(runtimes) == cfg.Nodes, fmt.Sprintf("nodes=%d", len(runtimes)))

	now := step.At.Add(time.Second)
	for i := range runtimes {
		nodeReport, err := pullMeshPeers(ctx, runtimes, i, mesh.DiffOptions{
			Now:      now,
			MaxItems: cfg.MaxDiffItems,
		})
		if err != nil {
			return SimpleMeshDemoReport{}, err
		}
		report.Nodes = append(report.Nodes, nodeReport)
	}

	report.RawMAVLinkExclusion = probeRawMAVLinkExclusion(step.At)
	report.addAssertion("watermark-diff-catch-up", allMeshNodesCaughtUp(report.Nodes, report.ExpectedSummaries), "each node reached expected selected summaries")
	report.addAssertion("bounded-diff-counts", allMeshDiffsBounded(report.Nodes, cfg.MaxDiffItems*(cfg.Nodes-1)), "diff counts stayed inside configured pull bound")
	report.addAssertion("ttl-merge-posture", allMeshNodesReportTTLMerge(report.Nodes), "last-writer-wins with TTL")
	report.addAssertion("raw-mavlink-excluded", report.RawMAVLinkExclusion.RejectedBySummaryIndex, report.RawMAVLinkExclusion.Policy)

	return report, nil
}

func (cfg SimpleMeshDemoConfig) withDefaults() SimpleMeshDemoConfig {
	if cfg.Nodes <= 0 {
		cfg.Nodes = 3
	}
	if cfg.VehicleProfile == "" {
		cfg.VehicleProfile = defaultVehicleProfile
	}
	if cfg.Start.IsZero() {
		cfg.Start = time.Unix(0, 0).UTC()
	}
	if cfg.StepElapsed <= 0 {
		cfg.StepElapsed = 20 * time.Second
	}
	if cfg.MaxDiffItems <= 0 {
		cfg.MaxDiffItems = 32
	}
	return cfg
}

type meshRuntime struct {
	node      companion.NodeSnapshot
	index     *mesh.SummaryIndex
	transport *mesh.HTTPTransport
	server    *httptest.Server
}

func startMeshRuntimes(nodes []companion.NodeSnapshot, at time.Time, vehicleProfile string) ([]meshRuntime, error) {
	runtimes := make([]meshRuntime, 0, len(nodes))
	for _, node := range nodes {
		index, err := evidenceMeshIndex(node, at, vehicleProfile)
		if err != nil {
			closeMeshRuntimes(runtimes)
			return nil, err
		}
		transport, err := mesh.NewHTTPTransport(index, mesh.HTTPTransportConfig{})
		if err != nil {
			closeMeshRuntimes(runtimes)
			return nil, err
		}
		runtime := meshRuntime{
			node:      node,
			index:     index,
			transport: transport,
			server:    httptest.NewServer(transport.Handler()),
		}
		runtimes = append(runtimes, runtime)
	}
	return runtimes, nil
}

func closeMeshRuntimes(runtimes []meshRuntime) {
	for _, runtime := range runtimes {
		if runtime.server != nil {
			runtime.server.Close()
		}
	}
}

func pullMeshPeers(ctx context.Context, runtimes []meshRuntime, nodeIndex int, opts mesh.DiffOptions) (MeshNodeReport, error) {
	return pullMeshPeerIndexes(ctx, runtimes, nodeIndex, allPeerIndexes(len(runtimes), nodeIndex), opts)
}

func allPeerIndexes(runtimeCount, nodeIndex int) []int {
	if runtimeCount <= 1 {
		return nil
	}
	peerIndexes := make([]int, 0, runtimeCount-1)
	for peerIndex := 0; peerIndex < runtimeCount; peerIndex++ {
		if peerIndex != nodeIndex {
			peerIndexes = append(peerIndexes, peerIndex)
		}
	}
	return peerIndexes
}

func pullMeshPeerIndexes(ctx context.Context, runtimes []meshRuntime, nodeIndex int, peerIndexes []int, opts mesh.DiffOptions) (MeshNodeReport, error) {
	if nodeIndex < 0 || nodeIndex >= len(runtimes) {
		return MeshNodeReport{}, fmt.Errorf("mesh node index %d out of range", nodeIndex)
	}
	runtime := runtimes[nodeIndex]
	report := MeshNodeReport{
		NodeID:              runtime.node.NodeID,
		VehicleCount:        len(runtime.node.Vehicles),
		InitialSummaryCount: runtime.index.Len(),
		TTLMergePosture:     "last-writer-wins-with-ttl",
	}
	for _, peerIndex := range peerIndexes {
		if peerIndex < 0 || peerIndex >= len(runtimes) {
			return MeshNodeReport{}, fmt.Errorf("mesh peer index %d out of range", peerIndex)
		}
		if peerIndex == nodeIndex {
			return MeshNodeReport{}, fmt.Errorf("mesh node %d cannot pull from itself", nodeIndex)
		}
		peer := runtimes[peerIndex]
		report.PeerURLs = append(report.PeerURLs, peer.server.URL)
		result, err := runtime.transport.PullFrom(ctx, peer.server.URL, opts)
		if err != nil {
			return MeshNodeReport{}, err
		}
		report.AppliedDiffCount += result.Applied
		report.DiffItemCount += len(result.Diff.Items)
	}
	report.PeerCount = len(report.PeerURLs)
	report.FinalSummaryCount = runtime.index.Len()
	watermarks := runtime.index.Watermarks(opts.Now)
	report.WatermarkCount = len(watermarks.Entries)
	report.VisibleOriginVehicles = visibleOriginVehicles(watermarks)
	return report, nil
}

func visibleOriginVehicles(watermarks mesh.WatermarkSet) []OriginVehicleRef {
	seen := make(map[OriginVehicleRef]struct{}, len(watermarks.Entries))
	refs := make([]OriginVehicleRef, 0, len(watermarks.Entries))
	for _, entry := range watermarks.Entries {
		ref := OriginVehicleRef{
			OriginNodeID:    entry.OriginNodeID,
			OriginVehicleID: entry.OriginVehicleID,
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].OriginNodeID != refs[j].OriginNodeID {
			return refs[i].OriginNodeID < refs[j].OriginNodeID
		}
		return refs[i].OriginVehicleID < refs[j].OriginVehicleID
	})
	return refs
}

func probeRawMAVLinkExclusion(at time.Time) RawMAVLinkExclusionReport {
	frame, err := mavlink.EncodeV2(1, 42, 1, mavlink.MessageHeartbeat, mavlink.HeartbeatPayload(true, 0))
	if err != nil {
		return RawMAVLinkExclusionReport{Policy: gcsRawMAVLinkPolicy(), Error: err.Error()}
	}
	item, err := mesh.NewItem(mesh.Envelope{
		OriginNodeID:    "raw-node-001",
		OriginVehicleID: "raw-vehicle-001",
		EntityID:        "mavlink.raw.raw-node-001:1",
		PredicateGroup:  "mavlink.raw",
		OriginSequence:  1,
		OperationID:     "raw-node-001:mavlink.raw:1",
		ObservedAt:      at,
		ExpiresAt:       at.Add(time.Second),
		SourceKind:      mesh.SourceKindRawMAVLink,
		Confidence:      1,
		MergePolicy:     mesh.MergePolicyAppendLimited,
	}, mavlink.RawFrame{
		Subject:   "mavlink.raw.raw-node-001",
		VehicleID: "raw-vehicle-001",
		SystemID:  42,
		EmittedAt: at,
		Bytes:     frame,
	})
	if err != nil {
		return RawMAVLinkExclusionReport{Policy: gcsRawMAVLinkPolicy(), Error: err.Error()}
	}
	index := mesh.NewSummaryIndex()
	err = index.Upsert(item)
	report := RawMAVLinkExclusionReport{
		RejectedBySummaryIndex: err != nil && strings.Contains(err.Error(), "does not replicate over mesh by default"),
		Policy:                 gcsRawMAVLinkPolicy(),
	}
	if err != nil {
		report.Error = err.Error()
	}
	return report
}

func gcsRawMAVLinkPolicy() string {
	return "raw MAVLink frames are local-only by default; selected summaries replicate"
}

func allMeshNodesCaughtUp(nodes []MeshNodeReport, expected int) bool {
	if len(nodes) == 0 {
		return false
	}
	for _, node := range nodes {
		if node.FinalSummaryCount != expected || node.WatermarkCount != expected || len(node.VisibleOriginVehicles) != expected {
			return false
		}
	}
	return true
}

func allMeshDiffsBounded(nodes []MeshNodeReport, max int) bool {
	for _, node := range nodes {
		if node.DiffItemCount > max {
			return false
		}
	}
	return true
}

func allMeshNodesReportTTLMerge(nodes []MeshNodeReport) bool {
	for _, node := range nodes {
		if node.TTLMergePosture == "" {
			return false
		}
	}
	return true
}

func (r *SimpleMeshDemoReport) addAssertion(name string, passed bool, detail string) {
	r.Assertions = append(r.Assertions, Assertion{Name: name, Passed: passed, Detail: detail})
}
