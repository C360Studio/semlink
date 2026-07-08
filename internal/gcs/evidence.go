package gcs

import (
	"strings"
	"time"

	"github.com/c360studio/semlink/internal/commandgate"
	"github.com/c360studio/semlink/internal/handoff"
	"github.com/c360studio/semlink/internal/mesh"
	"github.com/c360studio/semlink/internal/rules"
)

const (
	EvidenceContractName    = "c360.semlink.companion.evidence"
	EvidenceContractVersion = "v1"
	DefaultNodeID           = "semlink-local"
)

type EvidenceBundle struct {
	Contract    EvidenceContract  `json:"contract"`
	GeneratedAt time.Time         `json:"generated_at"`
	Node        NodeEvidence      `json:"node"`
	Profile     ProfileEvidence   `json:"profile"`
	Downstream  []DownstreamView  `json:"downstream"`
	Vehicles    []VehicleEvidence `json:"vehicles"`
	Mesh        MeshEvidence      `json:"mesh"`
	RuleTraces  []RuleTraceView   `json:"rule_traces"`
	Commands    []CommandEvidence `json:"commands"`
}

type EvidenceContract struct {
	Name      string            `json:"name"`
	Version   string            `json:"version"`
	Consumers []string          `json:"consumers"`
	APIPaths  map[string]string `json:"api_paths"`
}

type NodeEvidence struct {
	NodeID             string    `json:"node_id"`
	Status             string    `json:"status"`
	Runtime            string    `json:"runtime"`
	NATSURL            string    `json:"nats_url"`
	SemStreamsEmbedded bool      `json:"semstreams_embedded"`
	StartedAt          time.Time `json:"started_at"`
	UptimeSeconds      float64   `json:"uptime_seconds"`
	RawFrames          int64     `json:"raw_frames"`
	DecodedFrames      int64     `json:"decoded_frames"`
	GraphWrites        int64     `json:"graph_writes"`
	GraphErrors        int64     `json:"graph_errors"`
	BufferDrops        int64     `json:"buffer_drops"`
}

type ProfileEvidence struct {
	Status     string                    `json:"status"`
	NodeID     string                    `json:"node_id,omitempty"`
	VehicleID  string                    `json:"vehicle_id,omitempty"`
	Callsign   string                    `json:"callsign,omitempty"`
	HTTPListen string                    `json:"http_listen,omitempty"`
	BlueOS     BlueOSProfileEvidence     `json:"blueos"`
	SemStreams SemStreamsProfileEvidence `json:"semstreams"`
	MAVLink    MAVLinkProfileEvidence    `json:"mavlink"`
	Simulator  SimulatorProfileEvidence  `json:"simulator"`
	Mesh       MeshProfileEvidence       `json:"mesh"`
	Downstream DownstreamProfileEvidence `json:"downstream"`
	Command    CommandProfileEvidence    `json:"command"`
	TAK        TAKProfileEvidence        `json:"tak"`
}

type BlueOSProfileEvidence struct {
	HostPort int `json:"host_port,omitempty"`
}

type SemStreamsProfileEvidence struct {
	Embedded bool   `json:"embedded"`
	NATSURL  string `json:"nats_url,omitempty"`
}

type MAVLinkProfileEvidence struct {
	UDPListen               string `json:"udp_listen,omitempty"`
	UDPHost                 string `json:"udp_host,omitempty"`
	UDPPort                 int    `json:"udp_port,omitempty"`
	ExternalInputConfigured bool   `json:"external_input_configured"`
}

type SimulatorProfileEvidence struct {
	Vehicles int    `json:"vehicles"`
	Hz       int    `json:"hz"`
	Buffer   int    `json:"buffer"`
	Enabled  bool   `json:"enabled"`
	Source   string `json:"source"`
}

type MeshProfileEvidence struct {
	Mode      string   `json:"mode"`
	PeerCount int      `json:"peer_count"`
	Peers     []string `json:"peers,omitempty"`
}

type DownstreamProfileEvidence struct {
	CSAPIConfigured bool   `json:"csapi_configured"`
	CSAPIURL        string `json:"csapi_url,omitempty"`
}

type CommandProfileEvidence struct {
	RuntimeMode             string `json:"runtime_mode"`
	HardwareTransmitEnabled bool   `json:"hardware_transmit_enabled"`
	HardwareTransmitStatus  string `json:"hardware_transmit_status"`
}

type TAKProfileEvidence struct {
	Enabled        bool   `json:"enabled"`
	MulticastAddr  string `json:"multicast_addr,omitempty"`
	TCPListen      string `json:"tcp_listen,omitempty"`
	InboundUDP     string `json:"inbound_udp,omitempty"`
	InboundTCP     string `json:"inbound_tcp,omitempty"`
	PublishSeconds int64  `json:"publish_seconds,omitempty"`
}

