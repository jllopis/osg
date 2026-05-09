package build

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"osg/internal/config"
)

// summariesDBFile is the on-disk filename for the AI summary store.
// It lives alongside the rest of the build cache (.osg/cache/) so a
// single sweep of that directory invalidates everything regenerable.
const summariesDBFile = "summaries.db"

// summariesDBPath returns the absolute path to the summaries database.
// Mirrors aiCachePath's logic so the file sits next to other cache
// artefacts (build.json) instead of polluting the top-level .osg dir.
func summariesDBPath(cfg config.Config) string {
	dir := cfg.BuildCacheDir
	if dir == "" {
		dir = ".osg/cache"
	}
	return filepath.Join(dir, summariesDBFile)
}

// summaryStore wraps the sqlite handle that backs the AI summary
// cache. Exposed only inside internal/build; the public surface stays
// the same as before (LookupAISummary / InvalidateAISummary helpers).
type summaryStore struct {
	db *sql.DB
}

// openSummaryStore opens (or creates) the summaries.db file with WAL
// journaling enabled and ensures the schema exists. Callers must
// Close() the returned store.
func openSummaryStore(cfg config.Config) (*summaryStore, error) {
	path := summariesDBPath(cfg)
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create summaries dir: %w", err)
		}
	}
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open summaries db: %w", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS summaries (
			hash       TEXT PRIMARY KEY,
			summary    TEXT NOT NULL,
			provider   TEXT,
			model      TEXT,
			created_at TEXT
		)
	`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create summaries table: %w", err)
	}
	return &summaryStore{db: db}, nil
}

// Close releases the underlying sql.DB handle.
func (s *summaryStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// LoadAll reads every cached entry into a map keyed by content hash.
// Used by the build at startup so the per-page lookup loop stays
// in-memory (same shape as the old JSON-backed cache). Errors are
// returned to the caller — callers may choose to log and continue.
func (s *summaryStore) LoadAll() (map[string]AICacheEntry, error) {
	rows, err := s.db.Query(`SELECT hash, summary, provider, model, created_at FROM summaries`)
	if err != nil {
		return nil, fmt.Errorf("query summaries: %w", err)
	}
	defer func() { _ = rows.Close() }()

	out := make(map[string]AICacheEntry)
	for rows.Next() {
		var (
			hash, summary   string
			provider, model sql.NullString
			createdAt       sql.NullString
		)
		if err := rows.Scan(&hash, &summary, &provider, &model, &createdAt); err != nil {
			return nil, fmt.Errorf("scan summary row: %w", err)
		}
		out[hash] = AICacheEntry{
			Summary:     summary,
			Provider:    provider.String,
			Model:       model.String,
			GeneratedAt: createdAt.String,
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// Upsert inserts or replaces the entry for the given content hash.
// `created_at` is overwritten on every call so callers can tell when
// a summary was last regenerated.
func (s *summaryStore) Upsert(hash string, entry AICacheEntry) error {
	createdAt := entry.GeneratedAt
	if createdAt == "" {
		createdAt = time.Now().UTC().Format(time.RFC3339)
	}
	_, err := s.db.Exec(`
		INSERT INTO summaries (hash, summary, provider, model, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(hash) DO UPDATE SET
			summary    = excluded.summary,
			provider   = excluded.provider,
			model      = excluded.model,
			created_at = excluded.created_at
	`, hash, entry.Summary, nullable(entry.Provider), nullable(entry.Model), createdAt)
	if err != nil {
		return fmt.Errorf("upsert summary: %w", err)
	}
	return nil
}

// Lookup returns the entry for the given hash, or false when missing.
func (s *summaryStore) Lookup(hash string) (AICacheEntry, bool, error) {
	row := s.db.QueryRow(`SELECT summary, provider, model, created_at FROM summaries WHERE hash = ?`, hash)
	var (
		summary         string
		provider, model sql.NullString
		createdAt       sql.NullString
	)
	if err := row.Scan(&summary, &provider, &model, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return AICacheEntry{}, false, nil
		}
		return AICacheEntry{}, false, err
	}
	return AICacheEntry{
		Summary:     summary,
		Provider:    provider.String,
		Model:       model.String,
		GeneratedAt: createdAt.String,
	}, true, nil
}

// Remove drops the entry for the given hash. Returns true when a row
// was actually deleted (matching AICache.Remove's semantics).
func (s *summaryStore) Remove(hash string) (bool, error) {
	res, err := s.db.Exec(`DELETE FROM summaries WHERE hash = ?`, hash)
	if err != nil {
		return false, fmt.Errorf("delete summary: %w", err)
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

// nullable converts an empty string to sql.NullString{Valid: false}
// so optional metadata columns store NULL instead of "".
func nullable(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

// loadAISummariesIntoCache opens the summary store, copies every row
// into the in-memory AICache, and closes the store. Errors are logged
// to keep the build path resilient: a corrupt or missing DB just
// means we'll regenerate everything from scratch.
func loadAISummariesIntoCache(cfg config.Config, logger *slog.Logger) *AICache {
	cache := newAICache("", "")
	store, err := openSummaryStore(cfg)
	if err != nil {
		if logger != nil {
			logger.Warn("failed to open summary store", "error", err)
		}
		return cache
	}
	defer func() { _ = store.Close() }()
	entries, err := store.LoadAll()
	if err != nil {
		if logger != nil {
			logger.Warn("failed to load summaries from db", "error", err)
		}
		return cache
	}
	cache.Entries = entries
	if logger != nil {
		logger.Info("AI summary cache loaded", "entries", len(entries))
	}
	return cache
}

// persistAICache flushes the in-memory cache to SQLite. The whole map
// is upserted because the build mutates entries through the AICache
// shim and we don't track which keys changed; the row count is small
// (one per page that uses an AI summary) so the cost is negligible.
func persistAICache(cfg config.Config, cache *AICache, logger *slog.Logger) error {
	if cache == nil {
		return nil
	}
	store, err := openSummaryStore(cfg)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	cache.mu.RLock()
	defer cache.mu.RUnlock()
	for h, e := range cache.Entries {
		if err := store.Upsert(h, e); err != nil {
			return err
		}
	}
	if logger != nil {
		logger.Info("AI summary cache saved", "entries", len(cache.Entries))
	}
	return nil
}
