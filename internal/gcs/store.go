package gcs

import (
	"sort"
	"sync"
	"time"

	"github.com/c360studio/semlink/internal/projector"
)

type Snapshot struct {
	GeneratedAt time.Time     `json:"generated_at"`
	Vehicles    []VehicleView `json:"vehicles"`
	Alerts      []AlertView   `json:"alerts"`
	Commands    []CommandView `json:"commands"`
	Metrics     MetricsView   `json:"metrics"`
}

type VehicleView struct {
	EntityID         string    `json:"entity_id"`
	Callsign         string    `json:"callsign"`
	SystemID         uint8     `json:"system_id"`
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
	mu       sync.RWMutex
	started  time.Time
	vehicles map[string]VehicleView
	alerts   map[string]AlertView
	commands []CommandView
	metrics  MetricsView
}

func NewStore(natsURL string, embedded bool) *Store {
	now := time.Now()
	return &Store{
		started:  now,
		vehicles: make(map[string]VehicleView),
		alerts:   make(map[string]AlertView),
		metrics: MetricsView{
			NATSURL:            natsURL,
			SemStreamsEmbedded: embedded,
			StartedAt:          now,
		},
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
	metrics := s.metrics
	uptime := time.Since(s.started).Seconds()
	if uptime > 0 {
		metrics.FramesPerSecond = float64(metrics.RawFrames) / uptime
		metrics.GraphWritesPerSec = float64(metrics.GraphWrites) / uptime
	}

	return Snapshot{
		GeneratedAt: time.Now(),
		Vehicles:    vehicles,
		Alerts:      alerts,
		Commands:    commands,
		Metrics:     metrics,
	}
}
