package csapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/c360studio/semlink/internal/gcs"
)

const (
	defaultInterval            = 2 * time.Second
	defaultObservationInterval = 5 * time.Second
	defaultRequestTimeout      = 3 * time.Second
	bridgeUser                 = "semlink-demo"
)

type Snapshotter interface {
	Snapshot() gcs.Snapshot
}

type Config struct {
	BaseURL             string
	Interval            time.Duration
	ObservationInterval time.Duration
	RequestTimeout      time.Duration
	HTTPClient          *http.Client
	Logger              *slog.Logger
}

type Bridge struct {
	cfg     Config
	store   Snapshotter
	client  *http.Client
	logger  *slog.Logger
	baseURL string

	mu               sync.Mutex
	systems          map[string]string
	datastreams      map[string]map[string]string
	lastObservations map[string]time.Time
	alertsPosted     map[string]struct{}
	controlStreams   map[string]string
	commandsPosted   map[string]struct{}
}

type metricSpec struct {
	Key              string
	Label            string
	ObservedProperty string
	UOMCode          string
	Value            func(gcs.VehicleView) float64
}

var telemetryMetrics = []metricSpec{
	{
		Key:              "battery",
		Label:            "Battery remaining",
		ObservedProperty: "https://c360.studio/def/robot/power/batteryRemaining",
		UOMCode:          "%",
		Value: func(v gcs.VehicleView) float64 {
			return float64(v.BatteryRemaining)
		},
	},
	{
		Key:              "altitude",
		Label:            "Altitude",
		ObservedProperty: "https://c360.studio/def/robot/position/altitude",
		UOMCode:          "m",
		Value: func(v gcs.VehicleView) float64 {
			return v.AltitudeM
		},
	},
	{
		Key:              "ground-speed",
		Label:            "Ground speed",
		ObservedProperty: "https://c360.studio/def/robot/motion/groundSpeed",
		UOMCode:          "m/s",
		Value: func(v gcs.VehicleView) float64 {
			return v.GroundSpeedMS
		},
	},
}

func NewBridge(cfg Config, store Snapshotter) (*Bridge, error) {
	if store == nil {
		return nil, errors.New("csapi bridge: nil snapshot store")
	}
	if strings.TrimSpace(cfg.BaseURL) == "" {
		return nil, errors.New("csapi bridge: base URL is required")
	}
	parsed, err := url.Parse(cfg.BaseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("csapi bridge: invalid base URL %q", cfg.BaseURL)
	}
	if cfg.Interval <= 0 {
		cfg.Interval = defaultInterval
	}
	if cfg.ObservationInterval <= 0 {
		cfg.ObservationInterval = defaultObservationInterval
	}
	if cfg.RequestTimeout <= 0 {
		cfg.RequestTimeout = defaultRequestTimeout
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{}
	}
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}

	return &Bridge{
		cfg:              cfg,
		store:            store,
		client:           client,
		logger:           logger,
		baseURL:          strings.TrimRight(cfg.BaseURL, "/"),
		systems:          make(map[string]string),
		datastreams:      make(map[string]map[string]string),
		lastObservations: make(map[string]time.Time),
		alertsPosted:     make(map[string]struct{}),
		controlStreams:   make(map[string]string),
		commandsPosted:   make(map[string]struct{}),
	}, nil
}

func (b *Bridge) Start(ctx context.Context) {
	go b.run(ctx)
}

func (b *Bridge) run(ctx context.Context) {
	ticker := time.NewTicker(b.cfg.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := b.Sync(ctx); err != nil {
				b.logger.Debug("cs api bridge sync failed", slog.Any("error", err))
			}
		}
	}
}

func (b *Bridge) Sync(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	snapshot := b.store.Snapshot()
	vehicleByEntity := make(map[string]gcs.VehicleView, len(snapshot.Vehicles))
	for _, vehicle := range snapshot.Vehicles {
		vehicleByEntity[vehicle.EntityID] = vehicle
		if vehicle.LastSeen.IsZero() {
			continue
		}
		systemID, err := b.ensureSystem(ctx, vehicle)
		if err != nil {
			return err
		}
		if err := b.ensureDatastreams(ctx, systemID, vehicle); err != nil {
			return err
		}
		if err := b.postDueObservations(ctx, vehicle); err != nil {
			return err
		}
	}
	for _, alert := range snapshot.Alerts {
		if err := b.postAlertEvent(ctx, vehicleByEntity, alert); err != nil {
			return err
		}
	}
	for _, command := range snapshot.Commands {
		if err := b.postCommand(ctx, vehicleByEntity, command); err != nil {
			return err
		}
	}
	return nil
}

