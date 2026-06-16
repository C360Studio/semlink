package gcs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/c360studio/semlink/internal/cop"
	"github.com/c360studio/semlink/internal/projector"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
)

type EntityQuerier interface {
	QueryEntity(ctx context.Context, id string) (*graph.EntityState, error)
}

type GraphView struct {
	GeneratedAt time.Time   `json:"generated_at"`
	EntityID    string      `json:"entity_id"`
	EntityKind  string      `json:"entity_kind"`
	EntityLabel string      `json:"entity_label"`
	VehicleID   string      `json:"vehicle_id,omitempty"`
	Lenses      []GraphLens `json:"lenses"`
}

type GraphLens struct {
	Source  string      `json:"source"`
	Label   string      `json:"label"`
	Status  string      `json:"status"`
	Summary string      `json:"summary"`
	Nodes   []GraphNode `json:"nodes"`
	Edges   []GraphEdge `json:"edges"`
	Facts   []GraphFact `json:"facts"`
	Stats   []GraphStat `json:"stats"`
}

type GraphNode struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	Profile string `json:"profile,omitempty"`
	Detail  string `json:"detail,omitempty"`
	Status  string `json:"status,omitempty"`
}

type GraphEdge struct {
	From  string `json:"from"`
	To    string `json:"to"`
	Label string `json:"label"`
}

type GraphFact struct {
	Subject   string `json:"subject"`
	Predicate string `json:"predicate"`
	Object    string `json:"object"`
	Source    string `json:"source,omitempty"`
}

type GraphStat struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func (s *Server) BuildGraphView(ctx context.Context, entityID string) (GraphView, error) {
	snapshot := s.store.Snapshot()
	if vehicle, ok := findVehicle(snapshot.Vehicles, entityID); ok {
		return s.buildVehicleGraphView(ctx, snapshot, vehicle), nil
	}

	if item, ok := findCOPView(snapshot, entityID); ok {
		return GraphView{
			GeneratedAt: time.Now(),
			EntityID:    item.EntityID,
			EntityKind:  string(item.Kind),
			EntityLabel: copDisplayName(item),
			Lenses: []GraphLens{
				s.buildCOPLens(ctx, item),
				s.buildCOPCSAPILens(ctx, snapshot, item),
			},
		}, nil
	}

	return GraphView{}, errors.New("entity not found")
}

func (s *Server) buildVehicleGraphView(ctx context.Context, snapshot Snapshot, vehicle VehicleView) GraphView {
	view := GraphView{
		GeneratedAt: time.Now(),
		EntityID:    vehicle.EntityID,
		EntityKind:  "vehicle",
		EntityLabel: vehicle.Callsign,
		VehicleID:   vehicle.EntityID,
		Lenses: []GraphLens{
			s.buildSemLinkLens(ctx, snapshot, vehicle),
			s.buildCSAPILens(ctx, snapshot, vehicle),
		},
	}
	return view
}

func findVehicle(vehicles []VehicleView, id string) (VehicleView, bool) {
	if id != "" {
		for _, vehicle := range vehicles {
			if vehicle.EntityID == id {
				return vehicle, true
			}
		}
		return VehicleView{}, false
	}
	if len(vehicles) == 0 {
		return VehicleView{}, false
	}
	return vehicles[0], true
}

func findCOPView(snapshot Snapshot, id string) (cop.View, bool) {
	if id == "" {
		return cop.View{}, false
	}
	for _, item := range snapshot.Operators {
		if item.EntityID == id {
			return item, true
		}
	}
	for _, item := range snapshot.Markers {
		if item.EntityID == id {
			return item, true
		}
	}
	for _, item := range snapshot.Messages {
		if item.EntityID == id {
			return item, true
		}
	}
	return cop.View{}, false
}

func (s *Server) buildSemLinkLens(ctx context.Context, snapshot Snapshot, vehicle VehicleView) GraphLens {
	nodes := []GraphNode{
		{
			ID:      vehicle.EntityID,
			Label:   vehicle.Callsign,
			Kind:    "Vehicle",
			Profile: vehicle.IndexingProfile,
			Detail:  fmt.Sprintf("rev %d", vehicle.GraphRevision),
			Status:  vehicle.LinkStatus,
		},
		{
			ID:      vehicle.EntityID + "#battery",
			Label:   "Battery",
			Kind:    "Telemetry",
			Profile: "signal",
			Detail:  fmt.Sprintf("%d%%", vehicle.BatteryRemaining),
			Status:  batteryStatus(vehicle.BatteryRemaining),
		},
		{
			ID:      vehicle.EntityID + "#link",
			Label:   "Link",
			Kind:    "Derived",
			Profile: "signal",
			Detail:  vehicle.LinkStatus,
			Status:  vehicle.LinkStatus,
		},
		{
			ID:      vehicle.EntityID + "#position",
			Label:   "Position",
			Kind:    "Telemetry",
			Profile: "signal",
			Detail:  fmt.Sprintf("%.5f, %.5f", vehicle.LatitudeDeg, vehicle.LongitudeDeg),
			Status:  "active",
		},
	}
	edges := []GraphEdge{
		{From: vehicle.EntityID, To: vehicle.EntityID + "#battery", Label: "summarizes"},
		{From: vehicle.EntityID, To: vehicle.EntityID + "#link", Label: "governs"},
		{From: vehicle.EntityID, To: vehicle.EntityID + "#position", Label: "locates"},
	}

	activeAlerts := 0
	for _, alert := range snapshot.Alerts {
		if alert.SubjectEntity != vehicle.EntityID {
			continue
		}
		if alert.Active {
			activeAlerts++
		}
		nodes = append(nodes, GraphNode{
			ID:      alert.EntityID,
			Label:   titleToken(alert.Kind),
			Kind:    "Alert",
			Profile: "control",
			Detail:  alert.Message,
			Status:  alert.Severity,
		})
		edges = append(edges, GraphEdge{From: vehicle.EntityID, To: alert.EntityID, Label: "raises"})
	}

	for _, command := range snapshot.Commands {
		if command.TargetEntity != vehicle.EntityID {
			continue
		}
		nodes = append(nodes, GraphNode{
			ID:      command.EntityID,
			Label:   titleToken(command.Verb),
			Kind:    "Command",
			Profile: "control",
			Detail:  command.Status,
			Status:  command.Status,
		})
		edges = append(edges, GraphEdge{From: command.EntityID, To: vehicle.EntityID, Label: "targets"})
	}

	facts, partial := s.semLinkFacts(ctx, snapshot, vehicle)
	status := "live"
	summary := "SemStreams operational graph"
	if partial {
		status = "partial"
		summary = "SemStreams operational graph, with snapshot fallback"
	}
	return GraphLens{
		Source:  "semlink",
		Label:   "SemLink Graph",
		Status:  status,
		Summary: summary,
		Nodes:   nodes,
		Edges:   edges,
		Facts:   facts,
		Stats: []GraphStat{
			{Label: "profile", Value: vehicle.IndexingProfile},
			{Label: "alerts", Value: strconv.Itoa(activeAlerts)},
			{Label: "commands", Value: strconv.Itoa(countCommandsForVehicle(snapshot.Commands, vehicle.EntityID))},
			{Label: "revision", Value: strconv.FormatUint(vehicle.GraphRevision, 10)},
		},
	}
}

