package projector

import (
	"github.com/c360studio/semstreams/pkg/ownership"
	"github.com/c360studio/semstreams/pkg/projection"
	"github.com/c360studio/semstreams/vocabulary"
)

const Owner = "semlink.gcs.projector"

var (
	VehicleTelemetryType = mustType("mavlink", "vehicle_state", "v1")
	AlertType            = mustType("gcs", "alert", "v1")
	CommandType          = mustType("gcs", "command_intent", "v1")
)

// Contracts declares the graph footprint this product owns when writing into
// SemStreams. Telemetry current state is signal-profiled; alerts and command
// intents are control-profiled.
func Contracts() []projection.Contract {
	return []projection.Contract{
		{
			Name:            VehicleTelemetryType.String(),
			MessageType:     VehicleTelemetryType.String(),
			EntityPattern:   "c360.semlink.robotics.fleet.drone.*",
			IndexingProfile: vocabulary.IndexingProfileSignal,
			Groups: []projection.PredicateGroup{{
				Mode: ownership.ModeReplaceOwned,
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
				Mode: ownership.ModeReplaceOwned,
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
				Mode: ownership.ModeReplaceOwned,
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
