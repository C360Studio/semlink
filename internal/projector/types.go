package projector

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/c360studio/semlink/internal/graphprojection"
	"github.com/c360studio/semstreams/message"
	"github.com/c360studio/semstreams/payloadregistry"
	"github.com/c360studio/semstreams/vocabulary"
)

type semType = message.Type
type Projection = graphprojection.Projection

func mustType(domain, category, version string) semType {
	return semType{Domain: domain, Category: category, Version: version}
}

type VehicleStatePayload struct {
	ID                  string    `json:"entity_id"`
	Callsign            string    `json:"callsign"`
	SystemID            uint8     `json:"system_id"`
	VehicleType         string    `json:"vehicle_type"`
	Sequence            uint8     `json:"sequence"`
	Armed               bool      `json:"armed"`
	Mode                string    `json:"mode"`
	FlightStatus        string    `json:"flight_status"`
	LinkStatus          string    `json:"link_status"`
	LastSeen            time.Time `json:"last_seen"`
	BatteryRemaining    int       `json:"battery_remaining"`
	VoltageBatteryMV    int       `json:"voltage_battery_mv"`
	LatitudeDeg         float64   `json:"latitude_deg"`
	LongitudeDeg        float64   `json:"longitude_deg"`
	AltitudeM           float64   `json:"altitude_m"`
	GroundSpeedMS       float64   `json:"ground_speed_mps"`
	HeadingDeg          float64   `json:"heading_deg"`
	TelemetrySampleTime time.Time `json:"telemetry_sample_time"`
}

func (p *VehicleStatePayload) Schema() message.Type { return VehicleTelemetryType }
func (p *VehicleStatePayload) EntityID() string     { return p.ID }
func (p *VehicleStatePayload) IndexingProfile() string {
	return vocabulary.IndexingProfileSignal
}
func (p *VehicleStatePayload) Validate() error {
	if p.ID == "" {
		return errors.New("entity id is required")
	}
	if math.Abs(p.LatitudeDeg) > 90 {
		return fmt.Errorf("latitude out of range: %f", p.LatitudeDeg)
	}
	if math.Abs(p.LongitudeDeg) > 180 {
		return fmt.Errorf("longitude out of range: %f", p.LongitudeDeg)
	}
	return nil
}
func (p *VehicleStatePayload) MarshalJSON() ([]byte, error) {
	type alias VehicleStatePayload
	return json.Marshal((*alias)(p))
}
func (p *VehicleStatePayload) UnmarshalJSON(data []byte) error {
	type alias VehicleStatePayload
	return json.Unmarshal(data, (*alias)(p))
}
func (p *VehicleStatePayload) Triples() []message.Triple {
	now := p.TelemetrySampleTime
	if now.IsZero() {
		now = time.Now()
	}
	vehicleType := p.VehicleType
	if vehicleType == "" {
		vehicleType = "unknown"
	}
	return []message.Triple{
		triple(p.ID, PredicateVehicleCallsign, p.Callsign, SourceProjector, now, 1),
		triple(p.ID, PredicateVehicleSystemID, int(p.SystemID), SourceMAVLink, now, 1),
		triple(p.ID, PredicateVehicleType, vehicleType, SourceMAVLink, now, 1),
		triple(p.ID, PredicateFlightArmed, p.Armed, SourceMAVLink, now, 1),
		triple(p.ID, PredicateFlightMode, p.Mode, SourceMAVLink, now, 0.9),
		triple(p.ID, PredicateFlightStatus, p.FlightStatus, SourceMAVLink, now, 1),
		triple(p.ID, PredicateLinkStatus, p.LinkStatus, SourceProjector, now, 1),
		triple(p.ID, PredicateLinkLastSeenUnixMS, p.LastSeen.UnixMilli(), SourceProjector, now, 1),
		triple(p.ID, PredicateTelemetrySequence, int(p.Sequence), SourceMAVLink, now, 1),
		triple(p.ID, PredicateTelemetrySampleUnixMS, now.UnixMilli(), SourceMAVLink, now, 1),
		triple(p.ID, PredicateBatteryRemainingPct, p.BatteryRemaining, SourceMAVLink, now, 1),
		triple(p.ID, PredicateBatteryVoltageMV, p.VoltageBatteryMV, SourceMAVLink, now, 1),
		triple(p.ID, PredicatePositionLatitudeDeg, p.LatitudeDeg, SourceMAVLink, now, 1),
		triple(p.ID, PredicatePositionLongitudeDeg, p.LongitudeDeg, SourceMAVLink, now, 1),
		triple(p.ID, PredicatePositionAltitudeM, p.AltitudeM, SourceMAVLink, now, 1),
		triple(p.ID, PredicatePositionGroundSpeedMS, p.GroundSpeedMS, SourceMAVLink, now, 1),
		triple(p.ID, PredicatePositionHeadingDeg, p.HeadingDeg, SourceMAVLink, now, 1),
	}
}

type AlertPayload struct {
	ID            string    `json:"entity_id"`
	Kind          string    `json:"kind"`
	Severity      string    `json:"severity"`
	Active        bool      `json:"active"`
	SubjectEntity string    `json:"subject_entity"`
	Message       string    `json:"message"`
	RaisedAt      time.Time `json:"raised_at"`
}