type DownstreamView struct {
	Name       string   `json:"name"`
	Owner      string   `json:"owner"`
	Role       string   `json:"role"`
	Optional   bool     `json:"optional"`
	Direction  string   `json:"direction"`
	Enabled    bool     `json:"enabled"`
	Status     string   `json:"status"`
	APIPaths   []string `json:"api_paths,omitempty"`
	TargetURL  string   `json:"target_url,omitempty"`
	Boundary   string   `json:"boundary"`
	NoRawMesh  bool     `json:"no_raw_mesh,omitempty"`
	NoGCSGlass bool     `json:"no_gcs_glass,omitempty"`
}

type VehicleEvidence struct {
	EntityID         string    `json:"entity_id"`
	Callsign         string    `json:"callsign"`
	SystemID         uint8     `json:"system_id"`
	VehicleType      string    `json:"vehicle_type,omitempty"`
	Mode             string    `json:"mode"`
	FlightStatus     string    `json:"flight_status"`
	LinkStatus       string    `json:"link_status"`
	BatteryRemaining int       `json:"battery_remaining"`
	LatitudeDeg      float64   `json:"latitude_deg"`
	LongitudeDeg     float64   `json:"longitude_deg"`
	LastSeen         time.Time `json:"last_seen"`
	GraphRevision    uint64    `json:"graph_revision"`
	IndexingProfile  string    `json:"indexing_profile"`
	EvidenceClass    string    `json:"evidence_class"`
}

type MeshEvidence struct {
	Status                        string               `json:"status"`
	SummaryCount                  int                  `json:"summary_count"`
	WatermarkCount                int                  `json:"watermark_count"`
	Watermarks                    []mesh.CellWatermark `json:"watermarks"`
	RawMAVLinkReplicatesByDefault bool                 `json:"raw_mavlink_replicates_by_default"`
}

type RuleTraceView struct {
	EntityID           string    `json:"entity_id"`
	RuleID             string    `json:"rule_id"`
	RuleVersion        string    `json:"rule_version"`
	NodeID             string    `json:"node_id"`
	VehicleID          string    `json:"vehicle_id,omitempty"`
	TargetEntity       string    `json:"target_entity,omitempty"`
	Decision           string    `json:"decision"`
	SuggestedAction    string    `json:"suggested_action"`
	ExecutionPosture   string    `json:"execution_posture"`
	FiredAt            time.Time `json:"fired_at"`
	InputHash          string    `json:"input_hash"`
	InputCount         int       `json:"input_count"`
	IndexingProfile    string    `json:"indexing_profile"`
	SemanticEntityType string    `json:"semantic_entity_type"`
}

type CommandGateView struct {
	Accepted          bool                                       `json:"accepted"`
	Status            string                                     `json:"status"`
	StartedAt         time.Time                                  `json:"started_at"`
	RuntimeMode       commandgate.RuntimeMode                    `json:"runtime_mode,omitempty"`
	SafetyProfile     string                                     `json:"safety_profile,omitempty"`
	TargetEntity      string                                     `json:"target_entity,omitempty"`
	Verb              string                                     `json:"verb,omitempty"`
	RequestedBy       string                                     `json:"requested_by,omitempty"`
	FrameCount        int                                        `json:"frame_count"`
	ACKCount          int                                        `json:"ack_count"`
	PostStateObserved bool                                       `json:"post_state_observed"`
	HardwareBlock     *commandgate.HardwareTransmitBlockEvidence `json:"hardware_block,omitempty"`
}

type CommandEvidence struct {
	Kind          string           `json:"kind"`
	EntityID      string           `json:"entity_id,omitempty"`
	TargetEntity  string           `json:"target_entity,omitempty"`
	Verb          string           `json:"verb,omitempty"`
	Status        string           `json:"status"`
	RequestedAt   time.Time        `json:"requested_at,omitempty"`
	GraphRevision uint64           `json:"graph_revision,omitempty"`
	Gate          *CommandGateView `json:"gate,omitempty"`
}

func (s *Server) EvidenceBundle(now time.Time) EvidenceBundle {
	if now.IsZero() {
		now = time.Now()
	}
	snapshot := s.store.Snapshot()
	return EvidenceBundle{
		Contract:    defaultEvidenceContract(),
		GeneratedAt: now,
		Node:        nodeEvidence(s.nodeID, snapshot.Metrics, now),
		Profile:     profileEvidence(s.handoffProfile),
		Downstream:  downstreamViews(s.csapiURL),
		Vehicles:    vehicleEvidence(snapshot.Vehicles),
		Mesh:        s.meshEvidence(now),
		RuleTraces:  snapshot.RuleTraces,
		Commands:    commandEvidence(snapshot.Commands, snapshot.CommandGates),
	}
}

