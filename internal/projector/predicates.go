package projector

const (
	PredicateVehicleCallsign        = "robot.identity.callsign"
	PredicateVehicleSystemID        = "robot.identity.system_id"
	PredicateVehicleType            = "robot.identity.vehicle_type"
	PredicateFlightArmed            = "robot.flight.armed"
	PredicateFlightMode             = "robot.flight.mode"
	PredicateFlightStatus           = "robot.flight.status"
	PredicateLinkStatus             = "robot.link.status"
	PredicateLinkLastSeenUnixMS     = "robot.link.last_seen_unix_ms"
	PredicateTelemetrySequence      = "robot.telemetry.sequence"
	PredicateTelemetrySampleUnixMS  = "robot.telemetry.sample_unix_ms"
	PredicateBatteryRemainingPct    = "robot.power.battery_remaining_pct"
	PredicateBatteryVoltageMV       = "robot.power.voltage_mv"
	PredicatePositionLatitudeDeg    = "robot.position.latitude_deg"
	PredicatePositionLongitudeDeg   = "robot.position.longitude_deg"
	PredicatePositionAltitudeM      = "robot.position.altitude_m"
	PredicatePositionGroundSpeedMS  = "robot.position.ground_speed_mps"
	PredicatePositionHeadingDeg     = "robot.position.heading_deg"
	PredicateAlertKind              = "robot.alert.kind"
	PredicateAlertSeverity          = "robot.alert.severity"
	PredicateAlertActive            = "robot.alert.active"
	PredicateAlertSubject           = "robot.alert.subject"
	PredicateAlertMessage           = "robot.alert.message"
	PredicateAlertRaisedUnixMS      = "robot.alert.raised_unix_ms"
	PredicateCommandTarget          = "robot.command.target"
	PredicateCommandVerb            = "robot.command.verb"
	PredicateCommandStatus          = "robot.command.status"
	PredicateCommandRequestedUnixMS = "robot.command.requested_unix_ms"
)

const (
	SourceMAVLink   = "semlink.mavlink.decoder"
	SourceProjector = "semlink.projector"
	SourceOperator  = "semlink.gcs.operator"
)
