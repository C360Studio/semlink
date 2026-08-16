package rules

import (
	"github.com/c360studio/semstreams/pkg/projection"
	"github.com/c360studio/semstreams/vocabulary"
)

const TraceGroup = "trace-current"

func Contracts() []projection.Contract {
	RegisterVocabulary()
	return []projection.Contract{{
		Name:            TraceType.String(),
		MessageType:     TraceType.String(),
		EntityPattern:   "c360.semlink.robotics.fleet.rule.*",
		IndexingProfile: vocabulary.IndexingProfileTrace,
		Groups: []projection.PredicateGroup{{
			Name: TraceGroup,
			Mode: projection.ModeReconcile,
			Predicates: []string{
				PredicateTraceRuleID,
				PredicateTraceRuleVersion,
				PredicateTraceNode,
				PredicateTraceVehicle,
				PredicateTraceTarget,
				PredicateTraceDecision,
				PredicateTraceSuggestedAction,
				PredicateTraceExecutionPosture,
				PredicateTraceInputCount,
				PredicateTraceInputHash,
				PredicateTraceInputFactsJSON,
				PredicateTraceFiredUnixMS,
			},
		}},
	}}
}

func RegisterVocabulary() {
	for _, predicate := range []string{
		PredicateTraceRuleID, PredicateTraceRuleVersion, PredicateTraceNode,
		PredicateTraceVehicle, PredicateTraceTarget, PredicateTraceDecision,
		PredicateTraceSuggestedAction, PredicateTraceExecutionPosture,
		PredicateTraceInputCount, PredicateTraceInputHash, PredicateTraceInputFactsJSON,
		PredicateTraceFiredUnixMS,
	} {
		vocabulary.RegisterPredicate(vocabulary.PredicateMetadata{Name: predicate})
	}
}
