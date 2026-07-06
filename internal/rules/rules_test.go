package rules

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/c360studio/semlink/internal/projector"
)

func TestLowBatteryPeerHoldRuleFiresFromMeshVisiblePeerCoverage(t *testing.T) {
	evaluatedAt := time.Date(2026, 7, 6, 14, 0, 0, 0, time.UTC)
	localEntity := "c360.semlink.robotics.fleet.drone.uav-001"
	peerEntity := "c360.semlink.robotics.fleet.drone.uav-002"
	input := EvaluationInput{
		NodeID:      "boat-alpha",
		VehicleID:   "sys-001",
		EvaluatedAt: evaluatedAt,
		Facts: []InputFact{
			batteryFact(InputScopeLocal, localEntity, 18, evaluatedAt.Add(-time.Second)),
			batteryFact(InputScopeMesh, peerEntity, 82, evaluatedAt.Add(-2*time.Second)),
		},
	}

	trace, fired, err := LowBatteryPeerHoldRule{}.Evaluate(input)
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if !fired {
		t.Fatal("Evaluate() fired = false, want true")
	}
	if trace.RuleID != RuleLowBatteryPeerHoldID || trace.RuleVersion != RuleLowBatteryPeerHoldVersion {
		t.Fatalf("trace rule = %q/%q", trace.RuleID, trace.RuleVersion)
	}
	if trace.TargetEntity != localEntity {
		t.Fatalf("trace target = %q, want %q", trace.TargetEntity, localEntity)
	}
	if trace.Decision != "suggest-hold-for-peer-coverage" || trace.SuggestedAction != "hold-position" {
		t.Fatalf("trace decision/action = %q/%q", trace.Decision, trace.SuggestedAction)
	}
	if trace.ExecutionPosture != ExecutionPostureObserveOnly {
		t.Fatalf("trace posture = %q, want observe-only", trace.ExecutionPosture)
	}

	projection, err := trace.Projection()
	if err != nil {
		t.Fatalf("Projection() error = %v", err)
	}
	triples := triplesByPredicate(projection.Triples)
	if triples[PredicateTraceTarget] != localEntity {
		t.Fatalf("target triple = %#v", triples[PredicateTraceTarget])
	}
	if triples[PredicateTraceInputCount] != 2 {
		t.Fatalf("input count triple = %#v", triples[PredicateTraceInputCount])
	}

	var facts []InputFact
	if err := json.Unmarshal([]byte(triples[PredicateTraceInputFactsJSON].(string)), &facts); err != nil {
		t.Fatalf("input facts JSON triple unmarshal error = %v", err)
	}
	if len(facts) != 2 || facts[0].Scope != InputScopeLocal || facts[1].Scope != InputScopeMesh {
		t.Fatalf("trace input facts = %#v", facts)
	}
}

func TestLowBatteryPeerHoldRuleDoesNotFireWithoutMeshPeerCoverage(t *testing.T) {
	evaluatedAt := time.Date(2026, 7, 6, 14, 0, 0, 0, time.UTC)
	trace, fired, err := LowBatteryPeerHoldRule{}.Evaluate(EvaluationInput{
		NodeID:      "boat-alpha",
		VehicleID:   "sys-001",
		EvaluatedAt: evaluatedAt,
		Facts: []InputFact{
			batteryFact(InputScopeLocal, "c360.semlink.robotics.fleet.drone.uav-001", 18, evaluatedAt.Add(-time.Second)),
		},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if fired || trace != nil {
		t.Fatalf("Evaluate() trace/fired = %#v/%v, want nil/false", trace, fired)
	}
}

func TestLowBatteryPeerHoldRuleDoesNotFireWhenLocalBatteryHealthy(t *testing.T) {
	evaluatedAt := time.Date(2026, 7, 6, 14, 0, 0, 0, time.UTC)
	trace, fired, err := LowBatteryPeerHoldRule{}.Evaluate(EvaluationInput{
		NodeID:      "boat-alpha",
		VehicleID:   "sys-001",
		EvaluatedAt: evaluatedAt,
		Facts: []InputFact{
			batteryFact(InputScopeLocal, "c360.semlink.robotics.fleet.drone.uav-001", 72, evaluatedAt.Add(-time.Second)),
			batteryFact(InputScopeMesh, "c360.semlink.robotics.fleet.drone.uav-002", 82, evaluatedAt.Add(-2*time.Second)),
		},
	})
	if err != nil {
		t.Fatalf("Evaluate() error = %v", err)
	}
	if fired || trace != nil {
		t.Fatalf("Evaluate() trace/fired = %#v/%v, want nil/false", trace, fired)
	}
}

func batteryFact(scope InputScope, subject string, value any, observedAt time.Time) InputFact {
	source := projector.SourceMAVLink
	if scope == InputScopeMesh {
		source = "semlink.mesh.summary"
	}
	return InputFact{
		Scope:      scope,
		Subject:    subject,
		Predicate:  projector.PredicateBatteryRemainingPct,
		Object:     value,
		Source:     source,
		ObservedAt: observedAt,
		Confidence: 1,
	}
}
