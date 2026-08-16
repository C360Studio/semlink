package cop

const (
	PredicateCOTUID               = "cop.identity.cot-uid"
	PredicateKind                 = "cop.identity.kind"
	PredicateCallsign             = "cop.identity.callsign"
	PredicateLabel                = "cop.identity.label"
	PredicateDescription          = "cop.content.description"
	PredicateMessageText          = "cop.message.text"
	PredicateMessageSenderUID     = "cop.message.sender-uid"
	PredicateMessageSenderEntity  = "cop.message.sender-entity"
	PredicateLastSeenUnixMS       = "cop.state.last-seen-unix-ms"
	PredicatePositionLatitudeDeg  = "cop.position.latitude-deg"
	PredicatePositionLongitudeDeg = "cop.position.longitude-deg"
	PredicatePositionAltitudeM    = "cop.position.altitude-m"
	PredicatePositionHeadingDeg   = "cop.position.heading-deg"
	PredicatePositionSpeedMS      = "cop.position.speed-mps"
)

const (
	SourceCoT = "semlink.cot"
	SourceCOP = "semlink.cop"
)
