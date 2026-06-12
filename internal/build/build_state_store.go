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

// buildStateDBFile is the on-disk filename for the build-state DB.
// Lives next to summaries.db inside cfg.BuildCacheDir (default
// .osg/cache/) so the whole build-cache footprint sits together.
const buildStateDBFile = "build_state.db"

// DeferredReason classifies why a rendered page must be excluded
// from the deploy upload. Stored in the DB as a plain string for
// readability when inspecting the table from the sqlite shell.
const (
	DeferredReasonDraft     = "draft"
	DeferredReasonScheduled = "scheduled"
)

// DeferredEntry is one row in deferred_publications. Path is the
// site-relative URL ("/2026/02/foo/"); Source is the markdown path
// relative to ContentDir; PublishAt is non-zero only for scheduled
// pages so the table self-documents when they will become live.
type DeferredEntry struct {
	Path       string
	Source     string
	Reason     string
	PublishAt  time.Time
	RecordedAt time.Time
}

// BuildStateStore is the SQLite-backed registry of build-time facts
// that survive across builds. Currently a single table — deferred
// publications — but the file is named generically so future build
// metadata can join without another DB file.
type BuildStateStore struct {
	db *sql.DB
}

// buildStateDBPath returns the absolute path of build_state.db,
// honouring cfg.BuildCacheDir like every other cache artefact.
func buildStateDBPath(cfg config.Config) string {
	dir := cfg.BuildCacheDir
	if dir == "" {
		dir = ".osg/cache"
	}
	return filepath.Join(dir, buildStateDBFile)
}

// OpenBuildStateStore opens (or creates) the build state DB and runs
// the schema migration. Callers must Close.
func OpenBuildStateStore(cfg config.Config) (*BuildStateStore, error) {
	path := buildStateDBPath(cfg)
	dir := filepath.Dir(path)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create build state dir: %w", err)
		}
	}
	// modernc/sqlite only applies PRAGMAs given as _pragma=...; the mattn-style
	// _journal_mode/_busy_timeout keys are silently ignored.
	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, fmt.Errorf("open build state db: %w", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS deferred_publications (
			path        TEXT PRIMARY KEY,
			source      TEXT NOT NULL,
			reason      TEXT NOT NULL,
			publish_at  TEXT,
			recorded_at TEXT NOT NULL
		)
	`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create deferred_publications table: %w", err)
	}
	return &BuildStateStore{db: db}, nil
}

// Close releases the underlying sql.DB handle.
func (s *BuildStateStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// LoadDeferred returns every row in deferred_publications, ordered by
// path so callers see deterministic output.
func (s *BuildStateStore) LoadDeferred() ([]DeferredEntry, error) {
	rows, err := s.db.Query(`
		SELECT path, source, reason, publish_at, recorded_at
		FROM deferred_publications
		ORDER BY path
	`)
	if err != nil {
		return nil, fmt.Errorf("query deferred_publications: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []DeferredEntry
	for rows.Next() {
		var (
			path, source, reason  string
			publishAt, recordedAt sql.NullString
		)
		if err := rows.Scan(&path, &source, &reason, &publishAt, &recordedAt); err != nil {
			return nil, fmt.Errorf("scan deferred row: %w", err)
		}
		entry := DeferredEntry{Path: path, Source: source, Reason: reason}
		if publishAt.Valid && publishAt.String != "" {
			if t, err := time.Parse(time.RFC3339, publishAt.String); err == nil {
				entry.PublishAt = t
			}
		}
		if recordedAt.Valid && recordedAt.String != "" {
			if t, err := time.Parse(time.RFC3339, recordedAt.String); err == nil {
				entry.RecordedAt = t
			}
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// ReplaceDeferred atomically wipes the deferred_publications table
// and inserts the given entries. Called once at the end of every
// build so the table always reflects the most recent render — no
// stale rows from past builds. Empty input clears the table, which
// is the right outcome when the user disables IncludeDrafts.
func (s *BuildStateStore) ReplaceDeferred(entries []DeferredEntry) error {
	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM deferred_publications`); err != nil {
		return fmt.Errorf("clear deferred: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	stmt, err := tx.Prepare(`
		INSERT INTO deferred_publications (path, source, reason, publish_at, recorded_at)
		VALUES (?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("prepare insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()
	for _, e := range entries {
		var publishAt sql.NullString
		if !e.PublishAt.IsZero() {
			publishAt = sql.NullString{String: e.PublishAt.UTC().Format(time.RFC3339), Valid: true}
		}
		recordedAt := now
		if !e.RecordedAt.IsZero() {
			recordedAt = e.RecordedAt.UTC().Format(time.RFC3339)
		}
		if _, err := stmt.Exec(e.Path, e.Source, e.Reason, publishAt, recordedAt); err != nil {
			return fmt.Errorf("insert deferred row %q: %w", e.Path, err)
		}
	}
	return tx.Commit()
}

// LoadDeferredPaths is a convenience wrapper used by the deploy
// pipeline: opens the store, returns just the URL paths, closes.
// Errors are logged and an empty slice returned so a missing/corrupt
// DB never blocks a deploy — at worst the user gets a deploy that
// uploads everything (the previous behaviour).
func LoadDeferredPaths(cfg config.Config, logger *slog.Logger) []string {
	store, err := OpenBuildStateStore(cfg)
	if err != nil {
		if logger != nil {
			logger.Warn("open build state store failed", "error", err)
		}
		return nil
	}
	defer func() { _ = store.Close() }()
	rows, err := store.LoadDeferred()
	if err != nil {
		if logger != nil {
			logger.Warn("load deferred publications failed", "error", err)
		}
		return nil
	}
	out := make([]string, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.Path)
	}
	return out
}
