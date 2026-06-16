package cop

const (
	PredicateCOTUID               = "cop.identity.cot_uid"
	PredicateKind                 = "cop.kind"
	PredicateCallsign             = "cop.identity.callsign"
	PredicateLabel                = "cop.label"
	PredicateDescription          = "cop.description"
	PredicateMessageText          = "cop.message.text"
	PredicateMessageSenderUID     = "cop.message.sender_uid"
	PredicateMessageSenderEntity  = "cop.message.sender_entity"
	PredicateLastSeenUnixMS       = "cop.last_seen_unix_ms"
	PredicatePositionLatitudeDeg  = "cop.position.latitude_deg"
	PredicatePositionLongitudeDeg = "cop.position.longitude_deg"
	PredicatePositionAltitudeM    = "cop.position.altitude_m"
	PredicatePositionHeadingDeg   = "cop.position.heading_deg"
	PredicatePositionSpeedMS      = "cop.position.speed_mps"
)

const (
	SourceCoT = "semlink.cot"
	SourceCOP = "semlink.cop"
)
