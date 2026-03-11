package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"osg/internal/config"
)

func TestDispatch_NoWebhooks(t *testing.T) {
	// Should not panic with empty webhooks.
	Dispatch(context.Background(), config.Config{}, "build.success", map[string]any{"ok": true}, nil)
}

func TestDispatch_Delivery(t *testing.T) {
	var received struct {
		body  []byte
		event string
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.event = r.Header.Get("X-OSG-Event")
		received.body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.Config{
		Webhooks: []config.WebhookConfig{
			{URL: srv.URL, Events: []string{"build.success"}},
		},
	}

	Dispatch(context.Background(), cfg, "build.success", map[string]any{"total": 10}, slog.Default())

	if received.event != "build.success" {
		t.Errorf("event = %q, want build.success", received.event)
	}

	var payload map[string]any
	if err := json.Unmarshal(received.body, &payload); err != nil {
		t.Fatalf("unmarshal body: %v", err)
	}
	if payload["total"] != float64(10) {
		t.Errorf("payload total = %v, want 10", payload["total"])
	}
}

func TestDispatch_EventFilter(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.Config{
		Webhooks: []config.WebhookConfig{
			{URL: srv.URL, Events: []string{"deploy.success"}},
		},
	}

	// Dispatch with a non-matching event.
	Dispatch(context.Background(), cfg, "build.success", map[string]any{}, nil)

	if called {
		t.Error("webhook should not have been called for non-matching event")
	}
}

func TestDispatch_EmptyEventsMatchesAll(t *testing.T) {
	called := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.Config{
		Webhooks: []config.WebhookConfig{
			{URL: srv.URL}, // no events filter = all events
		},
	}

	Dispatch(context.Background(), cfg, "build.success", map[string]any{}, nil)

	if !called {
		t.Error("webhook with empty events should match all events")
	}
}

func TestDispatch_HMACSignature(t *testing.T) {
	secret := "my-secret-key"
	var receivedSig string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedSig = r.Header.Get("X-OSG-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	cfg := config.Config{
		Webhooks: []config.WebhookConfig{
			{URL: srv.URL, Events: []string{"build.success"}, Secret: secret},
		},
	}

	payload := map[string]any{"pages": 5}
	Dispatch(context.Background(), cfg, "build.success", payload, nil)

	// Compute expected signature.
	body, _ := json.Marshal(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	if receivedSig != expected {
		t.Errorf("signature = %q, want %q", receivedSig, expected)
	}
}

func TestDispatch_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	cfg := config.Config{
		Webhooks: []config.WebhookConfig{
			{URL: srv.URL, Events: []string{"build.failure"}},
		},
	}

	// Should not panic; error is logged.
	Dispatch(context.Background(), cfg, "build.failure", map[string]any{}, slog.Default())
}

func TestMatchesEvent(t *testing.T) {
	tests := []struct {
		subscribed []string
		event      string
		want       bool
	}{
		{nil, "build.success", true},
		{[]string{}, "build.success", true},
		{[]string{"build.success"}, "build.success", true},
		{[]string{"deploy.success"}, "build.success", false},
		{[]string{"build.success", "build.failure"}, "build.failure", true},
	}
	for _, tt := range tests {
		got := matchesEvent(tt.subscribed, tt.event)
		if got != tt.want {
			t.Errorf("matchesEvent(%v, %q) = %v, want %v", tt.subscribed, tt.event, got, tt.want)
		}
	}
}
