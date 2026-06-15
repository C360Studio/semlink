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

	"github.com/c360studio/semlink/internal/projector"
	"github.com/c360studio/semstreams/graph"
	"github.com/c360studio/semstreams/message"
)

type EntityQuerier interface {
	QueryEntity(ctx context.Context, id string) (*graph.EntityState, error)
}

type GraphView struct {
	GeneratedAt time.Time   `json:"generated_at"`
	VehicleID   string      `json:"vehicle_id"`
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

func (s *Server) BuildGraphView(ctx context.Context, vehicleID string) (GraphView, error) {
	snapshot := s.store.Snapshot()
	vehicle, ok := findVehicle(snapshot.Vehicles, vehicleID)
	if !ok {
		return GraphView{}, errors.New("vehicle not found")
	}

	view := GraphView{
		GeneratedAt: time.Now(),
		VehicleID:   vehicle.EntityID,
		Lenses: []GraphLens{
			s.buildSemLinkLens(ctx, snapshot, vehicle),
			s.buildCSAPILens(ctx, snapshot, vehicle),
		},
	}
	return view, nil
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

	base, err := url.Parse(strings.TrimRight(s.csapiURL, "/"))
	if err != nil || base.Scheme == "" || base.Host == "" {
		return csapiErrorLens("invalid endpoint")
	}

	systems, err := s.fetchCSAPIItems(ctx, base, "/systems")
	if err != nil {
		return csapiErrorLens(err.Error())
	}
	system := findCSAPISystem(systems, vehicle)
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

	datastreams, _ := s.fetchCSAPIItems(ctx, base, "/datastreams")
	selectedStreams := filterCSAPIByField(datastreams, "system@id", systemID)
	if len(selectedStreams) == 0 {
		selectedStreams = filterCSAPIByField(datastreams, "system", systemID)
	}
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

	if battery := findByIDContains(selectedStreams, "battery"); battery != nil {
		streamID := stringField(battery, "id")
		if count, ok := s.fetchCSAPIObservationCount(ctx, base, streamID); ok {
			obsID := streamID + "#observations"
			nodes = append(nodes, GraphNode{
				ID:     obsID,
				Label:  "Observations",
				Kind:   "History",
				Detail: fmt.Sprintf("%d records", count),
				Status: "historical",
			})
			edges = append(edges, GraphEdge{From: streamID, To: obsID, Label: "records"})
			facts = append(facts, GraphFact{Subject: shortID(obsID), Predicate: "numberMatched", Object: strconv.Itoa(count), Source: "cs-api"})
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

func filterCSAPIByField(items []map[string]any, field, value string) []map[string]any {
	filtered := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if stringField(item, field) == value {
			filtered = append(filtered, item)
		}
	}
	return filtered
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

func stringField(item map[string]any, field string) string {
	if item == nil {
		return ""
	}
	value, ok := item[field]
	if !ok {
		return ""
	}
	return objectString(value)
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
