package projector

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/c360studio/semlink/internal/mavlink"
)

type Config struct {
	LowBatteryThreshold int
	LostLinkAfter       time.Duration
}

func DefaultConfig() Config {
	return Config{
		LowBatteryThreshold: 25,
		LostLinkAfter:       4 * time.Second,
	}
}

type Projector struct {
	mu     sync.RWMutex
	cfg    Config
	states map[uint8]*vehicleAccumulator
	alerts map[string]AlertPayload
}

type vehicleAccumulator struct {
	EntityID            string
	Callsign            string
	SystemID            uint8
	Sequence            uint8
	Armed               bool
	Mode                string
	FlightStatus        string
	LinkStatus          string
	LastSeen            time.Time
	BatteryRemaining    int
	VoltageBatteryMV    int
	LatitudeDeg         float64
	LongitudeDeg        float64
	AltitudeM           float64
	GroundSpeedMS       float64
	HeadingDeg          float64
	TelemetrySampleTime time.Time
}

func New(cfg Config) *Projector {
	if cfg.LowBatteryThreshold <= 0 {
		cfg.LowBatteryThreshold = DefaultConfig().LowBatteryThreshold
	}
	if cfg.LostLinkAfter <= 0 {
		cfg.LostLinkAfter = DefaultConfig().LostLinkAfter
	}
	return &Projector{
		cfg:    cfg,
		states: make(map[uint8]*vehicleAccumulator),
		alerts: make(map[string]AlertPayload),
	}
}

func VehicleEntityID(systemID uint8) string {
	return fmt.Sprintf("c360.semlink.robotics.fleet.drone.uav-%03d", systemID)
}

func AlertEntityID(kind string, systemID uint8) string {
	return fmt.Sprintf("c360.semlink.robotics.fleet.alert.%s-uav-%03d", safeToken(kind), systemID)
}

func CommandEntityID(verb string, systemID uint8, requestedAt time.Time) string {
	return fmt.Sprintf("c360.semlink.robotics.fleet.command.%s-uav-%03d-%d", safeToken(verb), systemID, requestedAt.UnixMilli())
}

// Apply updates current vehicle state from a decoded MAVLink message and returns
// the SemStreams graph writes that should be committed.
func (p *Projector) Apply(msg mavlink.Message, observedAt time.Time) []Projection {
	p.mu.Lock()
	defer p.mu.Unlock()

	state := p.stateFor(msg.System())
	state.Sequence = msg.SequenceNumber()
	state.TelemetrySampleTime = observedAt
	state.LastSeen = observedAt
	state.LinkStatus = "online"

	switch m := msg.(type) {
	case mavlink.Heartbeat:
		state.Armed = m.Armed()
		state.Mode = "guided"
		state.FlightStatus = "active"
	case mavlink.SysStatus:
		state.BatteryRemaining = int(m.BatteryRemaining)
		state.VoltageBatteryMV = int(m.VoltageBatteryMV)
	case mavlink.GlobalPositionInt:
		state.LatitudeDeg = m.LatitudeDeg()
		state.LongitudeDeg = m.LongitudeDeg()
		state.AltitudeM = m.AltitudeM()
		state.GroundSpeedMS = m.GroundSpeedMS()
		state.HeadingDeg = float64(m.HeadingCDeg) / 100
	default:
		return nil
	}

	projections := []Projection{p.vehicleProjection(state)}
	if alert, ok := p.lowBatteryAlert(state, observedAt); ok {
		p.alerts[alert.ID] = alert
		projections = append(projections, projectionFromPayload(&alert, AlertType, observedAt))
	}
	return projections
}

