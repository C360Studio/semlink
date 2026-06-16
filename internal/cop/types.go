package cop

import (
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/c360studio/semlink/internal/graphprojection"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/c360studio/semstreams/vocabulary"
)

type semType = message.Type

var (
	OperatorType = mustType("cop", "operator_position", "v1")
	MarkerType   = mustType("cop", "marker", "v1")
	MessageType  = mustType("cop", "message", "v1")
)

type Kind string

const (
	KindOperator Kind = "operator"
	KindMarker   Kind = "marker"
	KindMessage  Kind = "message"
)

type View struct {
	Kind            Kind      `json:"kind"`
	EntityID        string    `json:"entity_id"`
	UID             string    `json:"uid"`
	Callsign        string    `json:"callsign,omitempty"`
	Label           string    `json:"label,omitempty"`
	Description     string    `json:"description,omitempty"`
	Text            string    `json:"text,omitempty"`
	SenderUID       string    `json:"sender_uid,omitempty"`
	SenderEntity    string    `json:"sender_entity,omitempty"`
	LatitudeDeg     float64   `json:"latitude_deg,omitempty"`
	LongitudeDeg    float64   `json:"longitude_deg,omitempty"`
	AltitudeM       float64   `json:"altitude_m,omitempty"`
	HeadingDeg      float64   `json:"heading_deg,omitempty"`
	GroundSpeedMS   float64   `json:"ground_speed_mps,omitempty"`
	HasPosition     bool      `json:"has_position"`
	LastSeen        time.Time `json:"last_seen"`
	GraphRevision   uint64    `json:"graph_revision"`
	IndexingProfile string    `json:"indexing_profile"`
}

type OperatorPayload struct {
	ID            string    `json:"entity_id"`
	UID           string    `json:"uid"`
	Callsign      string    `json:"callsign"`
	LatitudeDeg   float64   `json:"latitude_deg"`
	LongitudeDeg  float64   `json:"longitude_deg"`
	AltitudeM     float64   `json:"altitude_m"`
	HeadingDeg    float64   `json:"heading_deg"`
	GroundSpeedMS float64   `json:"ground_speed_mps"`
	LastSeen      time.Time `json:"last_seen"`
}

func (p *OperatorPayload) Schema() message.Type { return OperatorType }
func (p *OperatorPayload) EntityID() string     { return p.ID }
func (p *OperatorPayload) IndexingProfile() string {
	return vocabulary.IndexingProfileSignal
}
func (p *OperatorPayload) Validate() error {
	if p.ID == "" || p.UID == "" {
		return errors.New("entity id and uid are required")
	}
	return validatePosition(p.LatitudeDeg, p.LongitudeDeg)
}
func (p *OperatorPayload) MarshalJSON() ([]byte, error) {
	type alias OperatorPayload
	return json.Marshal((*alias)(p))
}
func (p *OperatorPayload) UnmarshalJSON(data []byte) error {
	type alias OperatorPayload
	return json.Unmarshal(data, (*alias)(p))
}
func (p *OperatorPayload) Triples() []message.Triple {
	now := nonZeroTime(p.LastSeen)
	return []message.Triple{
		triple(p.ID, PredicateCOTUID, p.UID, SourceCoT, now, 1),
		triple(p.ID, PredicateKind, string(KindOperator), SourceCOP, now, 1),
		triple(p.ID, PredicateCallsign, p.Callsign, SourceCoT, now, 0.9),
		triple(p.ID, PredicateLastSeenUnixMS, now.UnixMilli(), SourceCOP, now, 1),
		triple(p.ID, PredicatePositionLatitudeDeg, p.LatitudeDeg, SourceCoT, now, 1),
		triple(p.ID, PredicatePositionLongitudeDeg, p.LongitudeDeg, SourceCoT, now, 1),
		triple(p.ID, PredicatePositionAltitudeM, p.AltitudeM, SourceCoT, now, 1),
		triple(p.ID, PredicatePositionHeadingDeg, p.HeadingDeg, SourceCoT, now, 0.8),
		triple(p.ID, PredicatePositionSpeedMS, p.GroundSpeedMS, SourceCoT, now, 0.8),
	}
}

type MarkerPayload struct {
	ID           string    `json:"entity_id"`
	UID          string    `json:"uid"`
	Label        string    `json:"label"`
	Description  string    `json:"description"`
	LatitudeDeg  float64   `json:"latitude_deg"`
	LongitudeDeg float64   `json:"longitude_deg"`
	AltitudeM    float64   `json:"altitude_m"`
	LastSeen     time.Time `json:"last_seen"`
}