func (s *Server) semLinkFacts(ctx context.Context, snapshot Snapshot, vehicle VehicleView) ([]GraphFact, bool) {
	facts := make([]GraphFact, 0, 12)
	partial := false

	if s.graph != nil {
		entity, err := s.graph.QueryEntity(ctx, vehicle.EntityID)
		if err != nil {
			partial = true
		} else {
			facts = appendFacts(facts, entity, 8)
		}
	} else {
		partial = true
	}
	if len(facts) == 0 {
		facts = append(facts,
			GraphFact{Subject: shortID(vehicle.EntityID), Predicate: projector.PredicateBatteryRemainingPct, Object: fmt.Sprintf("%d", vehicle.BatteryRemaining), Source: projector.SourceMAVLink},
			GraphFact{Subject: shortID(vehicle.EntityID), Predicate: projector.PredicateLinkStatus, Object: vehicle.LinkStatus, Source: projector.SourceProjector},
			GraphFact{Subject: shortID(vehicle.EntityID), Predicate: projector.PredicateFlightMode, Object: vehicle.Mode, Source: projector.SourceMAVLink},
		)
	}

	for _, alert := range snapshot.Alerts {
		if alert.SubjectEntity != vehicle.EntityID || len(facts) >= 12 {
			continue
		}
		facts = append(facts, GraphFact{
			Subject:   shortID(alert.EntityID),
			Predicate: projector.PredicateAlertSeverity,
			Object:    alert.Severity,
			Source:    projector.SourceProjector,
		})
	}
	for _, command := range snapshot.Commands {
		if command.TargetEntity != vehicle.EntityID || len(facts) >= 12 {
			continue
		}
		facts = append(facts, GraphFact{
			Subject:   shortID(command.EntityID),
			Predicate: projector.PredicateCommandVerb,
			Object:    command.Verb,
			Source:    projector.SourceOperator,
		})
	}
	return facts, partial
}

func (s *Server) buildCOPLens(ctx context.Context, item cop.View) GraphLens {
	nodes := []GraphNode{{
		ID:      item.EntityID,
		Label:   copDisplayName(item),
		Kind:    copKindLabel(item.Kind),
		Profile: item.IndexingProfile,
		Detail:  fmt.Sprintf("rev %d", item.GraphRevision),
		Status:  "active",
	}}
	edges := make([]GraphEdge, 0, 3)
	if item.HasPosition {
		positionID := item.EntityID + "#position"
		nodes = append(nodes, GraphNode{
			ID:      positionID,
			Label:   "Position",
			Kind:    "CoT Point",
			Profile: item.IndexingProfile,
			Detail:  fmt.Sprintf("%.5f, %.5f", item.LatitudeDeg, item.LongitudeDeg),
			Status:  "located",
		})
		edges = append(edges, GraphEdge{From: item.EntityID, To: positionID, Label: "locates"})
	}
	if item.Kind == cop.KindMessage && item.SenderUID != "" {
		senderID := firstNonEmpty(item.SenderEntity, item.EntityID+"#sender")
		nodes = append(nodes, GraphNode{
			ID:     senderID,
			Label:  firstNonEmpty(item.Callsign, item.SenderUID),
			Kind:   "Sender",
			Detail: item.SenderUID,
			Status: "source",
		})
		edges = append(edges, GraphEdge{From: senderID, To: item.EntityID, Label: "sent"})
	}

	facts, partial := s.copFacts(ctx, item)
	status := "live"
	summary := "SemStreams COP graph"
	if partial {
		status = "partial"
		summary = "SemStreams COP graph, with snapshot fallback"
	}
	return GraphLens{
		Source:  "semlink",
		Label:   "SemLink Graph",
		Status:  status,
		Summary: summary,
		Nodes:   nodes,
		Edges:   edges,
		Facts:   facts,
		Stats: []GraphStat{
			{Label: "kind", Value: string(item.Kind)},
			{Label: "profile", Value: item.IndexingProfile},
			{Label: "revision", Value: strconv.FormatUint(item.GraphRevision, 10)},
			{Label: "uid", Value: item.UID},
		},
	}
}

