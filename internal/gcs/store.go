package gcs

import (
	"sort"
	"sync"
	"time"

	"github.com/c360studio/semlink/internal/commandgate"
	"github.com/c360studio/semlink/internal/cop"
	"github.com/c360studio/semlink/internal/projector"
	"github.com/c360studio/semlink/internal/rules"
)

type Snapshot struct {
	GeneratedAt  time.Time         `json:"generated_at"`
	Vehicles     []VehicleView     `json:"vehicles"`
	Alerts       []AlertView       `json:"alerts"`
	Commands     []CommandView     `json:"commands"`
	Operators    []cop.View        `json:"operators"`
	Markers      []cop.View        `json:"markers"`
	Messages     []cop.View        `json:"messages"`
	RuleTraces   []RuleTraceView   `json:"rule_traces"`
	CommandGates []CommandGateView `json:"command_gates"`
	Metrics      MetricsView       `json:"metrics"`
}

type VehicleView struct {
	EntityID         string    `json:"entity_id"`
	Callsign         string    `json:"callsign"`
	SystemID         uint8     `json:"system_id"`
	VehicleType      string    `json:"vehicle_type"`
	Armed            bool      `json:"armed"`
	Mode             string    `json:"mode"`
	FlightStatus     string    `json:"flight_status"`
	LinkStatus       string    `json:"link_status"`
	BatteryRemaining int       `json:"battery_remaining"`
	VoltageBatteryMV int       `json:"voltage_battery_mv"`
	LatitudeDeg      float64   `json:"latitude_deg"`
	LongitudeDeg     float64   `json:"longitude_deg"`
	AltitudeM        float64   `json:"altitude_m"`
	GroundSpeedMS    float64   `json:"ground_speed_mps"`
	HeadingDeg       float64   `json:"heading_deg"`
	Sequence         uint8     `json:"sequence"`
	LastSeen         time.Time `json:"last_seen"`
	GraphRevision    uint64    `json:"graph_revision"`
	IndexingProfile  string    `json:"indexing_profile"`
}

type AlertView struct {
	EntityID      string    `json:"entity_id"`
	Kind          string    `json:"kind"`
	Severity      string    `json:"severity"`
	Active        bool      `json:"active"`
	SubjectEntity string    `json:"subject_entity"`
	Message       string    `json:"message"`
	RaisedAt      time.Time `json:"raised_at"`
	GraphRevision uint64    `json:"graph_revision"`
}

type CommandView struct {
	EntityID      string    `json:"entity_id"`
	TargetEntity  string    `json:"target_entity"`
	Verb          string    `json:"verb"`
	Status        string    `json:"status"`
	RequestedAt   time.Time `json:"requested_at"`
	GraphRevision uint64    `json:"graph_revision"`
}

type MetricsView struct {
	RawFrames          int64     `json:"raw_frames"`
	DecodedFrames      int64     `json:"decoded_frames"`
	DecodeErrors       int64     `json:"decode_errors"`
	ProjectedWrites    int64     `json:"projected_writes"`
	GraphWrites        int64     `json:"graph_writes"`
	GraphErrors        int64     `json:"graph_errors"`
	RawPublishErrors   int64     `json:"raw_publish_errors"`
	BufferDrops        int64     `json:"buffer_drops"`
	BufferSize         int       `json:"buffer_size"`
	BufferCapacity     int       `json:"buffer_capacity"`
	FramesPerSecond    float64   `json:"frames_per_second"`
	GraphWritesPerSec  float64   `json:"graph_writes_per_second"`
	LastGraphLatencyMS float64   `json:"last_graph_latency_ms"`
	NATSURL            string    `json:"nats_url"`
	SemStreamsEmbedded bool      `json:"semstreams_embedded"`
	StartedAt          time.Time `json:"started_at"`
}

type Store struct {
	mu           sync.RWMutex
	started      time.Time
	vehicles     map[string]VehicleView
	alerts       map[string]AlertView
	commands     []CommandView
	operators    map[string]cop.View
	markers      map[string]cop.View
	messages     map[string]cop.View
	ruleTraces   []RuleTraceView
	commandGates []CommandGateView
	metrics      MetricsView
}

