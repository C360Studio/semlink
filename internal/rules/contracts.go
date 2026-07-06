package rules

import (
	"github.com/c360studio/semstreams/pkg/ownership"
	"github.com/c360studio/semstreams/pkg/projection"
	"github.com/c360studio/semstreams/vocabulary"
)

const Owner = "semlink.rules.engine"

func Contracts() []projection.Contract {
	return []projection.Contract{{
		Name:            TraceType.String(),
		MessageType:     TraceType.String(),
		EntityPattern:   "c360.semlink.robotics.fleet.rule.*",
		IndexingProfile: vocabulary.IndexingProfileTrace,
		Groups: []projection.PredicateGroup{{
			Mode: ownership.ModeReplaceOwned,
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