func (s *Server) copFacts(ctx context.Context, item cop.View) ([]GraphFact, bool) {
	facts := make([]GraphFact, 0, 12)
	partial := false
	if s.graph != nil {
		entity, err := s.graph.QueryEntity(ctx, item.EntityID)
		if err != nil {
			partial = true
		} else {
			facts = appendFacts(facts, entity, 12)
		}
	} else {
		partial = true
	}
	if len(facts) > 0 {
		return facts, partial
	}
	facts = append(facts,
		GraphFact{Subject: shortID(item.EntityID), Predicate: cop.PredicateCOTUID, Object: item.UID, Source: cop.SourceCoT},
		GraphFact{Subject: shortID(item.EntityID), Predicate: cop.PredicateKind, Object: string(item.Kind), Source: cop.SourceCOP},
	)
	switch item.Kind {
	case cop.KindOperator:
		facts = append(facts, GraphFact{Subject: shortID(item.EntityID), Predicate: cop.PredicateCallsign, Object: item.Callsign, Source: cop.SourceCoT})
	case cop.KindMarker:
		facts = append(facts,
			GraphFact{Subject: shortID(item.EntityID), Predicate: cop.PredicateLabel, Object: item.Label, Source: cop.SourceCoT},
			GraphFact{Subject: shortID(item.EntityID), Predicate: cop.PredicateDescription, Object: item.Description, Source: cop.SourceCoT},
		)
	case cop.KindMessage:
		facts = append(facts,
			GraphFact{Subject: shortID(item.EntityID), Predicate: cop.PredicateMessageText, Object: item.Text, Source: cop.SourceCoT},
			GraphFact{Subject: shortID(item.EntityID), Predicate: cop.PredicateMessageSenderUID, Object: item.SenderUID, Source: cop.SourceCoT},
		)
	}
	if item.HasPosition {
		facts = append(facts,
			GraphFact{Subject: shortID(item.EntityID), Predicate: cop.PredicatePositionLatitudeDeg, Object: objectString(item.LatitudeDeg), Source: cop.SourceCoT},
			GraphFact{Subject: shortID(item.EntityID), Predicate: cop.PredicatePositionLongitudeDeg, Object: objectString(item.LongitudeDeg), Source: cop.SourceCoT},
		)
	}
	return facts, true
}

func appendFacts(facts []GraphFact, entity *graph.EntityState, limit int) []GraphFact {
	if entity == nil {
		return facts
	}
	triples := append([]message.Triple(nil), entity.Triples...)
	sort.SliceStable(triples, func(i, j int) bool {
		return predicateRank(triples[i].Predicate) < predicateRank(triples[j].Predicate)
	})
	for _, triple := range triples {
		if len(facts) >= limit {
			return facts
		}
		facts = append(facts, GraphFact{
			Subject:   shortID(triple.Subject),
			Predicate: triple.Predicate,
			Object:    objectString(triple.Object),
			Source:    triple.Source,
		})
	}
	return facts
}

func predicateRank(predicate string) int {
	switch predicate {
	case cop.PredicateKind:
		return 0
	case cop.PredicateCOTUID:
		return 1
	case cop.PredicateCallsign, cop.PredicateLabel:
		return 2
	case cop.PredicateMessageText, cop.PredicateDescription:
		return 3
	case cop.PredicateMessageSenderUID, cop.PredicateMessageSenderEntity:
		return 4
	case projector.PredicateVehicleCallsign:
		return 0
	case projector.PredicateLinkStatus:
		return 1
	case projector.PredicateBatteryRemainingPct:
		return 2
	case projector.PredicatePositionAltitudeM:
		return 3
	case projector.PredicatePositionGroundSpeedMS:
		return 4
	case projector.PredicateFlightMode:
		return 5
	default:
		return 100
	}
}

