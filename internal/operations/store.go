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
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

// sqlitePragmas is the modernc/sqlite DSN query that enables WAL journaling
// and a busy timeout. The modernc driver only recognises the `_pragma=`
// form (it silently ignores the mattn-style `_journal_mode`/`_busy_timeout`
// keys), so each PRAGMA must be expressed this way to actually take effect.
const sqlitePragmas = "_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"

// storedTimeFormat is RFC 3339 UTC with the fractional seconds padded to a
// fixed nine digits. time.RFC3339Nano trims trailing zeros, which breaks the
// lexicographic ordering SQLite applies to TEXT timestamps ("...00Z" sorts
// after "...00.5Z"); fixed-width strings sort chronologically. Reads keep
// parsing with time.RFC3339Nano, which accepts both forms.
const storedTimeFormat = "2006-01-02T15:04:05.000000000Z07:00"

// storedTimeLen is the length of a UTC timestamp in storedTimeFormat; rows
// with a different length predate the fixed-width format and get rewritten
// by normalizeStoredTimes.
const storedTimeLen = 30

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
	db, err := sql.Open("sqlite", dbPath+"?"+sqlitePragmas)
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
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	return normalizeStoredTimes(db)
}

// normalizeStoredTimes rewrites timestamps written by earlier versions with
// time.RFC3339Nano (variable width) into storedTimeFormat so that string
// comparisons in ORDER BY / >= filters are chronological. Unparseable values
// are left untouched.
func normalizeStoredTimes(db *sql.DB) error {
	rows, err := db.Query(
		`SELECT id, started_at, ended_at FROM operations_runs
		 WHERE length(started_at) <> ?
		    OR (ended_at IS NOT NULL AND length(ended_at) <> ?)`,
		storedTimeLen, storedTimeLen,
	)
	if err != nil {
		return err
	}
	type fix struct {
		id             int64
		started, ended string
	}
	var fixes []fix
	for rows.Next() {
		var (
			id      int64
			started string
			ended   sql.NullString
		)
		if err := rows.Scan(&id, &started, &ended); err != nil {
			_ = rows.Close()
			return err
		}
		f := fix{id: id, started: started, ended: ended.String}
		if t, err := time.Parse(time.RFC3339Nano, started); err == nil {
			f.started = t.UTC().Format(storedTimeFormat)
		}
		if ended.Valid {
			if t, err := time.Parse(time.RFC3339Nano, ended.String); err == nil {
				f.ended = t.UTC().Format(storedTimeFormat)
			}
		}
		fixes = append(fixes, f)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return err
	}
	_ = rows.Close()
	if len(fixes) == 0 {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	for _, f := range fixes {
		var endedVal any
		if f.ended != "" {
			endedVal = f.ended
		}
		if _, err := tx.Exec(
			`UPDATE operations_runs SET started_at = ?, ended_at = ? WHERE id = ?`,
			f.started, endedVal, f.id,
		); err != nil {
			_ = tx.Rollback()
			return err
		}
	}
	return tx.Commit()
}

// migrateLegacyScheduler imports rows from a pre-Phase-30H scheduler.db
// (single table scheduler_runs with due_at/ran_at/status/error). On
// success the file is renamed to scheduler.db.bak so we do not migrate
// twice. Missing file is not an error.
func (s *Store) migrateLegacyScheduler(legacyPath string) error {
	if _, err := os.Stat(legacyPath); err != nil {
		return nil
	}
	// Open read-only so the legacy file is untouched before we rename it to
	// .bak. modernc only honours mode=ro when the DSN is a file: URI (a plain
	// path has its query stripped), so build one with proper escaping.
	abs, absErr := filepath.Abs(legacyPath)
	if absErr != nil {
		abs = legacyPath
	}
	roDSN := (&url.URL{Scheme: "file", Path: abs, RawQuery: "mode=ro"}).String()
	legacy, err := sql.Open("sqlite", roDSN)
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
		name, kind, paramsJSON, startedAt.UTC().Format(storedTimeFormat), StatusRunning,
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
		endedAt.UTC().Format(storedTimeFormat), status, errMsg, id,
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
		now.UTC().Format(storedTimeFormat), StatusCancelled, "shutdown",
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
		args = append(args, filter.Since.UTC().Format(storedTimeFormat))
	}
	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	args = append(args, limit)

	q := fmt.Sprintf(
		`SELECT id, name, kind, params, started_at, ended_at, status, error
		 FROM operations_runs %s
		 ORDER BY started_at DESC, id DESC LIMIT ?`, where,
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
