package semstreams

import (
	"strings"
	"testing"

	"github.com/c360studio/semlink/internal/cop"
	"github.com/c360studio/semlink/internal/projector"
	"github.com/c360studio/semlink/internal/rules"
	"github.com/c360studio/semstreams/pkg/projection"
	"github.com/c360studio/semstreams/vocabulary"
)

func TestCanonicalPredicateRenameMapIsCompleteAndRegistered(t *testing.T) {
	renames := map[string]string{
		"robot.identity.system_id":          projector.PredicateVehicleSystemID,
		"robot.identity.vehicle_type":       projector.PredicateVehicleType,
		"robot.link.last_seen_unix_ms":      projector.PredicateLinkLastSeenUnixMS,
		"robot.telemetry.sample_unix_ms":    projector.PredicateTelemetrySampleUnixMS,
		"robot.power.battery_remaining_pct": projector.PredicateBatteryRemainingPct,
		"robot.power.voltage_mv":            projector.PredicateBatteryVoltageMV,
		"robot.position.latitude_deg":       projector.PredicatePositionLatitudeDeg,
		"robot.position.longitude_deg":      projector.PredicatePositionLongitudeDeg,
		"robot.position.altitude_m":         projector.PredicatePositionAltitudeM,
		"robot.position.ground_speed_mps":   projector.PredicatePositionGroundSpeedMS,
		"robot.position.heading_deg":        projector.PredicatePositionHeadingDeg,
		"robot.alert.raised_unix_ms":        projector.PredicateAlertRaisedUnixMS,
		"robot.command.requested_unix_ms":   projector.PredicateCommandRequestedUnixMS,
		"cop.identity.cot_uid":              cop.PredicateCOTUID,
		"cop.kind":                          cop.PredicateKind,
		"cop.label":                         cop.PredicateLabel,
		"cop.description":                   cop.PredicateDescription,
		"cop.message.sender_uid":            cop.PredicateMessageSenderUID,
		"cop.message.sender_entity":         cop.PredicateMessageSenderEntity,
		"cop.last_seen_unix_ms":             cop.PredicateLastSeenUnixMS,
		"cop.position.latitude_deg":         cop.PredicatePositionLatitudeDeg,
		"cop.position.longitude_deg":        cop.PredicatePositionLongitudeDeg,
		"cop.position.altitude_m":           cop.PredicatePositionAltitudeM,
		"cop.position.heading_deg":          cop.PredicatePositionHeadingDeg,
		"cop.position.speed_mps":            cop.PredicatePositionSpeedMS,
		"rule.trace.rule_id":                rules.PredicateTraceRuleID,
		"rule.trace.rule_version":           rules.PredicateTraceRuleVersion,
		"rule.trace.suggested_action":       rules.PredicateTraceSuggestedAction,
		"rule.trace.execution_posture":      rules.PredicateTraceExecutionPosture,
		"rule.trace.input_count":            rules.PredicateTraceInputCount,
		"rule.trace.input_hash":             rules.PredicateTraceInputHash,
		"rule.trace.input_facts_json":       rules.PredicateTraceInputFactsJSON,
		"rule.trace.fired_unix_ms":          rules.PredicateTraceFiredUnixMS,
	}

	if got, want := len(renames), 33; got != want {
		t.Fatalf("rename count = %d, want %d", got, want)
	}
	if err := projection.ValidateContracts(ProjectionContracts()); err != nil {
		t.Fatalf("ValidateContracts: %v", err)
	}
	for old, current := range renames {
		t.Run(old, func(t *testing.T) {
			if strings.Contains(current, "_") || strings.Count(current, ".") != 2 {
				t.Fatalf("canonical predicate = %q", current)
			}
			if err := vocabulary.RequireDeclaredPredicate(current); err != nil {
				t.Fatalf("predicate %q is not registered: %v", current, err)
			}
		})
	}
}

func TestContractValidationRejectsUnregisteredAndNoncanonicalPredicates(t *testing.T) {
	for _, predicate := range []string{"semlink.test.not-registered", "semlink.test.not_canonical"} {
		t.Run(predicate, func(t *testing.T) {
			contract := projection.Contract{
				Name:          "invalid-test",
				EntityPattern: "c360.semlink.robotics.fleet.drone.*",
				Groups: []projection.PredicateGroup{{
					Name: "current", Mode: projection.ModeReconcile, Predicates: []string{predicate},
				}},
			}
			if err := contract.Validate(); err == nil {
				t.Fatalf("Contract.Validate accepted %q", predicate)
			}
		})
	}
}
