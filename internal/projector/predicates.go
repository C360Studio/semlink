package projector

const (
	PredicateVehicleCallsign        = "robot.identity.callsign"
	PredicateVehicleSystemID        = "robot.identity.system-id"
	PredicateVehicleType            = "robot.identity.vehicle-type"
	PredicateFlightArmed            = "robot.flight.armed"
	PredicateFlightMode             = "robot.flight.mode"
	PredicateFlightStatus           = "robot.flight.status"
	PredicateLinkStatus             = "robot.link.status"
	PredicateLinkLastSeenUnixMS     = "robot.link.last-seen-unix-ms"
	PredicateTelemetrySequence      = "robot.telemetry.sequence"
	PredicateTelemetrySampleUnixMS  = "robot.telemetry.sample-unix-ms"
	PredicateBatteryRemainingPct    = "robot.power.battery-remaining-pct"
	PredicateBatteryVoltageMV       = "robot.power.voltage-mv"
	PredicatePositionLatitudeDeg    = "robot.position.latitude-deg"
	PredicatePositionLongitudeDeg   = "robot.position.longitude-deg"
	PredicatePositionAltitudeM      = "robot.position.altitude-m"
	PredicatePositionGroundSpeedMS  = "robot.position.ground-speed-mps"
	PredicatePositionHeadingDeg     = "robot.position.heading-deg"
	PredicateAlertKind              = "robot.alert.kind"
	PredicateAlertSeverity          = "robot.alert.severity"
	PredicateAlertActive            = "robot.alert.active"
	PredicateAlertSubject           = "robot.alert.subject"
	PredicateAlertMessage           = "robot.alert.message"
	PredicateAlertRaisedUnixMS      = "robot.alert.raised-unix-ms"
	PredicateCommandTarget          = "robot.command.target"
	PredicateCommandVerb            = "robot.command.verb"
	PredicateCommandStatus          = "robot.command.status"
	PredicateCommandRequestedUnixMS = "robot.command.requested-unix-ms"
)

const (
	SourceMAVLink   = "semlink.mavlink.decoder"
	SourceProjector = "semlink.projector"
	SourceOperator  = "semlink.gcs.operator"
)
