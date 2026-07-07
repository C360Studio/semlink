package semops

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/c360studio/semlink/internal/handoff"
	"github.com/c360studio/semlink/internal/mavlink"
)

const (
	ReadbackContractV0 = "c360.semops.semlink.ardupilot.readback.v0"
	ReadbackPathV0     = "/api/cop/semlink/ardupilot/readback"

	DefaultTargetComponentID = 1
	DefaultReadbackTTL       = 30 * time.Second
	maxResponseBytes         = 1 << 20
)

type ReadbackRequest struct {
	Contract           string    `json:"contract"`
	CompanionNodeID    string    `json:"companion_node_id"`
	TargetSystemID     uint8     `json:"target_system_id"`
	TargetComponentID  uint8     `json:"target_component_id"`
	CommandID          uint16    `json:"command_id"`
	RequestedMessageID uint32    `json:"requested_message_id"`
	CorrelationID      string    `json:"correlation_id"`
	IdempotencyKey     string    `json:"idempotency_key"`
	RequestedAt        time.Time `json:"requested_at"`
	TTLSeconds         int       `json:"ttl_seconds"`
	SourceRef          string    `json:"source_ref,omitempty"`
}

type ReadbackResponse struct {
	Contract                  string     `json:"contract"`
	Accepted                  bool       `json:"accepted"`
	Status                    string     `json:"status"`
	Duplicate                 bool       `json:"duplicate"`
	CorrelationID             string     `json:"correlation_id,omitempty"`
	IdempotencyKey            string     `json:"idempotency_key,omitempty"`
	CompanionNodeID           string     `json:"companion_node_id,omitempty"`
	AuthorizedCompanionNodeID string     `json:"authorized_companion_node_id,omitempty"`
	AuthorityScope            string     `json:"authority_scope,omitempty"`
	TargetAssetID             string     `json:"target_asset_id,omitempty"`
	EntityID                  string     `json:"entity_id,omitempty"`
	NativeID                  string     `json:"native_id,omitempty"`
	ExistingNativeID          string     `json:"existing_native_id,omitempty"`
	ClaimScope                string     `json:"claim_scope,omitempty"`
	SourceRef                 string     `json:"source_ref,omitempty"`
	RequestedAt               *time.Time `json:"requested_at,omitempty"`
	ExpiresAt                 *time.Time `json:"expires_at,omitempty"`
	RejectedReason            string     `json:"rejected_reason,omitempty"`
	NativeExecutionAllowed    bool       `json:"native_execution_allowed"`
	CompanionTransmitAllowed  bool       `json:"companion_transmit_allowed"`
	Mutations                 int        `json:"mutations"`
	Error                     string     `json:"error,omitempty"`
}

type AutopilotVersionRequestInput struct {
	Profile           handoff.Profile
	CompanionNodeID   string
	TargetSystemID    uint8
	TargetComponentID uint8
	CorrelationID     string
	IdempotencyKey    string
	RequestedAt       time.Time
	TTL               time.Duration
	SourceRef         string
}

func NewAutopilotVersionRequest(input AutopilotVersionRequestInput) ReadbackRequest {
	companionNodeID := strings.TrimSpace(input.CompanionNodeID)
	if companionNodeID == "" {
		companionNodeID = strings.TrimSpace(input.Profile.NodeID)
	}
	targetComponentID := input.TargetComponentID
	if targetComponentID == 0 {
		targetComponentID = DefaultTargetComponentID
	}
	ttl := input.TTL
	if ttl == 0 {
		ttl = DefaultReadbackTTL
	}
	request := ReadbackRequest{
		Contract:           ReadbackContractV0,
		CompanionNodeID:    companionNodeID,
		TargetSystemID:     input.TargetSystemID,
		TargetComponentID:  targetComponentID,
		CommandID:          mavlink.MAVCmdRequestMessage,
		RequestedMessageID: uint32(mavlink.MessageAutopilotVersion),
		CorrelationID:      strings.TrimSpace(input.CorrelationID),
		IdempotencyKey:     strings.TrimSpace(input.IdempotencyKey),
		RequestedAt:        input.RequestedAt.UTC(),
		TTLSeconds:         int(ttl.Seconds()),
		SourceRef:          strings.TrimSpace(input.SourceRef),
	}
	if request.SourceRef == "" {
		request.SourceRef = SourceRef(request.CompanionNodeID, request.TargetSystemID, request.RequestedMessageID)
	}
	return request
}

