// Package operations provides a unified runner and audit log for both
// long-running services (serve, api, watcher, scheduler) and one-shot
// tasks (build, deploy, check, audit, ...) launched from the osg ui
// dashboard. Every Trigger creates a row in operations_runs that can be
// inspected later via the Recent / Filter queries.
package operations

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// Status values stored in the DB.
const (
	StatusRunning   = "running"
	StatusOK        = "ok"
	StatusError     = "error"
	StatusCancelled = "cancelled"
)

// Kind values stored in the DB.
const (
	KindService = "service"
	KindTask    = "task"
)

// HistoryRun is a row in operations_runs.
type HistoryRun struct {
	ID        int64
	Name      string
	Kind      string
	Params    map[string]any
	StartedAt time.Time
	EndedAt   time.Time // zero when still running
	Status    string
	Error     string
}

// Filter narrows down history queries.
type Filter struct {
	Name   string // exact match if non-empty
	Kind   string // "service" / "task" / "" (all)
	Status string // "running" / "ok" / "error" / "cancelled" / "" (all)
	Since  time.Time
	Limit  int
}

// Store is the SQLite-backed operations audit log.
type Store struct {
	db *sql.DB
}

// NewStore opens (or creates) operations.db at the given path. If a
// sibling scheduler.db exists from a prior osg version, its rows are
// migrated into operations_runs and the legacy file is renamed to
// scheduler.db.bak so the migration only runs once.
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
	store := &Store{db: db}
	// Best-effort migration: if it fails the operations DB is still
	// usable; the legacy file is left in place for retry next time.
	_ = store.migrateLegacyScheduler(filepath.Join(dir, "scheduler.db"))
	return store, nil
}

func migrate(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS operations_runs (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	name        TEXT    NOT NULL,
	kind        TEXT    NOT NULL CHECK(kind IN ('service','task')),
	params      TEXT    NOT NULL DEFAULT '',
	started_at  TEXT    NOT NULL,
	ended_at    TEXT,
	status      TEXT    NOT NULL DEFAULT 'running'
		CHECK(status IN ('running','ok','error','cancelled')),
	error       TEXT    NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_operations_runs_name
	ON operations_runs(name);
CREATE INDEX IF NOT EXISTS idx_operations_runs_started
	ON operations_runs(started_at DESC);
`
	_, err := db.Exec(schema)
	return err
}

// migrateLegacyScheduler imports rows from a pre-Phase-30H scheduler.db
// (single table scheduler_runs with due_at/ran_at/status/error). On
// success the file is renamed to scheduler.db.bak so we do not migrate
// twice. Missing file is not an error.
func (s *Store) migrateLegacyScheduler(legacyPath string) error {
	if _, err := os.Stat(legacyPath); err != nil {
		return nil
	}
	legacy, err := sql.Open("sqlite", legacyPath+"?mode=ro&_journal_mode=WAL")
	if err != nil {
		return err
	}
	defer func() { _ = legacy.Close() }()

	rows, err := legacy.Query(`SELECT due_at, ran_at, status, error FROM scheduler_runs ORDER BY id ASC`)
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT INTO operations_runs
		(name, kind, params, started_at, ended_at, status, error)
		VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	defer func() { _ = stmt.Close() }()

	count := 0
	for rows.Next() {
		var (
			dueAt, ranAt, status, errMsg string
		)
		if err := rows.Scan(&dueAt, &ranAt, &status, &errMsg); err != nil {
			_ = tx.Rollback()
			return err
		}
		params := fmt.Sprintf(`{"due_at":%q,"migrated":true}`, dueAt)
		if _, err := stmt.Exec("scheduler:trigger", KindTask, params, ranAt, ranAt, status, errMsg); err != nil {
			_ = tx.Rollback()
			return err
		}
		count++
	}
	if err := rows.Err(); err != nil {
		_ = tx.Rollback()
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if count > 0 {
		_ = legacy.Close()
		_ = os.Rename(legacyPath, legacyPath+".bak")
	}
	return nil
}

// Close closes the underlying handle.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Begin records a new run with status=running and returns the assigned id.
func (s *Store) Begin(name, kind string, params map[string]any, startedAt time.Time) (int64, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	paramsJSON := encodeParams(params)
	res, err := s.db.Exec(
		`INSERT INTO operations_runs (name, kind, params, started_at, status)
		 VALUES (?, ?, ?, ?, ?)`,
		name, kind, paramsJSON, startedAt.UTC().Format(time.RFC3339Nano), StatusRunning,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Finish marks a run as completed with the given status and error.
func (s *Store) Finish(id int64, status, errMsg string, endedAt time.Time) error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.Exec(
		`UPDATE operations_runs SET ended_at = ?, status = ?, error = ? WHERE id = ?`,
		endedAt.UTC().Format(time.RFC3339Nano), status, errMsg, id,
	)
	return err
}

// MarkInterruptedRunning marks every run still in status='running' as
// cancelled with error="shutdown" and ended_at=now. Called on dashboard
// shutdown so persisted history matches reality.
func (s *Store) MarkInterruptedRunning(now time.Time) (int64, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	res, err := s.db.Exec(
		`UPDATE operations_runs SET ended_at = ?, status = ?, error = ?
		 WHERE status = 'running'`,
		now.UTC().Format(time.RFC3339Nano), StatusCancelled, "shutdown",
	)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Recent returns up to limit runs in reverse chronological order,
// optionally filtered.
func (s *Store) Recent(filter Filter) ([]HistoryRun, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}

	var (
		clauses []string
		args    []any
	)
	if filter.Name != "" {
		clauses = append(clauses, "name = ?")
		args = append(args, filter.Name)
	}
	if filter.Kind != "" {
		clauses = append(clauses, "kind = ?")
		args = append(args, filter.Kind)
	}
	if filter.Status != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, filter.Status)
	}
	if !filter.Since.IsZero() {
		clauses = append(clauses, "started_at >= ?")
		args = append(args, filter.Since.UTC().Format(time.RFC3339Nano))
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	args = append(args, limit)

	q := fmt.Sprintf(
		`SELECT id, name, kind, params, started_at, ended_at, status, error
		 FROM operations_runs %s
		 ORDER BY started_at DESC LIMIT ?`, where,
	)
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []HistoryRun
	for rows.Next() {
		var (
			r                  HistoryRun
			startedAt          string
			endedAt, paramsRaw sql.NullString
		)
		if err := rows.Scan(&r.ID, &r.Name, &r.Kind, &paramsRaw, &startedAt, &endedAt, &r.Status, &r.Error); err != nil {
			return nil, err
		}
		r.StartedAt, _ = time.Parse(time.RFC3339Nano, startedAt)
		if endedAt.Valid {
			r.EndedAt, _ = time.Parse(time.RFC3339Nano, endedAt.String)
		}
		if paramsRaw.Valid && paramsRaw.String != "" {
			r.Params = decodeParams(paramsRaw.String)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func encodeParams(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	b, err := json.Marshal(params)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeParams(s string) map[string]any {
	var out map[string]any
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil
	}
	return out
}