func (p *AlertPayload) Schema() message.Type { return AlertType }
func (p *AlertPayload) EntityID() string     { return p.ID }
func (p *AlertPayload) IndexingProfile() string {
	return vocabulary.IndexingProfileControl
}
func (p *AlertPayload) Validate() error {
	if p.ID == "" || p.Kind == "" || p.SubjectEntity == "" {
		return errors.New("entity id, kind, and subject entity are required")
	}
	return nil
}
func (p *AlertPayload) MarshalJSON() ([]byte, error) {
	type alias AlertPayload
	return json.Marshal((*alias)(p))
}
func (p *AlertPayload) UnmarshalJSON(data []byte) error {
	type alias AlertPayload
	return json.Unmarshal(data, (*alias)(p))
}
func (p *AlertPayload) Triples() []message.Triple {
	now := p.RaisedAt
	if now.IsZero() {
		now = time.Now()
	}
	return []message.Triple{
		triple(p.ID, PredicateAlertKind, p.Kind, SourceProjector, now, 1),
		triple(p.ID, PredicateAlertSeverity, p.Severity, SourceProjector, now, 1),
		triple(p.ID, PredicateAlertActive, p.Active, SourceProjector, now, 1),
		triple(p.ID, PredicateAlertSubject, p.SubjectEntity, SourceProjector, now, 1),
		triple(p.ID, PredicateAlertMessage, p.Message, SourceProjector, now, 1),
		triple(p.ID, PredicateAlertRaisedUnixMS, now.UnixMilli(), SourceProjector, now, 1),
	}
}

type CommandPayload struct {
	ID           string    `json:"entity_id"`
	TargetEntity string    `json:"target_entity"`
	Verb         string    `json:"verb"`
	Status       string    `json:"status"`
	RequestedAt  time.Time `json:"requested_at"`
}

func (p *CommandPayload) Schema() message.Type { return CommandType }
func (p *CommandPayload) EntityID() string     { return p.ID }
func (p *CommandPayload) IndexingProfile() string {
	return vocabulary.IndexingProfileControl
}
func (p *CommandPayload) Validate() error {
	if p.ID == "" || p.TargetEntity == "" || p.Verb == "" {
		return errors.New("entity id, target entity, and verb are required")
	}
	return nil
}
func (p *CommandPayload) MarshalJSON() ([]byte, error) {
	type alias CommandPayload
	return json.Marshal((*alias)(p))
}
func (p *CommandPayload) UnmarshalJSON(data []byte) error {
	type alias CommandPayload
	return json.Unmarshal(data, (*alias)(p))
}
func (p *CommandPayload) Triples() []message.Triple {
	now := p.RequestedAt
	if now.IsZero() {
		now = time.Now()
	}
	return []message.Triple{
		triple(p.ID, PredicateCommandTarget, p.TargetEntity, SourceOperator, now, 1),
		triple(p.ID, PredicateCommandVerb, p.Verb, SourceOperator, now, 1),
		triple(p.ID, PredicateCommandStatus, p.Status, SourceOperator, now, 1),
		triple(p.ID, PredicateCommandRequestedUnixMS, now.UnixMilli(), SourceOperator, now, 1),
	}
}

func RegisterPayloads(reg *payloadregistry.Registry) error {
	if err := reg.Register(&payloadregistry.Registration{
		Domain:      VehicleTelemetryType.Domain,
		Category:    VehicleTelemetryType.Category,
		Version:     VehicleTelemetryType.Version,
		Description: "SemLink MAVLink projected vehicle state",
		Factory:     func() any { return &VehicleStatePayload{} },
	}); err != nil {
		return err
	}
	if err := reg.Register(&payloadregistry.Registration{
		Domain:      AlertType.Domain,
		Category:    AlertType.Category,
		Version:     AlertType.Version,
		Description: "SemLink GCS alert",
		Factory:     func() any { return &AlertPayload{} },
	}); err != nil {
		return err
	}
	return reg.Register(&payloadregistry.Registration{
		Domain:      CommandType.Domain,
		Category:    CommandType.Category,
		Version:     CommandType.Version,
		Description: "SemLink GCS operator command intent",
		Factory:     func() any { return &CommandPayload{} },
	})
}

func triple(subject, predicate string, object any, source string, timestamp time.Time, confidence float64) message.Triple {
	return graphprojection.Triple(subject, predicate, object, source, timestamp, confidence)
}

func ProjectionFromPayload(payload interface {
	message.Payload
	EntityID() string
	Triples() []message.Triple
	IndexingProfile() string
}, msgType message.Type, updatedAt time.Time) Projection {
	contract, group := projectionBinding(msgType)
	return graphprojection.ProjectionFromPayload(payload, msgType, updatedAt, contract, group)
}

func projectionBinding(msgType message.Type) (string, string) {
	switch msgType.Key() {
	case VehicleTelemetryType.Key():
		return VehicleTelemetryType.String(), VehicleTelemetryGroup
	case AlertType.Key():
		return AlertType.String(), AlertGroup
	case CommandType.Key():
		return CommandType.String(), CommandGroup
	default:
		return msgType.String(), ""
	}
}
