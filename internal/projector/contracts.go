package projector

import (
	"github.com/c360studio/semstreams/pkg/projection"
	"github.com/c360studio/semstreams/vocabulary"
)

const (
	VehicleTelemetryGroup = "vehicle-current"
	AlertGroup            = "alert-current"
	CommandGroup          = "command-current"
)

var (
	VehicleTelemetryType = mustType("mavlink", "vehicle_state", "v1")
	AlertType            = mustType("gcs", "alert", "v1")
	CommandType          = mustType("gcs", "command_intent", "v1")
)

// Contracts declares the graph shapes this product emits. Contracts validate
// producer intent; they do not reserve predicates or authorize writes.
func Contracts() []projection.Contract {
	RegisterVocabulary()
	return []projection.Contract{
		{
			Name:            VehicleTelemetryType.String(),
			MessageType:     VehicleTelemetryType.String(),
			EntityPattern:   "c360.semlink.robotics.fleet.drone.*",
			IndexingProfile: vocabulary.IndexingProfileSignal,
			Groups: []projection.PredicateGroup{{
				Name: VehicleTelemetryGroup,
				Mode: projection.ModeReconcile,
				Predicates: []string{
					PredicateVehicleCallsign,
					PredicateVehicleSystemID,
					PredicateVehicleType,
					PredicateFlightArmed,
					PredicateFlightMode,
					PredicateFlightStatus,
					PredicateLinkStatus,
					PredicateLinkLastSeenUnixMS,
					PredicateTelemetrySequence,
					PredicateTelemetrySampleUnixMS,
					PredicateBatteryRemainingPct,
					PredicateBatteryVoltageMV,
					PredicatePositionLatitudeDeg,
					PredicatePositionLongitudeDeg,
					PredicatePositionAltitudeM,
					PredicatePositionGroundSpeedMS,
					PredicatePositionHeadingDeg,
				},
			}},
		},
		{
			Name:            AlertType.String(),
			MessageType:     AlertType.String(),
			EntityPattern:   "c360.semlink.robotics.fleet.alert.*",
			IndexingProfile: vocabulary.IndexingProfileControl,
			Groups: []projection.PredicateGroup{{
				Name: AlertGroup,
				Mode: projection.ModeReconcile,
				Predicates: []string{
					PredicateAlertKind,
					PredicateAlertSeverity,
					PredicateAlertActive,
					PredicateAlertSubject,
					PredicateAlertMessage,
					PredicateAlertRaisedUnixMS,
				},
			}},
		},
		{
			Name:            CommandType.String(),
			MessageType:     CommandType.String(),
			EntityPattern:   "c360.semlink.robotics.fleet.command.*",
			IndexingProfile: vocabulary.IndexingProfileControl,
			Groups: []projection.PredicateGroup{{
				Name: CommandGroup,
				Mode: projection.ModeReconcile,
				Predicates: []string{
					PredicateCommandTarget,
					PredicateCommandVerb,
					PredicateCommandStatus,
					PredicateCommandRequestedUnixMS,
				},
			}},
		},
	}
}

// RegisterVocabulary declares all SemLink projector predicates before
// contract validation or mutation-client construction.
func RegisterVocabulary() {
	for _, predicate := range []string{
		PredicateVehicleCallsign, PredicateVehicleSystemID, PredicateVehicleType,
		PredicateFlightArmed, PredicateFlightMode, PredicateFlightStatus,
		PredicateLinkStatus, PredicateLinkLastSeenUnixMS, PredicateTelemetrySequence,
		PredicateTelemetrySampleUnixMS, PredicateBatteryRemainingPct, PredicateBatteryVoltageMV,
		PredicatePositionLatitudeDeg, PredicatePositionLongitudeDeg, PredicatePositionAltitudeM,
		PredicatePositionGroundSpeedMS, PredicatePositionHeadingDeg, PredicateAlertKind,
		PredicateAlertSeverity, PredicateAlertActive, PredicateAlertSubject, PredicateAlertMessage,
		PredicateAlertRaisedUnixMS, PredicateCommandTarget, PredicateCommandVerb,
		PredicateCommandStatus, PredicateCommandRequestedUnixMS,
	} {
		vocabulary.RegisterPredicate(vocabulary.PredicateMetadata{Name: predicate})
	}
}
