package cop

import (
	"github.com/c360studio/semstreams/pkg/projection"
	"github.com/c360studio/semstreams/vocabulary"
)

const (
	OperatorGroup = "operator-current"
	MarkerGroup   = "marker-current"
	MessageGroup  = "message-current"
)

func Contracts() []projection.Contract {
	RegisterVocabulary()
	return []projection.Contract{
		{
			Name:            OperatorType.String(),
			MessageType:     OperatorType.String(),
			EntityPattern:   "c360.semlink.cop.operator.position.*",
			IndexingProfile: vocabulary.IndexingProfileSignal,
			Groups: []projection.PredicateGroup{{
				Name: OperatorGroup,
				Mode: projection.ModeReconcile,
				Predicates: []string{
					PredicateCOTUID,
					PredicateKind,
					PredicateCallsign,
					PredicateLastSeenUnixMS,
					PredicatePositionLatitudeDeg,
					PredicatePositionLongitudeDeg,
					PredicatePositionAltitudeM,
					PredicatePositionHeadingDeg,
					PredicatePositionSpeedMS,
				},
			}},
		},
		{
			Name:            MarkerType.String(),
			MessageType:     MarkerType.String(),
			EntityPattern:   "c360.semlink.cop.marker.poi.*",
			IndexingProfile: vocabulary.IndexingProfileContent,
			Groups: []projection.PredicateGroup{{
				Name: MarkerGroup,
				Mode: projection.ModeReconcile,
				Predicates: []string{
					PredicateCOTUID,
					PredicateKind,
					PredicateLabel,
					PredicateDescription,
					PredicateLastSeenUnixMS,
					PredicatePositionLatitudeDeg,
					PredicatePositionLongitudeDeg,
					PredicatePositionAltitudeM,
				},
			}},
		},
		{
			Name:            MessageType.String(),
			MessageType:     MessageType.String(),
			EntityPattern:   "c360.semlink.cop.message.geochat.*",
			IndexingProfile: vocabulary.IndexingProfileContent,
			Groups: []projection.PredicateGroup{{
				Name: MessageGroup,
				Mode: projection.ModeReconcile,
				Predicates: []string{
					PredicateCOTUID,
					PredicateKind,
					PredicateCallsign,
					PredicateMessageText,
					PredicateMessageSenderUID,
					PredicateMessageSenderEntity,
					PredicateLastSeenUnixMS,
					PredicatePositionLatitudeDeg,
					PredicatePositionLongitudeDeg,
					PredicatePositionAltitudeM,
				},
			}},
		},
	}
}

func RegisterVocabulary() {
	for _, predicate := range []string{
		PredicateCOTUID, PredicateKind, PredicateCallsign, PredicateLabel,
		PredicateDescription, PredicateMessageText, PredicateMessageSenderUID,
		PredicateMessageSenderEntity, PredicateLastSeenUnixMS,
		PredicatePositionLatitudeDeg, PredicatePositionLongitudeDeg,
		PredicatePositionAltitudeM, PredicatePositionHeadingDeg, PredicatePositionSpeedMS,
	} {
		vocabulary.RegisterPredicate(vocabulary.PredicateMetadata{Name: predicate})
	}
}