func (s *Server) buildCSAPILens(ctx context.Context, snapshot Snapshot, vehicle VehicleView) GraphLens {
	if strings.TrimSpace(s.csapiURL) == "" {
		return csapiDisabledLens()
	}

	base, err := url.Parse(strings.TrimRight(s.csapiURL, "/"))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return csapiErrorLens("invalid endpoint")
	}

	var system map[string]any
	if id := csapiVehicleSystemID(vehicle); id != "" {
		system, _ = s.fetchCSAPIItem(ctx, base, "/systems/"+url.PathEscape(id))
	}
	if system == nil {
		systems, err := s.fetchCSAPIItems(ctx, base, "/systems")
		if err != nil {
			return csapiErrorLens(err.Error())
		}
		system = findCSAPISystem(systems, vehicle)
	}
	if system == nil {
		return GraphLens{
			Source:  "csapi",
			Label:   "SemConnect Projection",
			Status:  "pending",
			Summary: "CS API system not materialized yet",
			Stats:   []GraphStat{{Label: "endpoint", Value: base.String()}},
		}
	}
	systemID := stringField(system, "id")
	nodes := []GraphNode{{
		ID:     systemID,
		Label:  firstNonEmpty(stringField(system, "name"), vehicle.Callsign),
		Kind:   "System",
		Detail: "CS API",
		Status: "materialized",
	}}
	edges := make([]GraphEdge, 0, 12)
	facts := []GraphFact{
		{Subject: shortID(systemID), Predicate: "cs-api.resource", Object: "System", Source: "semconnect"},
	}

	selectedStreams := s.fetchVehicleCSAPIDatastreams(ctx, base, systemID, vehicle)
	for _, stream := range selectedStreams {
		id := stringField(stream, "id")
		nodes = append(nodes, GraphNode{
			ID:     id,
			Label:  firstNonEmpty(stringField(stream, "name"), "Datastream"),
			Kind:   "Datastream",
			Detail: shortIRI(stringField(stream, "observedProperty")),
			Status: "decimated",
		})
		edges = append(edges, GraphEdge{From: systemID, To: id, Label: "exposes"})
		if len(facts) < 10 {
			facts = append(facts, GraphFact{Subject: shortID(id), Predicate: "observedProperty", Object: shortIRI(stringField(stream, "observedProperty")), Source: "cs-api"})
		}
	}

	for _, stream := range selectedStreams {
		streamID := stringField(stream, "id")
		if count, ok := s.fetchCSAPIObservationCount(ctx, base, streamID); ok {
			obsID := streamID + "#observations"
			nodes = append(nodes, GraphNode{
				ID:     obsID,
				Label:  csapiObservationHistoryLabel(stream),
				Kind:   "History",
				Detail: fmt.Sprintf("%d records", count),
				Status: "historical",
			})
			edges = append(edges, GraphEdge{From: streamID, To: obsID, Label: "records"})
			if len(facts) < 12 {
				facts = append(facts, GraphFact{Subject: shortID(obsID), Predicate: "numberMatched", Object: strconv.Itoa(count), Source: "cs-api"})
			}
		}
	}

	events, _ := s.fetchCSAPIItems(ctx, base, "/systemEvents")
	selectedEvents := filterCSAPIByField(events, "system@id", systemID)
	for _, event := range selectedEvents {
		id := stringField(event, "id")
		nodes = append(nodes, GraphNode{
			ID:     id,
			Label:  firstNonEmpty(stringField(event, "eventType"), "SystemEvent"),
			Kind:   "SystemEvent",
			Detail: stringField(event, "message"),
			Status: "event",
		})
		edges = append(edges, GraphEdge{From: systemID, To: id, Label: "emits"})
		if len(facts) < 10 {
			facts = append(facts, GraphFact{Subject: shortID(id), Predicate: "eventType", Object: stringField(event, "eventType"), Source: "cs-api"})
		}
	}

	controlStreams, _ := s.fetchCSAPIItems(ctx, base, "/controlstreams")
	selectedControls := filterCSAPIByField(controlStreams, "system@id", systemID)
	for _, control := range selectedControls {
		id := stringField(control, "id")
		nodes = append(nodes, GraphNode{
			ID:     id,
			Label:  firstNonEmpty(stringField(control, "name"), "ControlStream"),
			Kind:   "ControlStream",
			Detail: firstNonEmpty(stringField(control, "inputName"), "gcs-command"),
			Status: "control",
		})
		edges = append(edges, GraphEdge{From: systemID, To: id, Label: "accepts"})
	}

	commands, _ := s.fetchCSAPIItems(ctx, base, "/commands")
	selectedCommands := filterCSAPICommandsForControls(commands, selectedControls)
	for _, command := range snapshot.Commands {
		if command.TargetEntity != vehicle.EntityID {
			continue
		}
		if item, ok := s.fetchCSAPIItem(ctx, base, "/commands/"+url.PathEscape(csapiCommandID(command))); ok {
			selectedCommands = appendUniqueCSAPIItem(selectedCommands, item)
		}
	}
	for _, command := range selectedCommands {
		controlID := stringField(command, "controlstream@id")
		if !containsCSAPIID(selectedControls, controlID) {
			continue
		}
		id := stringField(command, "id")
		nodes = append(nodes, GraphNode{
			ID:     id,
			Label:  firstNonEmpty(commandVerb(command), "Command"),
			Kind:   "Command",
			Detail: stringField(command, "status"),
			Status: "requested",
		})
		edges = append(edges, GraphEdge{From: controlID, To: id, Label: "issues"})
		if len(facts) < 10 {
			facts = append(facts, GraphFact{Subject: shortID(id), Predicate: "status", Object: stringField(command, "status"), Source: "cs-api"})
		}
	}

	return GraphLens{
		Source:  "csapi",
		Label:   "SemConnect Projection",
		Status:  "live",
		Summary: "CS API materialized view",
		Nodes:   nodes,
		Edges:   edges,
		Facts:   facts,
		Stats: []GraphStat{
			{Label: "systems", Value: "1"},
			{Label: "datastreams", Value: strconv.Itoa(len(selectedStreams))},
			{Label: "events", Value: strconv.Itoa(len(selectedEvents))},
			{Label: "commands", Value: strconv.Itoa(len(selectedCommands))},
		},
	}
}

func (s *Server) fetchVehicleCSAPIDatastreams(ctx context.Context, base *url.URL, systemID string, vehicle VehicleView) []map[string]any {
	selected := make([]map[string]any, 0, len(csapiVehicleTelemetryKeys))
	for _, key := range csapiVehicleTelemetryKeys {
		id := csapiVehicleDatastreamID(vehicle, key)
		if stream, ok := s.fetchCSAPIItem(ctx, base, "/datastreams/"+url.PathEscape(id)); ok {
			selected = appendUniqueCSAPIItem(selected, stream)
		}
	}

	datastreams, err := s.fetchCSAPIItems(ctx, base, "/datastreams")
	if err != nil {
		return selected
	}
	for _, stream := range filterCSAPIByField(datastreams, "system@id", systemID) {
		selected = appendUniqueCSAPIItem(selected, stream)
	}
	if len(selected) == 0 {
		for _, stream := range filterCSAPIByField(datastreams, "system", systemID) {
			selected = appendUniqueCSAPIItem(selected, stream)
		}
	}
	return selected
}