// CheckLinkTimeouts derives lost-link alerts from current state. It is separate
// from Apply because silence is the signal.
func (p *Projector) CheckLinkTimeouts(now time.Time) []Projection {
	p.mu.Lock()
	defer p.mu.Unlock()

	var projections []Projection
	for _, state := range p.states {
		if state.LastSeen.IsZero() || now.Sub(state.LastSeen) <= p.cfg.LostLinkAfter {
			continue
		}
		state.LinkStatus = "lost"
		state.TelemetrySampleTime = now
		projections = append(projections, p.vehicleProjection(state))
		alert := AlertPayload{
			ID:            AlertEntityID("lost-link", state.SystemID),
			Kind:          "lost-link",
			Severity:      "critical",
			Active:        true,
			SubjectEntity: state.EntityID,
			Message:       fmt.Sprintf("%s has not produced telemetry for %s", state.Callsign, now.Sub(state.LastSeen).Round(time.Second)),
			RaisedAt:      now,
		}
		p.alerts[alert.ID] = alert
		projections = append(projections, projectionFromPayload(&alert, AlertType, now))
	}
	return projections
}

func (p *Projector) Snapshot() ([]VehicleStatePayload, []AlertPayload) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	vehicles := make([]VehicleStatePayload, 0, len(p.states))
	for _, state := range p.states {
		vehicles = append(vehicles, p.payload(state))
	}
	sort.Slice(vehicles, func(i, j int) bool {
		return vehicles[i].SystemID < vehicles[j].SystemID
	})

	alerts := make([]AlertPayload, 0, len(p.alerts))
	for _, alert := range p.alerts {
		alerts = append(alerts, alert)
	}
	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].RaisedAt.After(alerts[j].RaisedAt)
	})
	return vehicles, alerts
}

func (p *Projector) stateFor(systemID uint8) *vehicleAccumulator {
	if state, ok := p.states[systemID]; ok {
		return state
	}
	state := &vehicleAccumulator{
		EntityID:         VehicleEntityID(systemID),
		Callsign:         fmt.Sprintf("UAV-%03d", systemID),
		SystemID:         systemID,
		FlightStatus:     "initializing",
		LinkStatus:       "online",
		BatteryRemaining: 100,
	}
	p.states[systemID] = state
	return state
}

func (p *Projector) vehicleProjection(state *vehicleAccumulator) Projection {
	payload := p.payload(state)
	return projectionFromPayload(&payload, VehicleTelemetryType, payload.TelemetrySampleTime)
}

func (p *Projector) payload(state *vehicleAccumulator) VehicleStatePayload {
	return VehicleStatePayload{
		ID:                  state.EntityID,
		Callsign:            state.Callsign,
		SystemID:            state.SystemID,
		Sequence:            state.Sequence,
		Armed:               state.Armed,
		Mode:                state.Mode,
		FlightStatus:        state.FlightStatus,
		LinkStatus:          state.LinkStatus,
		LastSeen:            state.LastSeen,
		BatteryRemaining:    state.BatteryRemaining,
		VoltageBatteryMV:    state.VoltageBatteryMV,
		LatitudeDeg:         state.LatitudeDeg,
		LongitudeDeg:        state.LongitudeDeg,
		AltitudeM:           state.AltitudeM,
		GroundSpeedMS:       state.GroundSpeedMS,
		HeadingDeg:          state.HeadingDeg,
		TelemetrySampleTime: state.TelemetrySampleTime,
	}
}

func (p *Projector) lowBatteryAlert(state *vehicleAccumulator, now time.Time) (AlertPayload, bool) {
	if state.BatteryRemaining <= 0 || state.BatteryRemaining > p.cfg.LowBatteryThreshold {
		return AlertPayload{}, false
	}
	return AlertPayload{
		ID:            AlertEntityID("low-battery", state.SystemID),
		Kind:          "low-battery",
		Severity:      "warning",
		Active:        true,
		SubjectEntity: state.EntityID,
		Message:       fmt.Sprintf("%s battery is %d%%", state.Callsign, state.BatteryRemaining),
		RaisedAt:      now,
	}, true
}

func safeToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if ok {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
