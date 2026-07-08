package gcs

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/c360studio/semlink/internal/blueos"
	"github.com/c360studio/semlink/internal/handoff"
	"github.com/c360studio/semlink/internal/mesh"
)

type Server struct {
	store          *Store
	commands       *CommandService
	graph          EntityQuerier
	csapiURL       string
	client         *http.Client
	static         string
	blueos         blueos.Registration
	nodeID         string
	mesh           *mesh.SummaryIndex
	handoffProfile *handoff.Profile
}

type ServerOptions struct {
	Graph              EntityQuerier
	CSAPIURL           string
	HTTPClient         *http.Client
	BlueOSRegistration *blueos.Registration
	NodeID             string
	MeshIndex          *mesh.SummaryIndex
	HandoffProfile     *handoff.Profile
}

func NewServer(store *Store, commands *CommandService, staticDir string, opts ServerOptions) *Server {
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Second}
	}
	registration := blueos.DefaultRegistration()
	if opts.BlueOSRegistration != nil {
		registration = *opts.BlueOSRegistration
	}
	profile := opts.HandoffProfile
	if profile != nil {
		copyProfile := *profile
		copyProfile.MeshPeers = append([]string(nil), profile.MeshPeers...)
		profile = &copyProfile
	}
	return &Server{
		store:          store,
		commands:       commands,
		graph:          opts.Graph,
		csapiURL:       opts.CSAPIURL,
		client:         client,
		static:         staticDir,
		blueos:         registration,
		nodeID:         opts.NodeID,
		mesh:           opts.MeshIndex,
		handoffProfile: profile,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/register_service", s.handleRegisterService)
	mux.HandleFunc("/api/snapshot", s.handleSnapshot)
	mux.HandleFunc("/api/evidence", s.handleEvidence)
	mux.HandleFunc("/api/graph", s.handleGraph)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/commands", s.handleCommands)
	mux.HandleFunc("/", s.handleStatic)
	return withCORS(mux)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, map[string]any{"ok": true, "time": time.Now()})
}

func (s *Server) handleRegisterService(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := s.blueos.Validate(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, s.blueos)
}

func (s *Server) handleSnapshot(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, s.store.Snapshot())
}

func (s *Server) handleEvidence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	writeJSON(w, s.EvidenceBundle(time.Now()))
}

func (s *Server) handleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	entityID := r.URL.Query().Get("entity_id")
	if entityID == "" {
		entityID = r.URL.Query().Get("vehicle_id")
	}
	view, err := s.BuildGraphView(r.Context(), entityID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, view)
}

func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			data, err := json.Marshal(s.store.Snapshot())
			if err != nil {
				return
			}
			if _, err := fmt.Fprintf(w, "event: snapshot\ndata: %s\n\n", data); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func (s *Server) handleCommands(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		VehicleID string `json:"vehicle_id"`
		Verb      string `json:"verb"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid command request", http.StatusBadRequest)
		return
	}
	if s.rejectHandoffHardwareCommand(w, req.VehicleID, req.Verb) {
		return
	}
	cmd, err := s.commands.Submit(r.Context(), req.VehicleID, req.Verb)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, cmd)
}

func (s *Server) handleStatic(w http.ResponseWriter, r *http.Request) {
	if s.static == "" {
		http.NotFound(w, r)
		return
	}
	path := filepath.Join(s.static, filepath.Clean(r.URL.Path))
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		http.ServeFile(w, r, path)
		return
	}
	index := filepath.Join(s.static, "index.html")
	if _, err := os.Stat(index); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.ServeFile(w, r, index)
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
