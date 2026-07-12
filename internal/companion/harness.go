package companion

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/c360studio/semlink/internal/mavlink"
	"github.com/c360studio/semlink/internal/projector"
)

var defaultStart = time.Unix(0, 0).UTC()

type VehicleSnapshot = projector.VehicleStatePayload
type AlertSnapshot = projector.AlertPayload

type HarnessConfig struct {
	Nodes           int
	NodeIDPrefix    string
	Start           time.Time
	ProjectorConfig projector.Config
}

type Harness struct {
	start time.Time
	nodes []*Node
}

type Node struct {
	id        string
	sim       *mavlink.Simulator
	projector *projector.Projector
	graph     *LocalGraph
}

type StepResult struct {
	At    time.Time      `json:"at"`
	Nodes []NodeSnapshot `json:"nodes"`
}

type NodeSnapshot struct {
	NodeID        string            `json:"node_id"`
	Frames        int               `json:"frames"`
	Decoded       int               `json:"decoded"`
	Projected     int               `json:"projected"`
	GraphEntities int               `json:"graph_entities"`
	EntityIDs     []string          `json:"entity_ids"`
	Vehicles      []VehicleSnapshot `json:"vehicles"`
	Alerts        []AlertSnapshot   `json:"alerts"`
}

func NewHarness(cfg HarnessConfig) (*Harness, error) {
	if cfg.Nodes <= 0 {
		cfg.Nodes = 1
	}
	if cfg.NodeIDPrefix == "" {
		cfg.NodeIDPrefix = "boat"
	}
	if cfg.Start.IsZero() {
		cfg.Start = defaultStart
	}

	nodes := make([]*Node, 0, cfg.Nodes)
	for i := 0; i < cfg.Nodes; i++ {
		nodes = append(nodes, &Node{
			id:        fmt.Sprintf("%s-%03d", cfg.NodeIDPrefix, i+1),
			sim:       mavlink.NewSimulatorWithStart(1, cfg.Start),
			projector: projector.New(cfg.ProjectorConfig),
			graph:     NewLocalGraph(),
		})
	}
	return &Harness{start: cfg.Start, nodes: nodes}, nil
}

func (h *Harness) Step(ctx context.Context, elapsed time.Duration) (StepResult, error) {
	return h.StepAt(ctx, h.start.Add(elapsed))
}

func (h *Harness) StepAt(ctx context.Context, at time.Time) (StepResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := StepResult{
		At:    at,
		Nodes: make([]NodeSnapshot, 0, len(h.nodes)),
	}
	for _, node := range h.nodes {
		if err := ctx.Err(); err != nil {
			return StepResult{}, err
		}
		snapshot, err := node.step(ctx, at)
		if err != nil {
			return StepResult{}, err
		}
		result.Nodes = append(result.Nodes, snapshot)
	}
	return result, nil
}

func (n *Node) step(ctx context.Context, at time.Time) (NodeSnapshot, error) {
	frames, err := n.sim.Next(at)
	if err != nil {
		return NodeSnapshot{}, fmt.Errorf("%s simulator tick: %w", n.id, err)
	}

	snapshot := NodeSnapshot{
		NodeID: n.id,
		Frames: len(frames),
	}
	for _, frame := range frames {
		if err := ctx.Err(); err != nil {
			return NodeSnapshot{}, err
		}
		msg, err := mavlink.DecodeMessage(frame.Bytes)
		if err != nil {
			return NodeSnapshot{}, fmt.Errorf("%s decode %s: %w", n.id, frame.Subject, err)
		}
		snapshot.Decoded++
		projections := n.projector.Apply(msg, frame.EmittedAt)
		snapshot.Projected += len(projections)
		n.graph.Apply(projections)
	}

	vehicles, alerts := n.projector.Snapshot()
	snapshot.Vehicles = vehicles
	snapshot.Alerts = alerts
	snapshot.GraphEntities = n.graph.Len()
	snapshot.EntityIDs = n.graph.EntityIDs()
	return snapshot, nil
}

type LocalGraph struct {
	mu          sync.RWMutex
	projections map[string]projector.Projection
}

func NewLocalGraph() *LocalGraph {
	return &LocalGraph{projections: make(map[string]projector.Projection)}
}

func (g *LocalGraph) Apply(projections []projector.Projection) {
	g.mu.Lock()
	defer g.mu.Unlock()

	for _, projection := range projections {
		if projection.Entity == nil {
			continue
		}
		g.projections[projection.Entity.ID] = cloneProjection(projection)
	}
}

func (g *LocalGraph) Len() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.projections)
}

func (g *LocalGraph) EntityIDs() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	ids := make([]string, 0, len(g.projections))
	for id := range g.projections {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (g *LocalGraph) Snapshot() map[string]projector.Projection {
	g.mu.RLock()
	defer g.mu.RUnlock()

	out := make(map[string]projector.Projection, len(g.projections))
	for id, projection := range g.projections {
		out[id] = cloneProjection(projection)
	}
	return out
}

func cloneProjection(projection projector.Projection) projector.Projection {
	out := projection
	if projection.Entity != nil {
		out.Entity = projection.Entity.Clone()
	}
	if projection.Triples != nil {
		out.Triples = append(projection.Triples[:0:0], projection.Triples...)
	}
	return out
}
