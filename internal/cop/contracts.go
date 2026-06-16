package cop

import (
	"github.com/c360studio/semstreams/pkg/ownership"
	"github.com/c360studio/semstreams/pkg/projection"
	"github.com/c360studio/semstreams/vocabulary"
)

const Owner = "semlink.cop.projector"

func Contracts() []projection.Contract {
	return []projection.Contract{
		{
			Name:            OperatorType.String(),
			MessageType:     OperatorType.String(),
			EntityPattern:   "c360.semlink.cop.operator.position.*",
			IndexingProfile: vocabulary.IndexingProfileSignal,
			Groups: []projection.PredicateGroup{{
				Mode: ownership.ModeReplaceOwned,
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
				Mode: ownership.ModeReplaceOwned,
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
				Mode: ownership.ModeReplaceOwned,
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