func (b *Bridge) ensureSystem(ctx context.Context, vehicle gcs.VehicleView) (string, error) {
	if id := b.systems[vehicle.EntityID]; id != "" {
		return id, nil
	}
	body := map[string]any{
		"type": "Feature",
		"geometry": map[string]any{
			"type":        "Point",
			"coordinates": []float64{vehicle.LongitudeDeg, vehicle.LatitudeDeg, vehicle.AltitudeM},
		},
		"properties": map[string]any{
			"uid":         vehicleToken(vehicle),
			"name":        vehicle.Callsign,
			"description": fmt.Sprintf("SemLink MAVLink vehicle %s projected into CS API from SemStreams governed state.", vehicle.Callsign),
		},
	}
	result, err := b.postJSON(ctx, "/systems", string(mediaJSON), body)
	if err != nil {
		return "", err
	}
	if result.ID == "" {
		return "", errors.New("csapi bridge: POST /systems returned no id")
	}
	b.systems[vehicle.EntityID] = result.ID
	return result.ID, nil
}

func (b *Bridge) ensureDatastreams(ctx context.Context, systemID string, vehicle gcs.VehicleView) error {
	if _, ok := b.datastreams[vehicle.EntityID]; !ok {
		b.datastreams[vehicle.EntityID] = make(map[string]string, len(telemetryMetrics))
	}
	for _, spec := range telemetryMetrics {
		if b.datastreams[vehicle.EntityID][spec.Key] != "" {
			continue
		}
		id := datastreamID(vehicle, spec)
		body := map[string]any{
			"id":               id,
			"name":             fmt.Sprintf("%s %s", vehicle.Callsign, strings.ToLower(spec.Label)),
			"description":      fmt.Sprintf("Decimated SemLink MAVLink %s telemetry for %s.", strings.ToLower(spec.Label), vehicle.Callsign),
			"system":           systemID,
			"observedProperty": spec.ObservedProperty,
			"phenomenonTime":   rfc3339(vehicle.LastSeen),
			"resultTime":       rfc3339(vehicle.LastSeen),
			"schema":           quantitySchema(spec.UOMCode),
		}
		if _, err := b.postJSON(ctx, "/datastreams", string(mediaJSON), body); err != nil {
			return err
		}
		b.datastreams[vehicle.EntityID][spec.Key] = id
	}
	return nil
}

func (b *Bridge) postDueObservations(ctx context.Context, vehicle gcs.VehicleView) error {
	streams := b.datastreams[vehicle.EntityID]
	if len(streams) == 0 {
		return nil
	}
	for _, spec := range telemetryMetrics {
		streamID := streams[spec.Key]
		if streamID == "" {
			continue
		}
		if last := b.lastObservations[streamID]; !last.IsZero() && !vehicle.LastSeen.After(last.Add(b.cfg.ObservationInterval)) {
			continue
		}
		body := map[string]any{
			"id":               observationID(vehicle, spec),
			"procedure":        "urn:c360:semlink:mavlink-projector",
			"observedProperty": spec.ObservedProperty,
			"resultTime":       rfc3339(vehicle.LastSeen),
			"result":           spec.Value(vehicle),
		}
		path := "/datastreams/" + streamID + "/observations"
		if _, err := b.postJSON(ctx, path, mediaOMS, body); err != nil {
			return err
		}
		b.lastObservations[streamID] = vehicle.LastSeen
	}
	return nil
}

func (b *Bridge) postAlertEvent(ctx context.Context, vehicles map[string]gcs.VehicleView, alert gcs.AlertView) error {
	if _, ok := b.alertsPosted[alert.EntityID]; ok {
		return nil
	}
	vehicle, ok := vehicles[alert.SubjectEntity]
	if !ok {
		return nil
	}
	systemID, err := b.ensureSystem(ctx, vehicle)
	if err != nil {
		return err
	}
	body := map[string]any{
		"id":        eventID(alert),
		"eventTime": rfc3339(alert.RaisedAt),
		"eventType": titleToken(alert.Kind),
		"message":   alert.Message,
		"source":    "semlink-gcs",
		"payload": map[string]any{
			"kind":            alert.Kind,
			"severity":        alert.Severity,
			"active":          alert.Active,
			"semlink_alert":   alert.EntityID,
			"subject_entity":  alert.SubjectEntity,
			"graph_revision":  alert.GraphRevision,
			"indexing_policy": "control",
		},
	}
	if _, err := b.postJSON(ctx, "/systems/"+systemID+"/events", string(mediaJSON), body); err != nil {
		return err
	}
	b.alertsPosted[alert.EntityID] = struct{}{}
	return nil
}

func (b *Bridge) postCommand(ctx context.Context, vehicles map[string]gcs.VehicleView, command gcs.CommandView) error {
	if _, ok := b.commandsPosted[command.EntityID]; ok {
		return nil
	}
	vehicle, ok := vehicles[command.TargetEntity]
	if !ok {
		return nil
	}
	systemID, err := b.ensureSystem(ctx, vehicle)
	if err != nil {
		return err
	}
	controlStreamID, err := b.ensureControlStream(ctx, systemID, vehicle)
	if err != nil {
		return err
	}
	body := map[string]any{
		"id":               commandID(command),
		"controlstream@id": controlStreamID,
		"issueTime":        rfc3339(command.RequestedAt),
		"status":           command.Status,
		"sender":           "semlink-gcs",
		"params": map[string]any{
			"verb":            command.Verb,
			"targetEntity":    command.TargetEntity,
			"semlink_command": command.EntityID,
			"graph_revision":  command.GraphRevision,
		},
	}
	if _, err := b.postJSON(ctx, "/commands", string(mediaJSON), body); err != nil {
		return err
	}
	b.commandsPosted[command.EntityID] = struct{}{}
	return nil
}

