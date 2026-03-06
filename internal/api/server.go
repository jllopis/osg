package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"osg/internal/config"
)

// Server is the interactions API HTTP server.
type Server struct {
	store  *Store
	mux    *http.ServeMux
	logger *slog.Logger
	cfg    config.InteractionsConfig
}

// NewServer creates a new interactions API server.
func NewServer(store *Store, cfg config.InteractionsConfig, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{
		store:  store,
		mux:    http.NewServeMux(),
		logger: logger,
		cfg:    cfg,
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /api/v1/pageview", s.handlePageView)
	s.mux.HandleFunc("POST /api/v1/vote", s.handleVote)
	// Health check.
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
}

// Handler returns the http.Handler with CORS middleware applied.
func (s *Server) Handler() http.Handler {
	return CORSMiddleware(s.cfg.CORSOrigins, s.mux)
}

func (s *Server) handlePageView(w http.ResponseWriter, r *http.Request) {
	var req PageViewRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if err := req.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	stats, err := s.store.RecordView(req.Path, req.Fingerprint)
	if err != nil {
		s.logger.Error("record view", "error", err, "path", req.Path)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleVote(w http.ResponseWriter, r *http.Request) {
	var req VoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if err := req.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	stats, err := s.store.Vote(req.Path, req.Fingerprint, req.Vote)
	if err != nil {
		s.logger.Error("record vote", "error", err, "path", req.Path)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