func (p *MarkerPayload) Schema() message.Type { return MarkerType }
func (p *MarkerPayload) EntityID() string     { return p.ID }
func (p *MarkerPayload) IndexingProfile() string {
	return vocabulary.IndexingProfileContent
}
func (p *MarkerPayload) Validate() error {
	if p.ID == "" || p.UID == "" {
		return errors.New("entity id and uid are required")
	}
	return validatePosition(p.LatitudeDeg, p.LongitudeDeg)
}
func (p *MarkerPayload) MarshalJSON() ([]byte, error) {
	type alias MarkerPayload
	return json.Marshal((*alias)(p))
}
func (p *MarkerPayload) UnmarshalJSON(data []byte) error {
	type alias MarkerPayload
	return json.Unmarshal(data, (*alias)(p))
}
func (p *MarkerPayload) Triples() []message.Triple {
	now := nonZeroTime(p.LastSeen)
	return []message.Triple{
		triple(p.ID, PredicateCOTUID, p.UID, SourceCoT, now, 1),
		triple(p.ID, PredicateKind, string(KindMarker), SourceCOP, now, 1),
		triple(p.ID, PredicateLabel, p.Label, SourceCoT, now, 0.9),
		triple(p.ID, PredicateDescription, p.Description, SourceCoT, now, 0.9),
		triple(p.ID, PredicateLastSeenUnixMS, now.UnixMilli(), SourceCOP, now, 1),
		triple(p.ID, PredicatePositionLatitudeDeg, p.LatitudeDeg, SourceCoT, now, 1),
		triple(p.ID, PredicatePositionLongitudeDeg, p.LongitudeDeg, SourceCoT, now, 1),
		triple(p.ID, PredicatePositionAltitudeM, p.AltitudeM, SourceCoT, now, 1),
	}
}

type MessagePayload struct {
	ID           string    `json:"entity_id"`
	UID          string    `json:"uid"`
	Callsign     string    `json:"callsign"`
	Text         string    `json:"text"`
	SenderUID    string    `json:"sender_uid"`
	SenderEntity string    `json:"sender_entity"`
	LatitudeDeg  float64   `json:"latitude_deg"`
	LongitudeDeg float64   `json:"longitude_deg"`
	AltitudeM    float64   `json:"altitude_m"`
	HasPosition  bool      `json:"has_position"`
	LastSeen     time.Time `json:"last_seen"`
}

func (p *MessagePayload) Schema() message.Type { return MessageType }
func (p *MessagePayload) EntityID() string     { return p.ID }
func (p *MessagePayload) IndexingProfile() string {
	return vocabulary.IndexingProfileContent
}
func (p *MessagePayload) Validate() error {
	if p.ID == "" || p.UID == "" || strings.TrimSpace(p.Text) == "" {
		return errors.New("entity id, uid, and text are required")
	}
	if p.HasPosition {
		return validatePosition(p.LatitudeDeg, p.LongitudeDeg)
	}
	return nil
}
func (p *MessagePayload) MarshalJSON() ([]byte, error) {
	type alias MessagePayload
	return json.Marshal((*alias)(p))
}
func (p *MessagePayload) UnmarshalJSON(data []byte) error {
	type alias MessagePayload
	return json.Unmarshal(data, (*alias)(p))
}
func (p *MessagePayload) Triples() []message.Triple {
	now := nonZeroTime(p.LastSeen)
	triples := []message.Triple{
		triple(p.ID, PredicateCOTUID, p.UID, SourceCoT, now, 1),
		triple(p.ID, PredicateKind, string(KindMessage), SourceCOP, now, 1),
		triple(p.ID, PredicateCallsign, p.Callsign, SourceCoT, now, 0.8),
		triple(p.ID, PredicateMessageText, p.Text, SourceCoT, now, 1),
		triple(p.ID, PredicateLastSeenUnixMS, now.UnixMilli(), SourceCOP, now, 1),
	}
	if p.SenderUID != "" {
		triples = append(triples, triple(p.ID, PredicateMessageSenderUID, p.SenderUID, SourceCoT, now, 0.9))
	}
	if p.SenderEntity != "" {
		triples = append(triples, triple(p.ID, PredicateMessageSenderEntity, p.SenderEntity, SourceCOP, now, 0.9))
	}
	if p.HasPosition {
		triples = append(triples,
			triple(p.ID, PredicatePositionLatitudeDeg, p.LatitudeDeg, SourceCoT, now, 1),
			triple(p.ID, PredicatePositionLongitudeDeg, p.LongitudeDeg, SourceCoT, now, 1),
			triple(p.ID, PredicatePositionAltitudeM, p.AltitudeM, SourceCoT, now, 1),
		)
	}
	return triples
}

