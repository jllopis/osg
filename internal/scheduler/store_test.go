package scheduler

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreRoundTrip(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, ".osg", "scheduler.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	now := time.Now().Truncate(time.Second)
	runs := []Run{
		{DueAt: now.Add(-2 * time.Hour), RanAt: now.Add(-2 * time.Hour), Status: "ok"},
		{DueAt: now.Add(-time.Hour), RanAt: now.Add(-time.Hour), Status: "error", Error: "build failed"},
		{DueAt: now, RanAt: now, Status: "ok"},
	}
	for _, r := range runs {
		if err := store.Record(r); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}

	got, err := store.Recent(10)
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("Recent len=%d want 3", len(got))
	}
	// DESC order: most recent first.
	if got[0].Status != "ok" || got[1].Status != "error" || got[2].Status != "ok" {
		t.Errorf("status order=%v %v %v", got[0].Status, got[1].Status, got[2].Status)
	}
	if got[1].Error != "build failed" {
		t.Errorf("error msg=%q", got[1].Error)
	}
}

func TestStoreLimit(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "s.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()

	now := time.Now()
	for i := 0; i < 5; i++ {
		_ = store.Record(Run{DueAt: now, RanAt: now.Add(time.Duration(i) * time.Second), Status: "ok"})
	}
	got, _ := store.Recent(2)
	if len(got) != 2 {
		t.Errorf("limit=%d want 2", len(got))
	}
}