func NewStore(natsURL string, embedded bool) *Store {
	now := time.Now()
	return &Store{
		started:   now,
		vehicles:  make(map[string]VehicleView),
		alerts:    make(map[string]AlertView),
		operators: make(map[string]cop.View),
		markers:   make(map[string]cop.View),
		messages:  make(map[string]cop.View),
		metrics: MetricsView{
			NATSURL:            natsURL,
			SemStreamsEmbedded: embedded,
			StartedAt:          now,
		},
	}
}

func (s *Store) ApplyCOPView(view cop.View) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch view.Kind {
	case cop.KindOperator:
		existing := s.operators[view.EntityID]
		view.GraphRevision = existing.GraphRevision
		s.operators[view.EntityID] = view
	case cop.KindMarker:
		existing := s.markers[view.EntityID]
		view.GraphRevision = existing.GraphRevision
		s.markers[view.EntityID] = view
	case cop.KindMessage:
		existing := s.messages[view.EntityID]
		view.GraphRevision = existing.GraphRevision
		s.messages[view.EntityID] = view
	}
}

func (s *Store) ApplyProjectorSnapshot(vehicles []projector.VehicleStatePayload, alerts []projector.AlertPayload) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, v := range vehicles {
		existing := s.vehicles[v.ID]
		s.vehicles[v.ID] = VehicleView{
			EntityID:         v.ID,
			Callsign:         v.Callsign,
			SystemID:         v.SystemID,
			VehicleType:      v.VehicleType,
			Armed:            v.Armed,
			Mode:             v.Mode,
			FlightStatus:     v.FlightStatus,
			LinkStatus:       v.LinkStatus,
			BatteryRemaining: v.BatteryRemaining,
			VoltageBatteryMV: v.VoltageBatteryMV,
			LatitudeDeg:      v.LatitudeDeg,
			LongitudeDeg:     v.LongitudeDeg,
			AltitudeM:        v.AltitudeM,
			GroundSpeedMS:    v.GroundSpeedMS,
			HeadingDeg:       v.HeadingDeg,
			Sequence:         v.Sequence,
			LastSeen:         v.LastSeen,
			GraphRevision:    existing.GraphRevision,
			IndexingProfile:  "signal",
		}
	}
	for _, alert := range alerts {
		existing := s.alerts[alert.ID]
		s.alerts[alert.ID] = AlertView{
			EntityID:      alert.ID,
			Kind:          alert.Kind,
			Severity:      alert.Severity,
			Active:        alert.Active,
			SubjectEntity: alert.SubjectEntity,
			Message:       alert.Message,
			RaisedAt:      alert.RaisedAt,
			GraphRevision: existing.GraphRevision,
		}
	}
}

func (s *Store) RecordRawFrame() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics.RawFrames++
}

func (s *Store) RecordDecodedFrame() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics.DecodedFrames++
}

func (s *Store) RecordDecodeError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics.DecodeErrors++
}

func (s *Store) RecordRawPublishError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics.RawPublishErrors++
}

func (s *Store) RecordProjectedWrite() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics.ProjectedWrites++
}

func (s *Store) RecordGraphError() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics.GraphErrors++
}

func (s *Store) RecordGraphWrite(entityID string, revision uint64, latency time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics.GraphWrites++
	s.metrics.LastGraphLatencyMS = float64(latency.Microseconds()) / 1000
	if vehicle, ok := s.vehicles[entityID]; ok {
		vehicle.GraphRevision = revision
		s.vehicles[entityID] = vehicle
	}
	if alert, ok := s.alerts[entityID]; ok {
		alert.GraphRevision = revision
		s.alerts[entityID] = alert
	}
	for i := range s.commands {
		if s.commands[i].EntityID == entityID {
			s.commands[i].GraphRevision = revision
		}
	}
	if operator, ok := s.operators[entityID]; ok {
		operator.GraphRevision = revision
		s.operators[entityID] = operator
	}
	if marker, ok := s.markers[entityID]; ok {
		marker.GraphRevision = revision
		s.markers[entityID] = marker
	}
	if message, ok := s.messages[entityID]; ok {
		message.GraphRevision = revision
		s.messages[entityID] = message
	}
}

func (s *Store) RecordBuffer(size, capacity int, drops int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.metrics.BufferSize = size
	s.metrics.BufferCapacity = capacity
	s.metrics.BufferDrops = drops
}

func (s *Store) AddCommand(cmd CommandView) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.commands = append([]CommandView{cmd}, s.commands...)
	if len(s.commands) > 20 {
		s.commands = s.commands[:20]
	}
}

