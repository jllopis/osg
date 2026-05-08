// Package scheduler holds the persistence layer for the osg ui
// scheduler service: an audit log of every rebuild it triggered, when
// the publish_at it was waiting for fired, and whether the rebuild
// succeeded.
package scheduler

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Run records a single scheduler trigger.
type Run struct {
	ID     int64
	DueAt  time.Time
	RanAt  time.Time
	Status string // "ok" or "error"
	Error  string
}

// Store persists scheduler runs to a SQLite database.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) the SQLite database at dbPath and ensures
// the schema is up to date. The parent directory is created if needed.
func NewStore(dbPath string) (*Store, error) {
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}
	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}
	return &Store{db: db}, nil
}

func migrate(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS scheduler_runs (
	id      INTEGER PRIMARY KEY AUTOINCREMENT,
	due_at  TEXT    NOT NULL,
	ran_at  TEXT    NOT NULL,
	status  TEXT    NOT NULL CHECK(status IN ('ok','error')),
	error   TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_scheduler_runs_ran_at ON scheduler_runs(ran_at DESC);
`
	_, err := db.Exec(schema)
	return err
}

// Close closes the underlying database handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Record inserts a new run entry.
func (s *Store) Record(r Run) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.Exec(
		`INSERT INTO scheduler_runs (due_at, ran_at, status, error) VALUES (?, ?, ?, ?)`,
		r.DueAt.UTC().Format(time.RFC3339Nano),
		r.RanAt.UTC().Format(time.RFC3339Nano),
		r.Status,
		r.Error,
	)
	return err
}

// Recent returns up to limit runs in reverse chronological order.
func (s *Store) Recent(limit int) ([]Run, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT id, due_at, ran_at, status, error FROM scheduler_runs ORDER BY ran_at DESC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []Run
	for rows.Next() {
		var (
			r            Run
			dueAt, ranAt string
		)
		if err := rows.Scan(&r.ID, &dueAt, &ranAt, &r.Status, &r.Error); err != nil {
			return nil, err
		}
		r.DueAt, _ = time.Parse(time.RFC3339Nano, dueAt)
		r.RanAt, _ = time.Parse(time.RFC3339Nano, ranAt)
		out = append(out, r)
	}
	return out, rows.Err()
}
