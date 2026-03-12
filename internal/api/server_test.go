package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"osg/internal/config"
)

func testServer(t *testing.T) (*Server, *Store) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath, 24)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	cfg := config.InteractionsConfig{
		Enabled:        true,
		ViewDedupHours: 24,
	}
	logger := slog.Default()
	srv := NewServer(store, cfg, logger, nil, nil, nil)
	return srv, store
}

func postJSON(t *testing.T, handler http.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func TestHandleHealth(t *testing.T) {
	srv, _ := testServer(t)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var resp map[string]string
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp["status"] != "ok" {
		t.Errorf("status = %q, want ok", resp["status"])
	}
}

func TestHandlePageView_Success(t *testing.T) {
	srv, _ := testServer(t)
	rec := postJSON(t, srv.Handler(), "/api/v1/pageview", PageViewRequest{
		Path:        "/posts/hello",
		Fingerprint: "abc123",
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var stats Stats
	_ = json.NewDecoder(rec.Body).Decode(&stats)
	if stats.Views != 1 {
		t.Errorf("views = %d, want 1", stats.Views)
	}
}

func TestHandlePageView_MissingPath(t *testing.T) {
	srv, _ := testServer(t)
	rec := postJSON(t, srv.Handler(), "/api/v1/pageview", PageViewRequest{
		Path:        "",
		Fingerprint: "abc123",
	})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandlePageView_MissingFingerprint(t *testing.T) {
	srv, _ := testServer(t)
	rec := postJSON(t, srv.Handler(), "/api/v1/pageview", PageViewRequest{
		Path:        "/posts/hello",
		Fingerprint: "",
	})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandlePageView_InvalidJSON(t *testing.T) {
	srv, _ := testServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/pageview",
		bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleVote_Like(t *testing.T) {
	srv, _ := testServer(t)
	rec := postJSON(t, srv.Handler(), "/api/v1/vote", VoteRequest{
		Path:        "/posts/hello",
		Fingerprint: "abc123",
		Vote:        1,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var stats Stats
	_ = json.NewDecoder(rec.Body).Decode(&stats)
	if stats.Likes != 1 {
		t.Errorf("likes = %d, want 1", stats.Likes)
	}
	if stats.UserVote != 1 {
		t.Errorf("user_vote = %d, want 1", stats.UserVote)
	}
}

func TestHandleVote_InvalidVote(t *testing.T) {
	srv, _ := testServer(t)
	rec := postJSON(t, srv.Handler(), "/api/v1/vote", VoteRequest{
		Path:        "/posts/hello",
		Fingerprint: "abc123",
		Vote:        5,
	})

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
}

func TestHandleVote_Retract(t *testing.T) {
	srv, _ := testServer(t)
	// First vote.
	postJSON(t, srv.Handler(), "/api/v1/vote", VoteRequest{
		Path: "/posts/hello", Fingerprint: "abc123", Vote: 1,
	})
	// Retract.
	rec := postJSON(t, srv.Handler(), "/api/v1/vote", VoteRequest{
		Path: "/posts/hello", Fingerprint: "abc123", Vote: 0,
	})

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	var stats Stats
	_ = json.NewDecoder(rec.Body).Decode(&stats)
	if stats.Likes != 0 {
		t.Errorf("likes = %d, want 0", stats.Likes)
	}
	if stats.UserVote != 0 {
		t.Errorf("user_vote = %d, want 0", stats.UserVote)
	}
}

func TestHandlePageView_ReturnsVoteInfo(t *testing.T) {
	srv, _ := testServer(t)

	// Vote first.
	postJSON(t, srv.Handler(), "/api/v1/vote", VoteRequest{
		Path: "/posts/hello", Fingerprint: "abc123", Vote: 1,
	})
	// Then record a view — response should include the vote.
	rec := postJSON(t, srv.Handler(), "/api/v1/pageview", PageViewRequest{
		Path: "/posts/hello", Fingerprint: "abc123",
	})

	var stats Stats
	_ = json.NewDecoder(rec.Body).Decode(&stats)
	if stats.UserVote != 1 {
		t.Errorf("user_vote = %d, want 1", stats.UserVote)
	}
	if stats.Likes != 1 {
		t.Errorf("likes = %d, want 1", stats.Likes)
	}
}

// CORS tests.

func TestCORS_NoOriginHeader(t *testing.T) {
	srv, _ := testServer(t)
	// Override cfg to add CORS origins.
	srv.cfg.CORSOrigins = []string{"https://example.com"}

	rec := postJSON(t, srv.Handler(), "/api/v1/pageview", PageViewRequest{
		Path: "/posts/hello", Fingerprint: "abc123",
	})
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("expected no CORS header when no Origin is sent")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	cfg := config.InteractionsConfig{
		Enabled:     true,
		CORSOrigins: []string{"https://example.com"},
	}
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, _ := NewStore(dbPath, 24)
	defer func() { _ = store.Close() }()
	srv := NewServer(store, cfg, slog.Default(), nil, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/pageview",
		bytes.NewReader([]byte(`{"path":"/a","fp":"x"}`)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://example.com")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Errorf("CORS origin = %q, want https://example.com", rec.Header().Get("Access-Control-Allow-Origin"))
	}
}

func TestCORS_Preflight(t *testing.T) {
	cfg := config.InteractionsConfig{
		Enabled:     true,
		CORSOrigins: []string{"https://example.com"},
	}
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, _ := NewStore(dbPath, 24)
	defer func() { _ = store.Close() }()
	srv := NewServer(store, cfg, slog.Default(), nil, nil, nil)

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/pageview", nil)
	req.Header.Set("Origin", "https://example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Methods") == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}
}
