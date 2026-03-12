package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// AnalyticsStore manages page analytics data in SQLite.
type AnalyticsStore struct {
	db *sql.DB
}

// AnalyticsEvent represents a single analytics hit.
type AnalyticsEvent struct {
	Path     string `json:"path"`
	Referrer string `json:"referrer"`
	UA       string `json:"ua"`
}

// AnalyticsSummary is the response for the analytics dashboard endpoint.
type AnalyticsSummary struct {
	TotalViews int64               `json:"total_views"`
	TopPages   []PageViewCount     `json:"top_pages"`
	TopRefs    []ReferrerCount     `json:"top_referrers"`
	Daily      []DailyViewCount    `json:"daily"`
	Browsers   []BrowserGroupCount `json:"browsers"`
}

// PageViewCount is a page path with its view count.
type PageViewCount struct {
	Path  string `json:"path"`
	Views int64  `json:"views"`
}

// ReferrerCount is a referrer domain with its count.
type ReferrerCount struct {
	Referrer string `json:"referrer"`
	Count    int64  `json:"count"`
}

// DailyViewCount is a day with its view count.
type DailyViewCount struct {
	Date  string `json:"date"`
	Views int64  `json:"views"`
}

// BrowserGroupCount is a browser family with its count.
type BrowserGroupCount struct {
	Browser string `json:"browser"`
	Count   int64  `json:"count"`
}

// NewAnalyticsStore creates an analytics store reusing the given DB connection.
func NewAnalyticsStore(db *sql.DB) (*AnalyticsStore, error) {
	if err := migrateAnalytics(db); err != nil {
		return nil, err
	}
	return &AnalyticsStore{db: db}, nil
}