func defaultEvidenceContract() EvidenceContract {
	return EvidenceContract{
		Name:    EvidenceContractName,
		Version: EvidenceContractVersion,
		Consumers: []string{
			"SemOps GCS/COP",
			"semstreams-ui ops/debug",
			"CLI/config automation",
		},
		APIPaths: map[string]string{
			"bundle":     "/api/evidence",
			"snapshot":   "/api/snapshot",
			"graph_lens": "/api/graph?entity_id={entity_id}",
			"events":     "/api/events",
			"commands":   "/api/commands",
		},
	}
}

func profileEvidence(profile *handoff.Profile) ProfileEvidence {
	if profile == nil {
		return ProfileEvidence{Status: "not-configured"}
	}
	simulatorEnabled := profile.MAVLinkUDPListen == ""
	simulatorSource := string(TelemetrySourceInternalSimulator)
	if !simulatorEnabled {
		simulatorSource = string(TelemetrySourceExternalMAVLinkUDP)
	}
	meshMode := "off"
	if len(profile.MeshPeers) > 0 {
		meshMode = "static-peers"
	}
	hardwareStatus := "blocked"
	if profile.HardwareTransmitEnabled {
		hardwareStatus = "enabled"
	}
	return ProfileEvidence{
		Status:     "configured",
		NodeID:     profile.NodeID,
		VehicleID:  profile.VehicleID,
		Callsign:   profile.Callsign,
		HTTPListen: profile.HTTPListen,
		BlueOS: BlueOSProfileEvidence{
			HostPort: profile.BlueOSHostPort,
		},
		SemStreams: SemStreamsProfileEvidence{
			Embedded: profile.EmbeddedNATS,
			NATSURL:  profile.NATSURL,
		},
		MAVLink: MAVLinkProfileEvidence{
			UDPListen:               profile.MAVLinkUDPListen,
			UDPHost:                 profile.MAVLinkUDPHost,
			UDPPort:                 profile.MAVLinkUDPPort,
			ExternalInputConfigured: profile.MAVLinkUDPListen != "",
		},
		Simulator: SimulatorProfileEvidence{
			Vehicles: profile.Vehicles,
			Hz:       profile.Hz,
			Buffer:   profile.Buffer,
			Enabled:  simulatorEnabled,
			Source:   simulatorSource,
		},
		Mesh: MeshProfileEvidence{
			Mode:      meshMode,
			PeerCount: len(profile.MeshPeers),
			Peers:     append([]string(nil), profile.MeshPeers...),
		},
		Downstream: DownstreamProfileEvidence{
			CSAPIConfigured: profile.CSAPIURL != "",
			CSAPIURL:        profile.CSAPIURL,
		},
		Command: CommandProfileEvidence{
			RuntimeMode:             string(profile.CommandRuntimeMode),
			HardwareTransmitEnabled: profile.HardwareTransmitEnabled,
			HardwareTransmitStatus:  hardwareStatus,
		},
		TAK: TAKProfileEvidence{
			Enabled:        profile.TAKEnabled,
			MulticastAddr:  profile.TAKMulticastAddr,
			TCPListen:      profile.TAKTCPListen,
			InboundUDP:     profile.TAKInboundUDPListen,
			InboundTCP:     profile.TAKInboundTCPListen,
			PublishSeconds: int64(profile.TAKInterval.Seconds()),
		},
	}
}

func downstreamViews(csapiURL string) []DownstreamView {
	csapiURL = strings.TrimSpace(csapiURL)
	semconnectStatus := "disabled"
	semconnectEnabled := false
	if csapiURL != "" {
		semconnectStatus = "configured"
		semconnectEnabled = true
	}
	return []DownstreamView{
		{
			Name:       "semops",
			Owner:      "SemOps",
			Role:       "GCS/COP glass and fusion",
			Optional:   true,
			Direction:  "pull-local-api",
			Enabled:    true,
			Status:     "available",
			APIPaths:   []string{"/api/evidence", "/api/events", "/api/graph?entity_id={entity_id}"},
			Boundary:   "SemOps consumes companion evidence; SemLink does not own COP/GCS glass.",
			NoGCSGlass: true,
		},
		{
			Name:       "semstreams-ui",
			Owner:      "semstreams-ui",
			Role:       "generic ops/debug view",
			Optional:   true,
			Direction:  "pull-local-api",
			Enabled:    true,
			Status:     "available",
			APIPaths:   []string{"/api/evidence", "/api/snapshot", "/api/graph?entity_id={entity_id}"},
			Boundary:   "semstreams-ui can inspect evidence without becoming a SemLink dependency.",
			NoGCSGlass: true,
		},
		{
			Name:      "semconnect-csapi",
			Owner:     "SemConnect",
			Role:      "OGC API - Connected Systems standards egress",
			Optional:  true,
			Direction: "egress-http",
			Enabled:   semconnectEnabled,
			Status:    semconnectStatus,
			TargetURL: csapiURL,
			Boundary:  "Curated low-rate standards projection only; not mesh sync and not raw MAVLink.",
			NoRawMesh: true,
		},
	}
}