func (s *Server) buildCOPCSAPILens(ctx context.Context, snapshot Snapshot, item cop.View) GraphLens {
	if strings.TrimSpace(s.csapiURL) == "" {
		return csapiDisabledLens()
	}

	base, err := url.Parse(strings.TrimRight(s.csapiURL, "/"))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return csapiErrorLens("invalid endpoint")
	}

	switch item.Kind {
	case cop.KindOperator:
		return s.buildOperatorCOPCSAPILens(ctx, base, item)
	case cop.KindMarker:
		return s.buildMarkerCOPCSAPILens(ctx, base, item)
	case cop.KindMessage:
		return s.buildMessageCOPCSAPILens(ctx, base, snapshot, item)
	default:
		return csapiErrorLens("unsupported COP kind")
	}
}

func (s *Server) buildOperatorCOPCSAPILens(ctx context.Context, base *url.URL, operator cop.View) GraphLens {
	var system map[string]any
	if id := csapiCOPSystemID(operator.UID); id != "" {
		system, _ = s.fetchCSAPIItem(ctx, base, "/systems/"+url.PathEscape(id))
	}
	if system == nil {
		systems, err := s.fetchCSAPIItems(ctx, base, "/systems")
		if err != nil {
			return csapiErrorLens(err.Error())
		}
		system = findCSAPICOPSystem(systems, operator.UID, operator.Callsign)
	}
	if system == nil {
		return csapiPendingLens("COP operator system not materialized yet", base.String())
	}

	systemID := stringField(system, "id")
	nodes := []GraphNode{csapiSystemNode(system, firstNonEmpty(operator.Callsign, operator.UID))}
	edges := make([]GraphEdge, 0, 4)
	facts := []GraphFact{
		{Subject: shortID(systemID), Predicate: "cs-api.resource", Object: "System", Source: "semconnect"},
		{Subject: shortID(systemID), Predicate: "uid", Object: firstNonEmpty(stringField(system, "properties.uid"), operator.UID), Source: "cs-api"},
	}

	selectedStreams := make([]map[string]any, 0, 1)
	if stream, ok := s.fetchCSAPIItem(ctx, base, "/datastreams/"+url.PathEscape(csapiCOPDatastreamID(operator))); ok {
		selectedStreams = append(selectedStreams, stream)
	} else if datastreams, err := s.fetchCSAPIItems(ctx, base, "/datastreams"); err == nil {
		selectedStreams = filterCSAPIByField(datastreams, "system@id", systemID)
		if len(selectedStreams) == 0 {
			selectedStreams = filterCSAPIByField(datastreams, "system", systemID)
		}
		if len(selectedStreams) == 0 {
			selectedStreams = filterCSAPIByIDOrProperty(datastreams, csapiCOPDatastreamID(operator), "observedProperty", "operatorPosition")
		}
	}

	observationCount := 0
	for _, stream := range selectedStreams {
		streamID := stringField(stream, "id")
		nodes = append(nodes, GraphNode{
			ID:     streamID,
			Label:  firstNonEmpty(stringField(stream, "name"), "Operator position"),
			Kind:   "Datastream",
			Detail: shortIRI(stringField(stream, "observedProperty")),
			Status: "decimated",
		})
		edges = append(edges, GraphEdge{From: systemID, To: streamID, Label: "exposes"})
		if len(facts) < 10 {
			facts = append(facts, GraphFact{Subject: shortID(streamID), Predicate: "observedProperty", Object: shortIRI(stringField(stream, "observedProperty")), Source: "cs-api"})
		}
		if count, ok := s.fetchCSAPIObservationCount(ctx, base, streamID); ok {
			observationCount += count
			obsID := streamID + "#observations"
			nodes = append(nodes, GraphNode{
				ID:     obsID,
				Label:  csapiObservationHistoryLabel(stream),
				Kind:   "History",
				Detail: fmt.Sprintf("%d records", count),
				Status: "historical",
			})
			edges = append(edges, GraphEdge{From: streamID, To: obsID, Label: "records"})
		}
	}

	return GraphLens{
		Source:  "csapi",
		Label:   "SemConnect Projection",
		Status:  "live",
		Summary: "CS API materialized TAK operator view",
		Nodes:   nodes,
		Edges:   edges,
		Facts:   facts,
		Stats: []GraphStat{
			{Label: "systems", Value: "1"},
			{Label: "datastreams", Value: strconv.Itoa(len(selectedStreams))},
			{Label: "observations", Value: strconv.Itoa(observationCount)},
		},
	}
}

func (s *Server) buildMarkerCOPCSAPILens(ctx context.Context, base *url.URL, marker cop.View) GraphLens {
	var feature map[string]any
	if id := csapiCOPSamplingFeatureID(marker.UID); id != "" {
		feature, _ = s.fetchCSAPIItem(ctx, base, "/samplingFeatures/"+url.PathEscape(id))
	}
	if feature == nil {
		features, err := s.fetchCSAPIItems(ctx, base, "/samplingFeatures")
		if err != nil {
			return csapiErrorLens(err.Error())
		}
		feature = findCSAPICOPSamplingFeature(features, marker)
	}
	if feature == nil {
		return csapiPendingLens("COP marker SamplingFeature not materialized yet", base.String())
	}

	featureID := stringField(feature, "id")
	nodes := []GraphNode{{
		ID:     featureID,
		Label:  firstNonEmpty(stringField(feature, "name"), stringField(feature, "properties.name"), marker.Label, marker.UID),
		Kind:   "SamplingFeature",
		Detail: "CS API",
		Status: "materialized",
	}}
	facts := []GraphFact{
		{Subject: shortID(featureID), Predicate: "cs-api.resource", Object: "SamplingFeature", Source: "semconnect"},
		{Subject: shortID(featureID), Predicate: "uid", Object: firstNonEmpty(stringField(feature, "properties.uid"), marker.UID), Source: "cs-api"},
		{Subject: shortID(featureID), Predicate: "description", Object: firstNonEmpty(stringField(feature, "description"), stringField(feature, "properties.description"), marker.Description), Source: "cs-api"},
	}

	return GraphLens{
		Source:  "csapi",
		Label:   "SemConnect Projection",
		Status:  "live",
		Summary: "CS API materialized TAK marker view",
		Nodes:   nodes,
		Facts:   facts,
		Stats: []GraphStat{
			{Label: "samplingFeatures", Value: "1"},
			{Label: "geometry", Value: boolText(marker.HasPosition)},
		},
	}
}