func SourceRef(companionNodeID string, targetSystemID uint8, requestedMessageID uint32) string {
	message := fmt.Sprintf("request-message-%d", requestedMessageID)
	if requestedMessageID == uint32(mavlink.MessageAutopilotVersion) {
		message = "request-autopilot-version"
	}
	return fmt.Sprintf(
		"semlink://%s/ardupilot/system-%d/%s",
		safeToken(companionNodeID),
		targetSystemID,
		message,
	)
}

func (r ReadbackRequest) Validate() error {
	var problems []string
	if strings.TrimSpace(r.Contract) != ReadbackContractV0 {
		problems = append(problems, "contract must be "+ReadbackContractV0)
	}
	if strings.TrimSpace(r.CompanionNodeID) == "" {
		problems = append(problems, "companion_node_id is required")
	}
	if r.TargetSystemID == 0 {
		problems = append(problems, "target_system_id must be in MAVLink range 1..255")
	}
	if r.TargetComponentID == 0 {
		problems = append(problems, "target_component_id must be in MAVLink range 1..255")
	}
	if r.CommandID != mavlink.MAVCmdRequestMessage {
		problems = append(problems, fmt.Sprintf("command_id must be MAV_CMD_REQUEST_MESSAGE %d", mavlink.MAVCmdRequestMessage))
	}
	if r.RequestedMessageID != uint32(mavlink.MessageAutopilotVersion) {
		problems = append(problems, fmt.Sprintf("requested_message_id must be AUTOPILOT_VERSION %d", mavlink.MessageAutopilotVersion))
	}
	if strings.TrimSpace(r.CorrelationID) == "" {
		problems = append(problems, "correlation_id is required")
	}
	if strings.TrimSpace(r.IdempotencyKey) == "" {
		problems = append(problems, "idempotency_key is required")
	}
	if r.RequestedAt.IsZero() {
		problems = append(problems, "requested_at is required")
	}
	if r.TTLSeconds <= 0 {
		problems = append(problems, "ttl_seconds must be positive")
	}
	if len(problems) > 0 {
		return errors.New("invalid SemOps readback request: " + strings.Join(problems, "; "))
	}
	return nil
}

type Client struct {
	baseURL string
	client  *http.Client
	headers http.Header
}

type ClientConfig struct {
	BaseURL    string
	HTTPClient *http.Client
	Headers    http.Header
}

func NewClient(cfg ClientConfig) (*Client, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		return nil, errors.New("semops readback client: base URL is required")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, fmt.Errorf("semops readback client: invalid base URL %q", cfg.BaseURL)
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 3 * time.Second}
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		client:  client,
		headers: cloneHeader(cfg.Headers),
	}, nil
}

func (c *Client) SubmitReadback(ctx context.Context, request ReadbackRequest) (ReadbackResponse, error) {
	if c == nil {
		return ReadbackResponse{}, errors.New("semops readback client is nil")
	}
	if err := request.Validate(); err != nil {
		return ReadbackResponse{}, err
	}
	body, err := json.Marshal(request)
	if err != nil {
		return ReadbackResponse{}, fmt.Errorf("marshal SemOps readback request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+ReadbackPathV0, bytes.NewReader(body))
	if err != nil {
		return ReadbackResponse{}, fmt.Errorf("build SemOps readback request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	for key, values := range c.headers {
		for _, value := range values {
			httpReq.Header.Add(key, value)
		}
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return ReadbackResponse{}, fmt.Errorf("post SemOps readback request: %w", err)
	}
	defer resp.Body.Close()

	var response ReadbackResponse
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes))
	if err := decoder.Decode(&response); err != nil {
		return response, fmt.Errorf("decode SemOps readback response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return response, fmt.Errorf("SemOps readback response status %d: %s", resp.StatusCode, firstNonEmpty(response.Error, response.RejectedReason, response.Status))
	}
	return response, nil
}

func cloneHeader(in http.Header) http.Header {
	if len(in) == 0 {
		return nil
	}
	out := make(http.Header, len(in))
	for key, values := range in {
		out[key] = append([]string(nil), values...)
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return "unexpected response"
}

func safeToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var builder strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			builder.WriteRune(r)
			lastDash = false
		default:
			if !lastDash {
				builder.WriteByte('-')
				lastDash = true
			}
		}
	}
	token := strings.Trim(builder.String(), "-")
	if token == "" {
		return "unknown"
	}
	return token
}