func (s *Store) RecordRuleTrace(trace *rules.TracePayload) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ruleTraces = append([]RuleTraceView{ruleTraceView(trace)}, s.ruleTraces...)
	if len(s.ruleTraces) > 50 {
		s.ruleTraces = s.ruleTraces[:50]
	}
}

func (s *Store) RecordCommandGateResult(result commandgate.TransmitResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	view := CommandGateView{
		Accepted:                   result.Accepted,
		Status:                     result.Status,
		StartedAt:                  result.StartedAt,
		PreflightAccepted:          result.Preflight.Accepted,
		SimulatorOnly:              result.Preflight.Evidence.RuntimeMode == commandgate.RuntimeModeSimulator,
		FrameCount:                 len(result.Frames),
		ACKCount:                   len(result.ACKs),
		ACKAccepted:                commandACKAccepted(result.ACKs),
		PostStateObserved:          result.PostState.Observed,
		HardwareTransmitAuthorized: false,
		HardwareBlock:              result.HardwareBlock,
	}
	if result.Preflight.Evidence.TargetEntity != "" {
		view.RuntimeMode = result.Preflight.Evidence.RuntimeMode
		view.SafetyProfile = result.Preflight.Evidence.SafetyProfile
		view.TargetEntity = result.Preflight.Evidence.TargetEntity
		view.Verb = result.Preflight.Evidence.Verb
		view.RequestedBy = result.Preflight.Evidence.RequestedBy
	}
	if result.HardwareBlock != nil {
		view.RuntimeMode = result.HardwareBlock.RuntimeMode
		view.SafetyProfile = result.HardwareBlock.SafetyProfile
		view.TargetEntity = result.HardwareBlock.TargetEntity
		view.Verb = result.HardwareBlock.Verb
		view.RequestedBy = result.HardwareBlock.RequestedBy
	}
	s.commandGates = append([]CommandGateView{view}, s.commandGates...)
	if len(s.commandGates) > 50 {
		s.commandGates = s.commandGates[:50]
	}
}

func commandACKAccepted(acks []commandgate.ACKEvidence) bool {
	for _, ack := range acks {
		if ack.Accepted() {
			return true
		}
	}
	return false
}

func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	vehicles := make([]VehicleView, 0, len(s.vehicles))
	for _, vehicle := range s.vehicles {
		vehicles = append(vehicles, vehicle)
	}
	sort.Slice(vehicles, func(i, j int) bool {
		return vehicles[i].SystemID < vehicles[j].SystemID
	})

	alerts := make([]AlertView, 0, len(s.alerts))
	for _, alert := range s.alerts {
		alerts = append(alerts, alert)
	}
	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].RaisedAt.After(alerts[j].RaisedAt)
	})

	commands := append([]CommandView(nil), s.commands...)
	ruleTraces := append([]RuleTraceView(nil), s.ruleTraces...)
	commandGates := append([]CommandGateView(nil), s.commandGates...)
	operators := make([]cop.View, 0, len(s.operators))
	for _, operator := range s.operators {
		operators = append(operators, operator)
	}
	sort.Slice(operators, func(i, j int) bool {
		return operators[i].Callsign < operators[j].Callsign
	})

	markers := make([]cop.View, 0, len(s.markers))
	for _, marker := range s.markers {
		markers = append(markers, marker)
	}
	sort.Slice(markers, func(i, j int) bool {
		return markers[i].LastSeen.After(markers[j].LastSeen)
	})

	messages := make([]cop.View, 0, len(s.messages))
	for _, message := range s.messages {
		messages = append(messages, message)
	}
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].LastSeen.After(messages[j].LastSeen)
	})

	metrics := s.metrics
	uptime := time.Since(s.started).Seconds()
	if uptime > 0 {
		metrics.FramesPerSecond = float64(metrics.RawFrames) / uptime
		metrics.GraphWritesPerSec = float64(metrics.GraphWrites) / uptime
	}

	return Snapshot{
		GeneratedAt:  time.Now(),
		Vehicles:     vehicles,
		Alerts:       alerts,
		Commands:     commands,
		Operators:    operators,
		Markers:      markers,
		Messages:     messages,
		RuleTraces:   ruleTraces,
		CommandGates: commandGates,
		Metrics:      metrics,
	}
}
