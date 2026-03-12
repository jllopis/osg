package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

func newTestAnalyticsStore(t *testing.T) *AnalyticsStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath, 24)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	as, err := NewAnalyticsStore(store.DB())
	if err != nil {
		t.Fatal(err)
	}
	return as
}

func TestAnalyticsStore_RecordAndSummary(t *testing.T) {
	as := newTestAnalyticsStore(t)

	// Record some events.
	for _, evt := range []AnalyticsEvent{
		{Path: "/blog/post-1", Referrer: "https://google.com/search?q=test", UA: "Mozilla/5.0 Chrome/120"},
		{Path: "/blog/post-1", Referrer: "https://twitter.com/status/123", UA: "Mozilla/5.0 Safari/605"},
		{Path: "/about", Referrer: "", UA: "Mozilla/5.0 Firefox/121"},
	} {
		if err := as.Record(evt); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	summary, err := as.Summary(30)
	if err != nil {
		t.Fatalf("Summary: %v", err)
	}

	if summary.TotalViews != 3 {
		t.Errorf("TotalViews = %d, want 3", summary.TotalViews)
	}
	if len(summary.TopPages) < 1 {
		t.Errorf("TopPages empty")
	}
	if len(summary.Browsers) < 1 {
		t.Errorf("Browsers empty")
	}
}

func TestAnalyticsHandlers_Hit(t *testing.T) {
	as := newTestAnalyticsStore(t)
	h := NewAnalyticsHandlers(as, nil)

	body, _ := json.Marshal(AnalyticsEvent{Path: "/test"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/hit", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "TestBot/1.0")
	w := httptest.NewRecorder()
	h.HandleHit(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

func TestAnalyticsHandlers_DNT(t *testing.T) {
	as := newTestAnalyticsStore(t)
	h := NewAnalyticsHandlers(as, nil)

	body, _ := json.Marshal(AnalyticsEvent{Path: "/test"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/hit", bytes.NewReader(body))
	req.Header.Set("DNT", "1")
	w := httptest.NewRecorder()
	h.HandleHit(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	// Verify nothing was recorded.
	summary, _ := as.Summary(30)
	if summary.TotalViews != 0 {
		t.Errorf("views = %d after DNT, want 0", summary.TotalViews)
	}
}

func TestAnalyticsHandlers_Summary(t *testing.T) {
	as := newTestAnalyticsStore(t)
	_ = as.Record(AnalyticsEvent{Path: "/page", UA: "Chrome"})
	h := NewAnalyticsHandlers(as, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/analytics/summary", nil)
	w := httptest.NewRecorder()
	h.HandleSummary(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}

	var summary AnalyticsSummary
	if err := json.NewDecoder(w.Body).Decode(&summary); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if summary.TotalViews != 1 {
		t.Errorf("TotalViews = %d, want 1", summary.TotalViews)
	}
}

func TestClassifyBrowser(t *testing.T) {
	tests := []struct {
		ua, want string
	}{
		{"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120.0", "Chrome"},
		{"Mozilla/5.0 (Macintosh) AppleWebKit/605.1.15 Safari/605.1", "Safari"},
		{"Mozilla/5.0 Firefox/121.0", "Firefox"},
		{"Mozilla/5.0 Edg/120.0", "Edge"},
		{"Googlebot/2.1", "Bot"},
		{"curl/7.88.1", "Other"},
	}
	for _, tt := range tests {
		got := classifyBrowser(tt.ua)
		if got != tt.want {
			t.Errorf("classifyBrowser(%q) = %q, want %q", tt.ua, got, tt.want)
		}
	}
}

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"https://google.com/search?q=test", "google.com"},
		{"http://example.org/page", "example.org"},
		{"", ""},
		{"example.com/path", "example.com"},
	}
	for _, tt := range tests {
		got := extractDomain(tt.input)
		if got != tt.want {
			t.Errorf("extractDomain(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
