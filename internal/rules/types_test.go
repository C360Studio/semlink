package rules

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/vocabulary"
)

func TestNewTracePayloadDefaultsObserveOnlyAndProjectsInputEvidence(t *testing.T) {
	evaluatedAt := time.Date(2026, 7, 6, 13, 0, 0, 0, time.UTC)
	input := EvaluationInput{
		NodeID:      "boat-alpha",
		VehicleID:   "sys-001",
		EvaluatedAt: evaluatedAt,
		Facts: []InputFact{
			{
				Scope:      InputScopeLocal,
				Subject:    "c360.semlink.robotics.fleet.drone.uav-001",
				Predicate:  "robot.power.battery-remaining-pct",
				Object:     18,
				Source:     "semlink.mavlink.decoder",
				ObservedAt: evaluatedAt.Add(-time.Second),
				Confidence: 1,
			},
			{
				Scope:      InputScopeMesh,
				Subject:    "c360.semlink.robotics.fleet.drone.uav-002",
				Predicate:  "robot.position.ground-speed-mps",
				Object:     0.8,
				Source:     "semlink.mesh.summary",
				ObservedAt: evaluatedAt.Add(-2 * time.Second),
				Confidence: 0.9,
			},
		},
	}

	trace, err := NewTracePayload(input, EvaluationResult{
		RuleID:          "low-battery-spacing",
		RuleVersion:     "v1",
		TargetEntity:    "c360.semlink.robotics.fleet.drone.uav-001",
		Decision:        "suggest-hold",
		SuggestedAction: "hold-position",
	})
	if err != nil {
		t.Fatalf("NewTracePayload() error = %v", err)
	}

	if trace.ExecutionPosture != ExecutionPostureObserveOnly {
		t.Fatalf("ExecutionPosture = %q, want observe-only", trace.ExecutionPosture)
	}
	if !strings.HasPrefix(trace.InputHash, "sha256:") {
		t.Fatalf("InputHash = %q", trace.InputHash)
	}
	if !strings.HasPrefix(trace.ID, "c360.semlink.robotics.fleet.rule.trace-low-battery-spacing-boat-alpha-") {
		t.Fatalf("trace ID = %q", trace.ID)
	}

	projection, err := trace.Projection()
	if err != nil {
		t.Fatalf("Projection() error = %v", err)
	}
	if projection.Entity.ID != trace.ID {
		t.Fatalf("projection entity = %q, want %q", projection.Entity.ID, trace.ID)
	}
	if projection.Entity.MessageType != TraceType {
		t.Fatalf("projection message type = %s, want %s", projection.Entity.MessageType, TraceType)
	}
	if projection.IndexingProfile != vocabulary.IndexingProfileTrace {
		t.Fatalf("projection profile = %q, want trace", projection.IndexingProfile)
	}
	if projection.Contract != TraceType.String() || projection.Group != TraceGroup {
		t.Fatalf("projection binding = %q/%q", projection.Contract, projection.Group)
	}

	triples := triplesByPredicate(projection.Triples)
	if triples[PredicateTraceDecision] != "suggest-hold" {
		t.Fatalf("decision triple = %#v", triples[PredicateTraceDecision])
	}
	if triples[PredicateTraceExecutionPosture] != string(ExecutionPostureObserveOnly) {
		t.Fatalf("posture triple = %#v", triples[PredicateTraceExecutionPosture])
	}
	if triples[PredicateTraceInputCount] != 2 {
		t.Fatalf("input count triple = %#v", triples[PredicateTraceInputCount])
	}
	if triples[PredicateTraceInputHash] != trace.InputHash {
		t.Fatalf("input hash triple = %#v, want %s", triples[PredicateTraceInputHash], trace.InputHash)
	}
	if triples[PredicateTraceFiredUnixMS] != evaluatedAt.UnixMilli() {
		t.Fatalf("fired timestamp triple = %#v", triples[PredicateTraceFiredUnixMS])
	}

	var facts []InputFact
	if err := json.Unmarshal([]byte(triples[PredicateTraceInputFactsJSON].(string)), &facts); err != nil {
		t.Fatalf("input facts JSON triple unmarshal error = %v", err)
	}
	if len(facts) != 2 || facts[1].Scope != InputScopeMesh {
		t.Fatalf("input facts JSON = %#v", facts)
	}
}

func TestTracePayloadRejectsMissingInputs(t *testing.T) {
	_, err := NewTracePayload(EvaluationInput{
		NodeID:      "boat-alpha",
		EvaluatedAt: time.Date(2026, 7, 6, 13, 0, 0, 0, time.UTC),
	}, EvaluationResult{
		RuleID:          "low-battery-spacing",
		Decision:        "suggest-hold",
		SuggestedAction: "hold-position",
	})
	if err == nil {
		t.Fatal("NewTracePayload() error = nil, want missing input fact error")
	}
	if !strings.Contains(err.Error(), "at least one input fact") {
		t.Fatalf("NewTracePayload() error = %v", err)
	}
}

func TestTraceContractDeclaresTraceProfileAndPredicates(t *testing.T) {
	contracts := Contracts()
	if got, want := len(contracts), 1; got != want {
		t.Fatalf("contract count = %d, want %d", got, want)
	}
	contract := contracts[0]
	if contract.MessageType != TraceType.String() {
		t.Fatalf("contract message type = %q, want %q", contract.MessageType, TraceType.String())
	}
	if contract.IndexingProfile != vocabulary.IndexingProfileTrace {
		t.Fatalf("contract indexing profile = %q, want trace", contract.IndexingProfile)
	}
	if contract.Groups[0].Name != TraceGroup || contract.Groups[0].Mode != "reconcile" {
		t.Fatalf("trace group = %#v", contract.Groups[0])
	}
	predicates := make(map[string]struct{})
	for _, predicate := range contract.Groups[0].Predicates {
		predicates[predicate] = struct{}{}
	}
	for _, predicate := range []string{
		PredicateTraceDecision,
		PredicateTraceSuggestedAction,
		PredicateTraceExecutionPosture,
		PredicateTraceInputHash,
		PredicateTraceInputFactsJSON,
		PredicateTraceFiredUnixMS,
	} {
		if _, ok := predicates[predicate]; !ok {
			t.Fatalf("contract missing predicate %q", predicate)
		}
	}
}

func triplesByPredicate(triples []message.Triple) map[string]any {
	out := make(map[string]any, len(triples))
	for _, triple := range triples {
		out[triple.Predicate] = triple.Object
	}
	return out
}