func (s *Server) buildMessageCOPCSAPILens(ctx context.Context, base *url.URL, snapshot Snapshot, message cop.View) GraphLens {
	sender := findMessageSender(snapshot, message)
	systemUID := firstNonEmpty(message.SenderUID, sender.UID, message.UID)
	systemName := firstNonEmpty(message.Callsign, sender.Callsign, systemUID)
	var system map[string]any
	if id := csapiCOPSystemID(systemUID); id != "" {
		system, _ = s.fetchCSAPIItem(ctx, base, "/systems/"+url.PathEscape(id))
	}

	selectedEvents := make([]map[string]any, 0, 1)
	if event, ok := s.fetchCSAPIItem(ctx, base, "/systemEvents/"+url.PathEscape(csapiCOPEventID(message))); ok {
		selectedEvents = append(selectedEvents, event)
	} else if events, err := s.fetchCSAPIItems(ctx, base, "/systemEvents"); err == nil {
		selectedEvents = filterCSAPIByIDOrProperty(events, csapiCOPEventID(message), "payload.semlink_cop", message.EntityID)
		if len(selectedEvents) == 0 && system != nil {
			selectedEvents = filterCSAPIByField(events, "system@id", stringField(system, "id"))
			selectedEvents = filterCSAPIByEventKind(selectedEvents, "GeoChat")
		}
	}
	if len(selectedEvents) == 0 {
		return csapiPendingLens("COP GeoChat SystemEvent not materialized yet", base.String())
	}
	if system == nil {
		systemID := stringField(selectedEvents[0], "system@id")
		if systemID != "" {
			system, _ = s.fetchCSAPIItem(ctx, base, "/systems/"+url.PathEscape(systemID))
		}
	}
	if system == nil {
		systems, err := s.fetchCSAPIItems(ctx, base, "/systems")
		if err != nil {
			return csapiErrorLens(err.Error())
		}
		system = findCSAPICOPSystem(systems, systemUID, systemName)
		if system == nil {
			system = findCSAPIByID(systems, stringField(selectedEvents[0], "system@id"))
		}
	}

	nodes := make([]GraphNode, 0, 1+len(selectedEvents))
	edges := make([]GraphEdge, 0, len(selectedEvents))
	systemID := ""
	if system != nil {
		systemID = stringField(system, "id")
		nodes = append(nodes, csapiSystemNode(system, firstNonEmpty(systemName, "GeoChat sender")))
	}
	facts := []GraphFact{
		{Subject: shortID(message.EntityID), Predicate: "cs-api.resource", Object: "SystemEvent", Source: "semconnect"},
	}
	for _, event := range selectedEvents {
		eventID := stringField(event, "id")
		nodes = append(nodes, GraphNode{
			ID:     eventID,
			Label:  firstNonEmpty(stringField(event, "eventType"), "GeoChat"),
			Kind:   "SystemEvent",
			Detail: firstNonEmpty(stringField(event, "message"), message.Text),
			Status: "event",
		})
		parentID := firstNonEmpty(systemID, stringField(event, "system@id"))
		if parentID != "" {
			edges = append(edges, GraphEdge{From: parentID, To: eventID, Label: "emits"})
		}
		if len(facts) < 10 {
			facts = append(facts,
				GraphFact{Subject: shortID(eventID), Predicate: "eventType", Object: firstNonEmpty(stringField(event, "eventType"), "GeoChat"), Source: "cs-api"},
				GraphFact{Subject: shortID(eventID), Predicate: "message", Object: firstNonEmpty(stringField(event, "message"), message.Text), Source: "cs-api"},
				GraphFact{Subject: shortID(eventID), Predicate: "payload.sender_uid", Object: firstNonEmpty(stringField(event, "payload.sender_uid"), message.SenderUID), Source: "cs-api"},
			)
		}
	}

	systemCount := 0
	if system != nil {
		systemCount = 1
	}
	return GraphLens{
		Source:  "csapi",
		Label:   "SemConnect Projection",
		Status:  "live",
		Summary: "CS API materialized TAK GeoChat view",
		Nodes:   nodes,
		Edges:   edges,
		Facts:   facts,
		Stats: []GraphStat{
			{Label: "systems", Value: strconv.Itoa(systemCount)},
			{Label: "events", Value: strconv.Itoa(len(selectedEvents))},
			{Label: "sender", Value: firstNonEmpty(message.SenderUID, "unresolved")},
		},
	}
}

func (s *Server) fetchCSAPIItems(ctx context.Context, base *url.URL, path string) ([]map[string]any, error) {
	endpoint := *base
	endpoint.Path = strings.TrimRight(base.Path, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("GET %s returned %d", path, resp.StatusCode)
	}
	var collection struct {
		Items []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return nil, err
	}
	return collection.Items, nil
}

func (s *Server) fetchCSAPIItem(ctx context.Context, base *url.URL, path string) (map[string]any, bool) {
	endpoint := *base
	endpoint.Path = strings.TrimRight(base.Path, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, false
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, false
	}
	var item map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&item); err != nil {
		return nil, false
	}
	if stringField(item, "id") == "" {
		return nil, false
	}
	return item, true
}