func (b *Bridge) ensureControlStream(ctx context.Context, systemID string, vehicle gcs.VehicleView) (string, error) {
	if id := b.controlStreams[vehicle.EntityID]; id != "" {
		return id, nil
	}
	id := controlStreamID(vehicle)
	body := map[string]any{
		"id":          id,
		"name":        fmt.Sprintf("%s GCS command intent", vehicle.Callsign),
		"description": "SemLink operator command-intent metadata. Device-side execution remains owned by SemLink.",
		"system@id":   systemID,
		"inputName":   "gcs-command",
		"issueTime":   rfc3339(time.Now()),
		"async":       false,
		"schema": map[string]any{
			"commandFormat": "application/json",
			"parametersSchema": map[string]any{
				"type": "DataRecord",
				"fields": []map[string]any{
					{"name": "verb", "type": "Text", "label": "Command verb"},
					{"name": "targetEntity", "type": "Text", "label": "SemLink target entity"},
				},
			},
		},
	}
	if _, err := b.postJSON(ctx, "/controlstreams", string(mediaJSON), body); err != nil {
		return "", err
	}
	b.controlStreams[vehicle.EntityID] = id
	return id, nil
}

type postResult struct {
	ID string
}

func (b *Bridge) postJSON(ctx context.Context, path, contentType string, body any) (postResult, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return postResult{}, err
	}
	reqCtx, cancel := context.WithTimeout(ctx, b.cfg.RequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, b.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return postResult{}, err
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Accept", string(mediaJSON))
	req.Header.Set("X-Forwarded-User", bridgeUser)
	req.Header.Set("X-Forwarded-Email", "semlink-demo@localhost")

	resp, err := b.client.Do(req)
	if err != nil {
		return postResult{}, err
	}
	defer resp.Body.Close()
	out, err := decodePostResult(resp)
	if err != nil {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return postResult{}, fmt.Errorf("csapi bridge POST %s: %w: %s", path, err, strings.TrimSpace(string(raw)))
	}
	return out, nil
}

func decodePostResult(resp *http.Response) (postResult, error) {
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusConflict {
		return postResult{}, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	var body struct {
		ID string `json:"id"`
	}
	if resp.Body != nil {
		_ = json.NewDecoder(resp.Body).Decode(&body)
	}
	id := body.ID
	if id == "" {
		id = idFromLocation(resp.Header.Get("Location"))
	}
	if id == "" {
		id = resp.Header.Get("X-CS-Attempted-ID")
	}
	if resp.StatusCode == http.StatusConflict && id == "" {
		return postResult{}, errors.New("conflict without resource id")
	}
	return postResult{ID: id}, nil
}

func idFromLocation(loc string) string {
	loc = strings.TrimSpace(loc)
	if loc == "" {
		return ""
	}
	if i := strings.LastIndex(loc, "/"); i >= 0 {
		return loc[i+1:]
	}
	return loc
}

const (
	mediaJSON = "application/json"
	mediaOMS  = "application/om+json"
)

func quantitySchema(uomCode string) map[string]any {
	return map[string]any{
		"type": "DataRecord",
		"fields": []map[string]any{
			{"name": "time", "type": "Time"},
			{"name": "value", "type": "Quantity", "uomCode": uomCode},
		},
	}
}

func vehicleToken(vehicle gcs.VehicleView) string {
	if vehicle.SystemID > 0 {
		return fmt.Sprintf("uav-%03d", vehicle.SystemID)
	}
	return safeToken(vehicle.Callsign)
}

func datastreamID(vehicle gcs.VehicleView, spec metricSpec) string {
	return fmt.Sprintf("c360.semlink.robotics.csapi.datastream.%s-%s", safeToken(spec.Key), vehicleToken(vehicle))
}

func observationID(vehicle gcs.VehicleView, spec metricSpec) string {
	return fmt.Sprintf("semlink-%s-%s-%d", vehicleToken(vehicle), safeToken(spec.Key), vehicle.LastSeen.UnixMilli())
}

func eventID(alert gcs.AlertView) string {
	return "c360.semlink.robotics.csapi.event." + safeToken(lastToken(alert.EntityID))
}

func controlStreamID(vehicle gcs.VehicleView) string {
	return "c360.semlink.robotics.csapi.control." + vehicleToken(vehicle)
}

func commandID(command gcs.CommandView) string {
	return "c360.semlink.robotics.csapi.command." + safeToken(lastToken(command.EntityID))
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

func titleToken(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '-' || r == '_' || r == ' '
	})
	var b strings.Builder
	for _, part := range parts {
		if part == "" {
			continue
		}
		b.WriteString(strings.ToUpper(part[:1]))
		if len(part) > 1 {
			b.WriteString(strings.ToLower(part[1:]))
		}
	}
	if b.Len() == 0 {
		return "SystemChanged"
	}
	return b.String()
}

func rfc3339(t time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}
	return t.UTC().Format(time.RFC3339Nano)
}
