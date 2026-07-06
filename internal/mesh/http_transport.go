package mesh

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
)

const (
	HTTPWatermarksPath = "/mesh/v1/watermarks"
	HTTPDiffPath       = "/mesh/v1/diff"
)

const defaultHTTPTimeout = 5 * time.Second

type HTTPTransportConfig struct {
	Client *http.Client
}

type HTTPTransport struct {
	index  *SummaryIndex
	client *http.Client
}

type DiffRequest struct {
	Watermarks WatermarkSet `json:"watermarks"`
	Options    DiffOptions  `json:"options"`
}

type PullResult struct {
	Diff    Diff `json:"diff"`
	Applied int  `json:"applied"`
}

func NewHTTPTransport(index *SummaryIndex, cfg HTTPTransportConfig) (*HTTPTransport, error) {
	if index == nil {
		return nil, errors.New("mesh http transport: nil summary index")
	}
	client := cfg.Client
	if client == nil {
		client = &http.Client{Timeout: defaultHTTPTimeout}
	}
	return &HTTPTransport{index: index, client: client}, nil
}

func (t *HTTPTransport) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc(HTTPWatermarksPath, t.handleWatermarks)
	mux.HandleFunc(HTTPDiffPath, t.handleDiff)
	return mux
}

func (t *HTTPTransport) PullFrom(ctx context.Context, peerBaseURL string, opts DiffOptions) (PullResult, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	endpoint, err := peerEndpoint(peerBaseURL, HTTPDiffPath)
	if err != nil {
		return PullResult{}, err
	}
	reqBody, err := json.Marshal(DiffRequest{
		Watermarks: t.index.Watermarks(opts.Now),
		Options:    opts,
	})
	if err != nil {
		return PullResult{}, fmt.Errorf("marshal mesh diff request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(reqBody))
	if err != nil {
		return PullResult{}, fmt.Errorf("build mesh diff request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return PullResult{}, fmt.Errorf("request mesh diff from peer: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return PullResult{}, fmt.Errorf("request mesh diff from peer: %s: %s", resp.Status, responseSnippet(resp.Body))
	}

	var diff Diff
	if err := json.NewDecoder(resp.Body).Decode(&diff); err != nil {
		return PullResult{}, fmt.Errorf("decode mesh diff response: %w", err)
	}
	applied := 0
	for _, item := range diff.Items {
		if err := t.index.Upsert(item); err != nil {
			return PullResult{}, fmt.Errorf("apply mesh diff item %s: %w", item.Envelope.OperationID, err)
		}
		applied++
	}
	return PullResult{Diff: diff, Applied: applied}, nil
}

func (t *HTTPTransport) handleWatermarks(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	now, err := parseOptionalTime(r.URL.Query().Get("now"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeHTTPJSON(w, t.index.Watermarks(now))
}

func (t *HTTPTransport) handleDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	var req DiffRequest
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("invalid mesh diff request: %v", err), http.StatusBadRequest)
		return
	}
	diff, err := t.index.Diff(req.Watermarks, req.Options)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeHTTPJSON(w, diff)
}

func peerEndpoint(base, path string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("parse mesh peer url: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("mesh peer url must be absolute, got %q", base)
	}
	u.Path = strings.TrimRight(u.Path, "/") + path
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

func parseOptionalTime(raw string) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, nil
	}
	now, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid now timestamp: %w", err)
	}
	return now, nil
}

func writeHTTPJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func responseSnippet(body io.Reader) string {
	data, _ := io.ReadAll(io.LimitReader(body, 512))
	return strings.TrimSpace(string(data))
}