func (s *Server) fetchCSAPIObservationCount(ctx context.Context, base *url.URL, streamID string) (int, bool) {
	path := "/datastreams/" + url.PathEscape(streamID) + "/observations"
	endpoint := *base
	endpoint.Path = strings.TrimRight(base.Path, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return 0, false
	}
	req.Header.Set("Accept", "application/json")
	resp, err := s.client.Do(req)
	if err != nil {
		return 0, false
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return 0, false
	}
	var collection struct {
		NumberMatched int              `json:"numberMatched"`
		Items         []map[string]any `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return 0, false
	}
	if collection.NumberMatched > 0 {
		return collection.NumberMatched, true
	}
	return len(collection.Items), true
}

func csapiErrorLens(reason string) GraphLens {
	return GraphLens{
		Source:  "csapi",
		Label:   "SemConnect Projection",
		Status:  "unavailable",
		Summary: reason,
		Stats:   []GraphStat{{Label: "status", Value: "unavailable"}},
	}
}

func csapiDisabledLens() GraphLens {
	return GraphLens{
		Source:  "csapi",
		Label:   "SemConnect Projection",
		Status:  "disabled",
		Summary: "CS API bridge disabled",
		Nodes:   nil,
		Edges:   nil,
		Facts:   nil,
		Stats:   []GraphStat{{Label: "endpoint", Value: "not configured"}},
	}
}

func csapiPendingLens(summary, endpoint string) GraphLens {
	return GraphLens{
		Source:  "csapi",
		Label:   "SemConnect Projection",
		Status:  "pending",
		Summary: summary,
		Stats:   []GraphStat{{Label: "endpoint", Value: endpoint}},
	}
}

func csapiSystemNode(system map[string]any, fallback string) GraphNode {
	return GraphNode{
		ID:     stringField(system, "id"),
		Label:  firstNonEmpty(stringField(system, "name"), stringField(system, "properties.name"), fallback),
		Kind:   "System",
		Detail: "CS API",
		Status: "materialized",
	}
}

func findCSAPISystem(items []map[string]any, vehicle VehicleView) map[string]any {
	expectedID := fmt.Sprintf("c360.semconnect.systems.csapi.system.uav-%03d", vehicle.SystemID)
	token := fmt.Sprintf("uav-%03d", vehicle.SystemID)
	for _, item := range items {
		if stringField(item, "id") == expectedID {
			return item
		}
	}
	for _, item := range items {
		name := strings.EqualFold(stringField(item, "name"), vehicle.Callsign)
		idMatches := strings.Contains(strings.ToLower(stringField(item, "id")), token)
		if name || idMatches {
			return item
		}
	}
	return nil
}

func findCSAPICOPSystem(items []map[string]any, uid, name string) map[string]any {
	uid = strings.TrimSpace(uid)
	name = strings.TrimSpace(name)
	token := safeToken(firstNonEmpty(uid, name))
	for _, item := range items {
		if uid != "" && strings.EqualFold(stringField(item, "properties.uid"), uid) {
			return item
		}
	}
	for _, item := range items {
		if name != "" && (strings.EqualFold(stringField(item, "name"), name) || strings.EqualFold(stringField(item, "properties.name"), name)) {
			return item
		}
	}
	for _, item := range items {
		if token != "unknown" && strings.Contains(strings.ToLower(stringField(item, "id")), token) {
			return item
		}
	}
	return nil
}

func findCSAPICOPSamplingFeature(items []map[string]any, marker cop.View) map[string]any {
	for _, item := range items {
		if marker.UID != "" && strings.EqualFold(stringField(item, "properties.uid"), marker.UID) {
			return item
		}
	}
	for _, item := range items {
		if marker.Label != "" && (strings.EqualFold(stringField(item, "name"), marker.Label) || strings.EqualFold(stringField(item, "properties.name"), marker.Label)) {
			return item
		}
	}
	token := safeToken(firstNonEmpty(marker.UID, marker.Label, lastToken(marker.EntityID)))
	for _, item := range items {
		if token != "unknown" && strings.Contains(strings.ToLower(stringField(item, "id")), token) {
			return item
		}
	}
	return nil
}

func findMessageSender(snapshot Snapshot, message cop.View) cop.View {
	for _, operator := range snapshot.Operators {
		if message.SenderEntity != "" && operator.EntityID == message.SenderEntity {
			return operator
		}
		if message.SenderUID != "" && operator.UID == message.SenderUID {
			return operator
		}
	}
	return cop.View{}
}

func filterCSAPIByField(items []map[string]any, field, value string) []map[string]any {
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if stringField(item, field) == value {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterCSAPIByIDOrProperty(items []map[string]any, id, field, value string) []map[string]any {
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		fieldValue := stringField(item, field)
		if stringField(item, "id") == id ||
			fieldValue == value ||
			(value != "" && strings.Contains(strings.ToLower(fieldValue), strings.ToLower(value))) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterCSAPIByEventKind(items []map[string]any, kind string) []map[string]any {
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if strings.EqualFold(stringField(item, "eventType"), kind) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func findCSAPIByID(items []map[string]any, id string) map[string]any {
	for _, item := range items {
		if stringField(item, "id") == id {
			return item
		}
	}
	return nil
}

func findByIDContains(items []map[string]any, needle string) map[string]any {
	for _, item := range items {
		if strings.Contains(strings.ToLower(stringField(item, "id")), needle) {
			return item
		}
	}
	return nil
}

func containsCSAPIID(items []map[string]any, id string) bool {
	for _, item := range items {
		if stringField(item, "id") == id {
			return true
		}
	}
	return false
}

func countCSAPICommandsForControls(commands, controls []map[string]any) int {
	count := 0
	for _, command := range commands {
		if containsCSAPIID(controls, stringField(command, "controlstream@id")) {
			count++
		}
	}
	return count
}

func filterCSAPICommandsForControls(commands, controls []map[string]any) []map[string]any {
	filtered := make([]map[string]any, 0, len(commands))
	for _, command := range commands {
		if containsCSAPIID(controls, stringField(command, "controlstream@id")) {
			filtered = append(filtered, command)
		}
	}
	return filtered
}

func appendUniqueCSAPIItem(items []map[string]any, item map[string]any) []map[string]any {
	id := stringField(item, "id")
	if id == "" {
		return items
	}
	for _, existing := range items {
		if stringField(existing, "id") == id {
			return items
		}
	}
	return append(items, item)
}

func csapiCommandID(command CommandView) string {
	return "c360.semlink.robotics.csapi.command." + safeToken(lastToken(command.EntityID))
}

var csapiVehicleTelemetryKeys = []string{"battery", "altitude", "ground-speed"}

func csapiVehicleSystemID(vehicle VehicleView) string {
	return "c360.semconnect.systems.csapi.system." + csapiVehicleToken(vehicle)
}

func csapiVehicleDatastreamID(vehicle VehicleView, key string) string {
	return fmt.Sprintf("c360.semlink.robotics.csapi.datastream.%s-%s", safeToken(key), csapiVehicleToken(vehicle))
}

func csapiVehicleToken(vehicle VehicleView) string {
	if vehicle.SystemID > 0 {
		return fmt.Sprintf("uav-%03d", vehicle.SystemID)
	}
	return safeToken(vehicle.Callsign)
}

func csapiCOPSystemID(uid string) string {
	if strings.TrimSpace(uid) == "" {
		return ""
	}
	return "c360.semconnect.systems.csapi.system." + strings.TrimSpace(uid)
}

func csapiCOPSamplingFeatureID(uid string) string {
	if strings.TrimSpace(uid) == "" {
		return ""
	}
	return "c360.semconnect.systems.csapi.samplingfeature." + strings.TrimSpace(uid)
}

func csapiCOPDatastreamID(view cop.View) string {
	return "c360.semlink.cop.csapi.datastream.position-" + safeToken(lastToken(view.EntityID))
}

func csapiCOPEventID(view cop.View) string {
	return "c360.semlink.cop.csapi.event." + safeToken(lastToken(view.EntityID))
}

func stringField(item map[string]any, field string) string {
	if item == nil {
		return ""
	}
	value, ok := item[field]
	if ok {
		return objectString(value)
	}
	if !strings.Contains(field, ".") {
		return ""
	}
	var current any = item
	for _, part := range strings.Split(field, ".") {
		next, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current, ok = next[part]
		if !ok {
			return ""
		}
	}
	return objectString(current)
}

func commandVerb(command map[string]any) string {
	params, ok := command["params"].(map[string]any)
	if !ok {
		return ""
	}
	return objectString(params["verb"])
}

func objectString(value any) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case float64:
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'f', 2, 64)
	case float32:
		return strconv.FormatFloat(float64(v), 'f', 2, 32)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case bool:
		return strconv.FormatBool(v)
	default:
		raw, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(raw)
	}
}

func shortID(id string) string {
	if id == "" {
		return ""
	}
	if i := strings.LastIndex(id, "."); i >= 0 && i < len(id)-1 {
		return id[i+1:]
	}
	return id
}

func shortIRI(value string) string {
	if value == "" {
		return ""
	}
	if i := strings.LastIndexAny(value, "/#"); i >= 0 && i < len(value)-1 {
		return value[i+1:]
	}
	return value
}

func csapiObservationHistoryLabel(stream map[string]any) string {
	switch shortIRI(stringField(stream, "observedProperty")) {
	case "batteryRemaining":
		return "Battery history"
	case "altitude":
		return "Altitude history"
	case "groundSpeed":
		return "Ground speed history"
	case "operatorPosition":
		return "Position history"
	default:
		return "Observations"
	}
}

func boolText(value bool) string {
	if value {
		return "present"
	}
	return "missing"
}

func lastToken(id string) string {
	if i := strings.LastIndex(id, "."); i >= 0 {
		return id[i+1:]
	}
	return id
}

func safeToken(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	prevDash := false
	for _, r := range s {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-'
		if ok {
			b.WriteRune(r)
			prevDash = r == '-'
			continue
		}
		if !prevDash {
			b.WriteByte('-')
			prevDash = true
		}
	}
	out := strings.Trim(b.String(), "-_")
	if out == "" {
		return "unknown"
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func copDisplayName(item cop.View) string {
	switch item.Kind {
	case cop.KindOperator:
		return firstNonEmpty(item.Callsign, item.UID)
	case cop.KindMarker:
		return firstNonEmpty(item.Label, item.Description, item.UID)
	case cop.KindMessage:
		return firstNonEmpty(item.Text, item.Callsign, item.UID)
	default:
		return firstNonEmpty(item.Label, item.Callsign, item.Text, item.UID)
	}
}

func copKindLabel(kind cop.Kind) string {
	switch kind {
	case cop.KindOperator:
		return "Operator"
	case cop.KindMarker:
		return "Marker"
	case cop.KindMessage:
		return "GeoChat"
	default:
		return titleToken(string(kind))
	}
}

func titleToken(value string) string {
	value = strings.ReplaceAll(value, "_", "-")
	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '-' || r == '.' || r == ' '
	})
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func batteryStatus(value int) string {
	switch {
	case value <= 25:
		return "critical"
	case value <= 45:
		return "warning"
	default:
		return "nominal"
	}
}

func countCommandsForVehicle(commands []CommandView, vehicleID string) int {
	count := 0
	for _, command := range commands {
		if command.TargetEntity == vehicleID {
			count++
		}
	}
	return count
}
