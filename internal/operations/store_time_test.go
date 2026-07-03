package operations

import (
	"path/filepath"
	"testing"
	"time"
)

// Trailing-zero timestamps used to be stored trimmed (RFC3339Nano), which
// made "...00Z" sort after "...00.5Z" in SQLite's string comparison. The
// fixed-width storedTimeFormat must keep ordering chronological.
func TestRecent_OrderIsChronological(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "operations.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	base := time.Date(2026, 7, 3, 10, 0, 0, 0, time.UTC)
	older := base.Add(500 * time.Millisecond) // fractional part
	newer := base.Add(time.Second)            // whole second (trailing zeros)

	if _, err := store.Begin("older", KindTask, nil, older); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Begin("newer", KindTask, nil, newer); err != nil {
		t.Fatal(err)
	}

	runs, err := store.Recent(Filter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 2 {
		t.Fatalf("got %d runs, want 2", len(runs))
	}
	if runs[0].Name != "newer" || runs[1].Name != "older" {
		t.Errorf("order = [%s, %s], want [newer, older]", runs[0].Name, runs[1].Name)
	}

	// The Since filter compares strings too; a cutoff between the two runs
	// must return only the newer one.
	runs, err = store.Recent(Filter{Since: base.Add(700 * time.Millisecond)})
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 1 || runs[0].Name != "newer" {
		t.Errorf("Since filter returned %+v, want only 'newer'", runs)
	}
}

// Rows written by earlier versions with time.RFC3339Nano get rewritten to
// the fixed-width format when the store is reopened.
func TestNormalizeStoredTimes_MigratesLegacyRows(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "operations.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}

	// Simulate a legacy row: trimmed RFC3339Nano (no fractional seconds).
	legacyStart := "2026-07-03T10:00:01Z"
	legacyEnd := "2026-07-03T10:00:02.5Z"
	if _, err := store.db.Exec(
		`INSERT INTO operations_runs (name, kind, params, started_at, ended_at, status)
		 VALUES ('legacy', 'task', '', ?, ?, 'ok')`, legacyStart, legacyEnd,
	); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	// Reopen: migrate() runs normalizeStoredTimes.
	store, err = NewStore(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	var started, ended string
	if err := store.db.QueryRow(
		`SELECT started_at, ended_at FROM operations_runs WHERE name = 'legacy'`,
	).Scan(&started, &ended); err != nil {
		t.Fatal(err)
	}
	if started != "2026-07-03T10:00:01.000000000Z" {
		t.Errorf("started_at = %q, want fixed-width form", started)
	}
	if ended != "2026-07-03T10:00:02.500000000Z" {
		t.Errorf("ended_at = %q, want fixed-width form", ended)
	}
}