func RegisterPayloads(reg *payloadregistry.Registry) error {
	for _, registration := range []*payloadregistry.Registration{
		{
			Domain:      OperatorType.Domain,
			Category:    OperatorType.Category,
			Version:     OperatorType.Version,
			Description: "SemLink COP operator position",
			Factory:     func() any { return &OperatorPayload{} },
		},
		{
			Domain:      MarkerType.Domain,
			Category:    MarkerType.Category,
			Version:     MarkerType.Version,
			Description: "SemLink COP marker",
			Factory:     func() any { return &MarkerPayload{} },
		},
		{
			Domain:      MessageType.Domain,
			Category:    MessageType.Category,
			Version:     MessageType.Version,
			Description: "SemLink COP message",
			Factory:     func() any { return &MessagePayload{} },
		},
	} {
		if err := reg.Register(registration); err != nil {
			return err
		}
	}
	return nil
}

func EntityID(kind Kind, uid string) (string, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return "", errors.New("cot uid is required")
	}
	token := strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString([]byte(uid)))
	return fmt.Sprintf("c360.semlink.cop.%s.%s.%s", kind, entityCategory(kind), token), nil
}

func entityCategory(kind Kind) string {
	switch kind {
	case KindOperator:
		return "position"
	case KindMarker:
		return "poi"
	case KindMessage:
		return "geochat"
	default:
		return "unknown"
	}
}

func ViewFromPayload(payload graphprojection.Payload) View {
	switch p := payload.(type) {
	case *OperatorPayload:
		return View{
			Kind:            KindOperator,
			EntityID:        p.ID,
			UID:             p.UID,
			Callsign:        p.Callsign,
			LatitudeDeg:     p.LatitudeDeg,
			LongitudeDeg:    p.LongitudeDeg,
			AltitudeM:       p.AltitudeM,
			HeadingDeg:      p.HeadingDeg,
			GroundSpeedMS:   p.GroundSpeedMS,
			HasPosition:     true,
			LastSeen:        nonZeroTime(p.LastSeen),
			IndexingProfile: p.IndexingProfile(),
		}
	case *MarkerPayload:
		return View{
			Kind:            KindMarker,
			EntityID:        p.ID,
			UID:             p.UID,
			Label:           p.Label,
			Description:     p.Description,
			LatitudeDeg:     p.LatitudeDeg,
			LongitudeDeg:    p.LongitudeDeg,
			AltitudeM:       p.AltitudeM,
			HasPosition:     true,
			LastSeen:        nonZeroTime(p.LastSeen),
			IndexingProfile: p.IndexingProfile(),
		}
	case *MessagePayload:
		return View{
			Kind:            KindMessage,
			EntityID:        p.ID,
			UID:             p.UID,
			Callsign:        p.Callsign,
			Text:            p.Text,
			SenderUID:       p.SenderUID,
			SenderEntity:    p.SenderEntity,
			LatitudeDeg:     p.LatitudeDeg,
			LongitudeDeg:    p.LongitudeDeg,
			AltitudeM:       p.AltitudeM,
			HasPosition:     p.HasPosition,
			LastSeen:        nonZeroTime(p.LastSeen),
			IndexingProfile: p.IndexingProfile(),
		}
	default:
		return View{}
	}
}

func mustType(domain, category, version string) semType {
	return semType{Domain: domain, Category: category, Version: version}
}

func triple(subject, predicate string, object any, source string, timestamp time.Time, confidence float64) message.Triple {
	return graphprojection.Triple(subject, predicate, object, source, timestamp, confidence)
}

func validatePosition(lat, lon float64) error {
	if math.Abs(lat) > 90 {
		return fmt.Errorf("latitude out of range: %f", lat)
	}
	if math.Abs(lon) > 180 {
		return fmt.Errorf("longitude out of range: %f", lon)
	}
	return nil
}

func nonZeroTime(t time.Time) time.Time {
	if t.IsZero() {
		return time.Now()
	}
	return t
}
