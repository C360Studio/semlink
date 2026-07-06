package rules

import (
	"encoding/json"
	"strconv"

	"github.com/c360studio/semlink/internal/projector"
)

const (
	RuleLowBatteryPeerHoldID      = "low-battery-peer-hold"
	RuleLowBatteryPeerHoldVersion = "v1"

	DefaultLowBatteryThreshold = 25
)

type LowBatteryPeerHoldRule struct {
	LowBatteryThreshold int
}

func (r LowBatteryPeerHoldRule) Evaluate(input EvaluationInput) (*TracePayload, bool, error) {
	if err := input.Validate(); err != nil {
		return nil, false, err
	}
	threshold := r.LowBatteryThreshold
	if threshold <= 0 {
		threshold = DefaultLowBatteryThreshold
	}

	localBattery, ok := findBatteryFact(input.Facts, InputScopeLocal, "", func(value float64) bool {
		return value <= float64(threshold)
	})
	if !ok {
		return nil, false, nil
	}
	_, ok = findBatteryFact(input.Facts, InputScopeMesh, localBattery.Subject, func(value float64) bool {
		return value > float64(threshold)
	})
	if !ok {
		return nil, false, nil
	}

	trace, err := NewTracePayload(input, EvaluationResult{
		RuleID:           RuleLowBatteryPeerHoldID,
		RuleVersion:      RuleLowBatteryPeerHoldVersion,
		TargetEntity:     localBattery.Subject,
		Decision:         "suggest-hold-for-peer-coverage",
		SuggestedAction:  "hold-position",
		ExecutionPosture: ExecutionPostureObserveOnly,
	})
	if err != nil {
		return nil, false, err
	}
	return trace, true, nil
}

func findBatteryFact(facts []InputFact, scope InputScope, excludeSubject string, match func(float64) bool) (InputFact, bool) {
	for _, fact := range facts {
		if fact.Scope != scope || fact.Predicate != projector.PredicateBatteryRemainingPct || fact.Subject == excludeSubject {
			continue
		}
		value, ok := numericValue(fact.Object)
		if ok && match(value) {
			return fact, true
		}
	}
	return InputFact{}, false
}

func numericValue(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	case json.Number:
		parsed, err := strconv.ParseFloat(string(v), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}
