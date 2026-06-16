package cop

import (
	"errors"
	"strings"
	"time"

	"github.com/c360studio/semlink/internal/cot"
	"github.com/c360studio/semlink/internal/graphprojection"
)

type ProjectionResult struct {
	Projection graphprojection.Projection
	View       View
}

type Translator struct{}

func NewTranslator() *Translator {
	return &Translator{}
}

func (t *Translator) Apply(event cot.Event, observedAt time.Time) (ProjectionResult, bool, error) {
	if observedAt.IsZero() {
		observedAt = time.Now()
	}
	eventTime := event.Time
	if eventTime.IsZero() {
		eventTime = observedAt
	}
	switch {
	case cot.IsAirTrackType(event.Type), cot.IsAlertType(event.Type):
		return ProjectionResult{}, false, nil
	case cot.IsOperatorType(event.Type):
		payload, err := operatorPayload(event, eventTime)
		if err != nil {
			return ProjectionResult{}, false, err
		}
		return resultFromPayload(payload, OperatorType, eventTime), true, nil
	case cot.IsGeoChatType(event.Type) || strings.TrimSpace(event.ChatText) != "":
		payload, err := messagePayload(event, eventTime)
		if err != nil {
			return ProjectionResult{}, false, err
		}
		return resultFromPayload(payload, MessageType, eventTime), true, nil
	case cot.IsMarkerType(event.Type), event.Point != nil:
		payload, err := markerPayload(event, eventTime)
		if err != nil {
			return ProjectionResult{}, false, err
		}
		return resultFromPayload(payload, MarkerType, eventTime), true, nil
	default:
		return ProjectionResult{}, false, nil
	}
}

func operatorPayload(event cot.Event, seen time.Time) (*OperatorPayload, error) {
	if event.Point == nil {
		return nil, errors.New("operator CoT event requires point")
	}
	id, err := EntityID(KindOperator, event.UID)
	if err != nil {
		return nil, err
	}
	return &OperatorPayload{
		ID:            id,
		UID:           event.UID,
		Callsign:      firstNonEmpty(event.Callsign, event.UID),
		LatitudeDeg:   event.Point.Lat,
		LongitudeDeg:  event.Point.Lon,
		AltitudeM:     event.Point.HAE,
		HeadingDeg:    event.CourseDeg,
		GroundSpeedMS: event.SpeedMPS,
		LastSeen:      seen,
	}, nil
}

func markerPayload(event cot.Event, seen time.Time) (*MarkerPayload, error) {
	if event.Point == nil {
		return nil, errors.New("marker CoT event requires point")
	}
	id, err := EntityID(KindMarker, event.UID)
	if err != nil {
		return nil, err
	}
	return &MarkerPayload{
		ID:           id,
		UID:          event.UID,
		Label:        firstNonEmpty(event.Callsign, event.UID),
		Description:  event.Remarks,
		LatitudeDeg:  event.Point.Lat,
		LongitudeDeg: event.Point.Lon,
		AltitudeM:    event.Point.HAE,
		LastSeen:     seen,
	}, nil
}

func messagePayload(event cot.Event, seen time.Time) (*MessagePayload, error) {
	text := firstNonEmpty(event.ChatText, event.Remarks)
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("message CoT event requires text")
	}
	id, err := EntityID(KindMessage, event.UID)
	if err != nil {
		return nil, err
	}
	senderUID := strings.TrimSpace(event.SenderUID)
	senderEntity := ""
	if senderUID != "" {
		senderEntity, _ = EntityID(KindOperator, senderUID)
	}
	payload := &MessagePayload{
		ID:           id,
		UID:          event.UID,
		Callsign:     firstNonEmpty(event.Callsign, event.UID),
		Text:         text,
		SenderUID:    senderUID,
		SenderEntity: senderEntity,
		LastSeen:     seen,
	}
	if event.Point != nil {
		payload.LatitudeDeg = event.Point.Lat
		payload.LongitudeDeg = event.Point.Lon
		payload.AltitudeM = event.Point.HAE
		payload.HasPosition = true
	}
	return payload, nil
}

func resultFromPayload(payload graphprojection.Payload, msgType semType, updatedAt time.Time) ProjectionResult {
	return ProjectionResult{
		Projection: graphprojection.ProjectionFromPayload(payload, msgType, updatedAt),
		View:       ViewFromPayload(payload),
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
