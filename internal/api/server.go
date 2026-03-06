package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"osg/internal/config"
)

// Server is the interactions API HTTP server.
type Server struct {
	store        *Store
	commentStore *CommentStore
	authHandlers *AuthHandlers
	commentH     *CommentHandlers
	mux          *http.ServeMux
	logger       *slog.Logger
	cfg          config.InteractionsConfig
}

// NewServer creates a new interactions API server.
// commentStore may be nil if comments are disabled.
func NewServer(store *Store, cfg config.InteractionsConfig, logger *slog.Logger, commentStore *CommentStore, authProviders map[string]*AuthProvider) *Server {
	if logger == nil {
		logger = slog.Default()
	}
	s := &Server{
		store:        store,
		commentStore: commentStore,
		mux:          http.NewServeMux(),
		logger:       logger,
		cfg:          cfg,
	}

	if commentStore != nil && len(authProviders) > 0 {
		secureCookies := cfg.Comments.AuthCallbackURL != "" && strings.HasPrefix(cfg.Comments.AuthCallbackURL, "https")
		s.authHandlers = NewAuthHandlers(commentStore, authProviders, logger, secureCookies)
		s.commentH = NewCommentHandlers(commentStore, logger)
	}

	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("POST /api/v1/pageview", s.handlePageView)
	s.mux.HandleFunc("POST /api/v1/vote", s.handleVote)
	// Health check.
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)

	// Comment and auth routes (only when comments are enabled).
	if s.authHandlers != nil {
		// Auth: login, callback, me, logout.
		// Note: Go 1.22+ ServeMux matches literal paths before wildcards,
		// so "GET /api/v1/auth/me" takes precedence over "GET /api/v1/auth/{provider}".
		s.mux.HandleFunc("GET /api/v1/auth/me", s.authHandlers.HandleMe)
		s.mux.HandleFunc("POST /api/v1/auth/logout", s.authHandlers.HandleLogout)
		s.mux.HandleFunc("GET /api/v1/auth/{provider}", s.authHandlers.HandleLogin)
		s.mux.HandleFunc("GET /api/v1/auth/{provider}/callback", s.authHandlers.HandleCallback)

		// Comments: list, create, delete.
		s.mux.HandleFunc("GET /api/v1/comments", s.commentH.HandleList)
		s.mux.HandleFunc("POST /api/v1/comments", s.commentH.HandleCreate)
		s.mux.HandleFunc("DELETE /api/v1/comments/{id}", s.commentH.HandleDelete)
	}
}

// Handler returns the http.Handler with CORS middleware applied.
func (s *Server) Handler() http.Handler {
	hasComments := s.commentStore != nil
	return CORSMiddleware(s.cfg.CORSOrigins, hasComments, s.mux)
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
	_ = json.NewEncoder(w).Encode(data)
}
