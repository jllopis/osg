package operations

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestTriggerStop(t *testing.T) {
	stopped := make(chan struct{})
	r := New([]Definition{{
		Name: "echo",
		Kind: KindService,
		Run: func(ctx context.Context, _ map[string]any, w io.Writer) error {
			_, _ = w.Write([]byte("starting\n"))
			<-ctx.Done()
			close(stopped)
			return nil
		},
	}}, nil)

	run, err := r.Trigger("echo", nil)
	if err != nil {
		t.Fatalf("Trigger: %v", err)
	}
	if run.State != StateRunning {
		t.Errorf("State=%s want running", run.State)
	}

	// Second Trigger fails (concurrency by name).
	if _, err := r.Trigger("echo", nil); err == nil {
		t.Fatalf("expected concurrency error")
	}

	if err := r.Stop("echo"); err != nil {
		t.Fatalf("Stop: %v", err)
	}
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatalf("runner did not observe ctx cancellation")
	}

	if _, busy := r.Active("echo"); busy {
		t.Fatalf("active map not cleared")
	}
}

func TestTriggerImmediateError(t *testing.T) {
	r := New([]Definition{{
		Name: "broken",
		Run: func(ctx context.Context, _ map[string]any, _ io.Writer) error {
			return errors.New("port in use")
		},
	}}, nil)
	if _, err := r.Trigger("broken", nil); err == nil {
		t.Fatalf("expected immediate error")
	}
	if _, busy := r.Active("broken"); busy {
		t.Fatalf("active map should not retain failed run")
	}
}

func TestTriggerUnknown(t *testing.T) {
	r := New(nil, nil)
	if _, err := r.Trigger("missing", nil); err == nil {
		t.Fatalf("expected unknown error")
	}
}

func TestStopAllPersistsCancellation(t *testing.T) {
	store := openTestStore(t)
	var wg sync.WaitGroup
	wg.Add(2)
	runner := New([]Definition{
		{Name: "a", Kind: KindService, Run: blockingRun(&wg)},
		{Name: "b", Kind: KindService, Run: blockingRun(&wg)},
	}, store)
	if _, err := runner.Trigger("a", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := runner.Trigger("b", nil); err != nil {
		t.Fatal(err)
	}
	runner.StopAll()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("StopAll did not cancel runners")
	}
	rows, err := store.Recent(Filter{Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Status == StatusRunning {
			t.Errorf("row still in running state after StopAll: %+v", r)
		}
	}
}

func TestSnapshotIncludesActiveAndLast(t *testing.T) {
	store := openTestStore(t)
	r := New([]Definition{{
		Name: "build",
		Kind: KindTask,
		Run: func(ctx context.Context, _ map[string]any, _ io.Writer) error {
			return nil
		},
	}}, store)

	// One completed run.
	run1, err := r.Trigger("build", nil)
	if err != nil {
		t.Fatal(err)
	}
	<-run1.done

	snap := r.Snapshot()
	if len(snap) != 1 {
		t.Fatalf("snapshot len=%d", len(snap))
	}
	if snap[0].State != StateIdle {
		t.Errorf("State=%s after run completed", snap[0].State)
	}
	if snap[0].LastRun == nil {
		t.Errorf("LastRun nil after a completed run")
	}
}

func TestMigrateLegacySchedulerDB(t *testing.T) {
	dir := t.TempDir()
	legacyPath := filepath.Join(dir, "scheduler.db")
	// Create a legacy scheduler.db with one row.
	legacy, err := openLegacy(t, legacyPath)
	if err != nil {
		t.Fatalf("create legacy: %v", err)
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, _ = legacy.Exec(`INSERT INTO scheduler_runs (due_at, ran_at, status, error) VALUES (?, ?, ?, ?)`, now, now, "ok", "")
	_ = legacy.Close()

	// Open a Store rooted in the same dir; migration should fire.
	store, err := NewStore(filepath.Join(dir, "operations.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	rows, _ := store.Recent(Filter{Limit: 10})
	if len(rows) != 1 {
		t.Fatalf("migrated rows=%d want 1", len(rows))
	}
	if rows[0].Name != "scheduler:trigger" {
		t.Errorf("migrated name=%s", rows[0].Name)
	}
	// Legacy file should have been renamed to .bak.
	if _, err := stat(legacyPath); err == nil {
		t.Errorf("legacy file still present at %s", legacyPath)
	}
	if _, err := stat(legacyPath + ".bak"); err != nil {
		t.Errorf("backup not created at %s.bak", legacyPath)
	}
}

func blockingRun(wg *sync.WaitGroup) RunFunc {
	return func(ctx context.Context, _ map[string]any, _ io.Writer) error {
		defer wg.Done()
		<-ctx.Done()
		return nil
	}
}

// openLegacy creates a SQLite file mimicking the pre-Phase-30H
// scheduler.db schema so we can exercise migration without depending on
// the legacy package.
func openLegacy(t *testing.T, path string) (*sql.DB, error) {
	t.Helper()
	db, err := sql.Open("sqlite", path+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}
	_, err = db.Exec(`CREATE TABLE scheduler_runs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		due_at TEXT NOT NULL,
		ran_at TEXT NOT NULL,
		status TEXT NOT NULL,
		error  TEXT NOT NULL DEFAULT ''
	)`)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func stat(path string) (os.FileInfo, error) { return os.Stat(path) }