func nodeEvidence(nodeID string, metrics MetricsView, now time.Time) NodeEvidence {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		nodeID = DefaultNodeID
	}
	uptime := 0.0
	if !metrics.StartedAt.IsZero() && now.After(metrics.StartedAt) {
		uptime = now.Sub(metrics.StartedAt).Seconds()
	}
	status := "ok"
	if metrics.GraphErrors > 0 || metrics.DecodeErrors > 0 || metrics.RawPublishErrors > 0 {
		status = "degraded"
	}
	runtime := "external-semstreams"
	if metrics.SemStreamsEmbedded {
		runtime = "embedded-semstreams"
	}
	return NodeEvidence{
		NodeID:             nodeID,
		Status:             status,
		Runtime:            runtime,
		NATSURL:            metrics.NATSURL,
		SemStreamsEmbedded: metrics.SemStreamsEmbedded,
		StartedAt:          metrics.StartedAt,
		UptimeSeconds:      uptime,
		RawFrames:          metrics.RawFrames,
		DecodedFrames:      metrics.DecodedFrames,
		GraphWrites:        metrics.GraphWrites,
		GraphErrors:        metrics.GraphErrors,
		BufferDrops:        metrics.BufferDrops,
	}
}

func vehicleEvidence(vehicles []VehicleView) []VehicleEvidence {
	out := make([]VehicleEvidence, 0, len(vehicles))
	for _, vehicle := range vehicles {
		out = append(out, VehicleEvidence{
			EntityID:         vehicle.EntityID,
			Callsign:         vehicle.Callsign,
			SystemID:         vehicle.SystemID,
			VehicleType:      vehicle.VehicleType,
			Mode:             vehicle.Mode,
			FlightStatus:     vehicle.FlightStatus,
			LinkStatus:       vehicle.LinkStatus,
			BatteryRemaining: vehicle.BatteryRemaining,
			LatitudeDeg:      vehicle.LatitudeDeg,
			LongitudeDeg:     vehicle.LongitudeDeg,
			LastSeen:         vehicle.LastSeen,
			GraphRevision:    vehicle.GraphRevision,
			IndexingProfile:  vehicle.IndexingProfile,
			EvidenceClass:    "mavlink-current-state",
		})
	}
	return out
}

func (s *Server) meshEvidence(now time.Time) MeshEvidence {
	evidence := MeshEvidence{
		Status:                        "not-configured",
		RawMAVLinkReplicatesByDefault: mesh.SourceKindRawMAVLink.ReplicatesOverMeshByDefault(),
	}
	if s.mesh == nil {
		return evidence
	}
	watermarks := s.mesh.Watermarks(now)
	evidence.Status = "configured"
	evidence.SummaryCount = s.mesh.Len()
	evidence.WatermarkCount = len(watermarks.Entries)
	evidence.Watermarks = watermarks.Entries
	return evidence
}

func commandEvidence(commands []CommandView, gates []CommandGateView) []CommandEvidence {
	out := make([]CommandEvidence, 0, len(commands)+len(gates))
	for _, command := range commands {
		out = append(out, CommandEvidence{
			Kind:          "command-intent",
			EntityID:      command.EntityID,
			TargetEntity:  command.TargetEntity,
			Verb:          command.Verb,
			Status:        command.Status,
			RequestedAt:   command.RequestedAt,
			GraphRevision: command.GraphRevision,
		})
	}
	for _, gate := range gates {
		gateCopy := gate
		out = append(out, CommandEvidence{
			Kind:         "command-gate",
			TargetEntity: gate.TargetEntity,
			Verb:         gate.Verb,
			Status:       gate.Status,
			Gate:         &gateCopy,
		})
	}
	return out
}

func ruleTraceView(trace *rules.TracePayload) RuleTraceView {
	if trace == nil {
		return RuleTraceView{}
	}
	return RuleTraceView{
		EntityID:           trace.ID,
		RuleID:             trace.RuleID,
		RuleVersion:        trace.RuleVersion,
		NodeID:             trace.NodeID,
		VehicleID:          trace.VehicleID,
		TargetEntity:       trace.TargetEntity,
		Decision:           trace.Decision,
		SuggestedAction:    trace.SuggestedAction,
		ExecutionPosture:   string(trace.ExecutionPosture),
		FiredAt:            trace.FiredAt,
		InputHash:          trace.InputHash,
		InputCount:         len(trace.InputFacts),
		IndexingProfile:    trace.IndexingProfile(),
		SemanticEntityType: rules.TraceType.String(),
	}
}