func migrateAnalytics(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS analytics (
			id        INTEGER PRIMARY KEY AUTOINCREMENT,
			path      TEXT    NOT NULL,
			referrer  TEXT    NOT NULL DEFAULT '',
			browser   TEXT    NOT NULL DEFAULT '',
			ts        TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		);
		CREATE INDEX IF NOT EXISTS idx_analytics_path ON analytics(path);
		CREATE INDEX IF NOT EXISTS idx_analytics_ts   ON analytics(ts);
	`)
	return err
}

// Record stores a single analytics event.
func (s *AnalyticsStore) Record(evt AnalyticsEvent) error {
	browser := classifyBrowser(evt.UA)
	referrer := extractDomain(evt.Referrer)
	_, err := s.db.Exec(
		`INSERT INTO analytics (path, referrer, browser) VALUES (?, ?, ?)`,
		evt.Path, referrer, browser,
	)
	return err
}

// Summary returns aggregated analytics for the last N days.
func (s *AnalyticsStore) Summary(days int) (*AnalyticsSummary, error) {
	if days <= 0 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	var total int64
	if err := s.db.QueryRow(
		`SELECT COUNT(*) FROM analytics WHERE ts >= ?`, since,
	).Scan(&total); err != nil {
		return nil, err
	}

	topPages, err := s.queryPageViews(since, 20)
	if err != nil {
		return nil, err
	}

	topRefs, err := s.queryReferrers(since, 10)
	if err != nil {
		return nil, err
	}

	daily, err := s.queryDaily(since)
	if err != nil {
		return nil, err
	}

	browsers, err := s.queryBrowsers(since)
	if err != nil {
		return nil, err
	}

	return &AnalyticsSummary{
		TotalViews: total,
		TopPages:   topPages,
		TopRefs:    topRefs,
		Daily:      daily,
		Browsers:   browsers,
	}, nil
}

func (s *AnalyticsStore) queryPageViews(since string, limit int) ([]PageViewCount, error) {
	rows, err := s.db.Query(
		`SELECT path, COUNT(*) as cnt FROM analytics WHERE ts >= ? GROUP BY path ORDER BY cnt DESC LIMIT ?`,
		since, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []PageViewCount
	for rows.Next() {
		var pv PageViewCount
		if err := rows.Scan(&pv.Path, &pv.Views); err != nil {
			return nil, err
		}
		out = append(out, pv)
	}
	return out, rows.Err()
}

func (s *AnalyticsStore) queryReferrers(since string, limit int) ([]ReferrerCount, error) {
	rows, err := s.db.Query(
		`SELECT referrer, COUNT(*) as cnt FROM analytics WHERE ts >= ? AND referrer != '' GROUP BY referrer ORDER BY cnt DESC LIMIT ?`,
		since, limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []ReferrerCount
	for rows.Next() {
		var rc ReferrerCount
		if err := rows.Scan(&rc.Referrer, &rc.Count); err != nil {
			return nil, err
		}
		out = append(out, rc)
	}
	return out, rows.Err()
}

func (s *AnalyticsStore) queryDaily(since string) ([]DailyViewCount, error) {
	rows, err := s.db.Query(
		`SELECT date(ts) as d, COUNT(*) as cnt FROM analytics WHERE ts >= ? GROUP BY d ORDER BY d`,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []DailyViewCount
	for rows.Next() {
		var dv DailyViewCount
		if err := rows.Scan(&dv.Date, &dv.Views); err != nil {
			return nil, err
		}
		out = append(out, dv)
	}
	return out, rows.Err()
}

func (s *AnalyticsStore) queryBrowsers(since string) ([]BrowserGroupCount, error) {
	rows, err := s.db.Query(
		`SELECT browser, COUNT(*) as cnt FROM analytics WHERE ts >= ? AND browser != '' GROUP BY browser ORDER BY cnt DESC`,
		since,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	var out []BrowserGroupCount
	for rows.Next() {
		var bc BrowserGroupCount
		if err := rows.Scan(&bc.Browser, &bc.Count); err != nil {
			return nil, err
		}
		out = append(out, bc)
	}
	return out, rows.Err()
}

// classifyBrowser extracts a simplified browser name from User-Agent.
func classifyBrowser(ua string) string {
	ua = strings.ToLower(ua)
	switch {
	case strings.Contains(ua, "firefox"):
		return "Firefox"
	case strings.Contains(ua, "edg"):
		return "Edge"
	case strings.Contains(ua, "chrome") || strings.Contains(ua, "chromium"):
		return "Chrome"
	case strings.Contains(ua, "safari"):
		return "Safari"
	case strings.Contains(ua, "bot") || strings.Contains(ua, "crawl") || strings.Contains(ua, "spider"):
		return "Bot"
	default:
		return "Other"
	}
}

// extractDomain extracts the host from a referrer URL.
func extractDomain(referrer string) string {
	if referrer == "" {
		return ""
	}
	// Strip protocol.
	r := referrer
	if idx := strings.Index(r, "://"); idx >= 0 {
		r = r[idx+3:]
	}
	// Strip path.
	if idx := strings.IndexByte(r, '/'); idx >= 0 {
		r = r[:idx]
	}
	return r
}

// AnalyticsHandlers holds HTTP handlers for analytics.
type AnalyticsHandlers struct {
	store  *AnalyticsStore
	logger interface{ Warn(string, ...any) }
}

// NewAnalyticsHandlers creates analytics HTTP handlers.
func NewAnalyticsHandlers(store *AnalyticsStore, logger interface{ Warn(string, ...any) }) *AnalyticsHandlers {
	return &AnalyticsHandlers{store: store, logger: logger}
}

// HandleHit records a single analytics event.
// Respects DNT (Do Not Track) header.
func (h *AnalyticsHandlers) HandleHit(w http.ResponseWriter, r *http.Request) {
	// Respect Do Not Track.
	if r.Header.Get("DNT") == "1" {
		writeJSON(w, http.StatusOK, map[string]string{"status": "dnt"})
		return
	}

	var evt AnalyticsEvent
	if err := json.NewDecoder(r.Body).Decode(&evt); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if evt.Path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "path required"})
		return
	}
	evt.UA = r.UserAgent()

	if err := h.store.Record(evt); err != nil {
		if h.logger != nil {
			h.logger.Warn("analytics record failed", "error", err)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleSummary returns aggregated analytics data.
func (h *AnalyticsHandlers) HandleSummary(w http.ResponseWriter, _ *http.Request) {
	summary, err := h.store.Summary(30)
	if err != nil {
		if h.logger != nil {
			h.logger.Warn("analytics summary failed", "error", err)
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	writeJSON(w, http.StatusOK, summary)
}
