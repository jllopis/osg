package operations

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreBeginFinish(t *testing.T) {
	store := openTestStore(t)

	now := time.Now().UTC().Truncate(time.Second)
	id, err := store.Begin("build", KindTask, map[string]any{"force": true}, now)
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	if id == 0 {
		t.Fatalf("Begin returned id=0")
	}

	if err := store.Finish(id, StatusOK, "", now.Add(2*time.Second)); err != nil {
		t.Fatalf("Finish: %v", err)
	}

	rows, err := store.Recent(Filter{Limit: 10})
	if err != nil {
		t.Fatalf("Recent: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len rows=%d want 1", len(rows))
	}
	r := rows[0]
	if r.Name != "build" || r.Kind != KindTask || r.Status != StatusOK {
		t.Errorf("row=%+v", r)
	}
	if v, _ := r.Params["force"].(bool); !v {
		t.Errorf("params lost: %v", r.Params)
	}
}

func TestStoreFilters(t *testing.T) {
	store := openTestStore(t)
	now := time.Now().UTC()
	mustBegin(t, store, "build", KindTask, now.Add(-2*time.Hour))
	mustBegin(t, store, "deploy", KindTask, now.Add(-time.Hour))
	mustBegin(t, store, "serve", KindService, now)

	cases := []struct {
		name string
		f    Filter
		want int
	}{
		{"all", Filter{Limit: 10}, 3},
		{"by name", Filter{Name: "build", Limit: 10}, 1},
		{"by kind service", Filter{Kind: KindService, Limit: 10}, 1},
		{"by kind task", Filter{Kind: KindTask, Limit: 10}, 2},
		{"running only", Filter{Status: StatusRunning, Limit: 10}, 3},
		{"since", Filter{Since: now.Add(-30 * time.Minute), Limit: 10}, 1},
		{"limit clamp", Filter{Limit: 2}, 2},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := store.Recent(c.f)
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != c.want {
				t.Errorf("got %d want %d", len(got), c.want)
			}
		})
	}
}

func TestStoreMarkInterruptedRunning(t *testing.T) {
	store := openTestStore(t)
	now := time.Now().UTC()
	id1 := mustBegin(t, store, "build", KindTask, now)
	id2 := mustBegin(t, store, "deploy", KindTask, now)
	if err := store.Finish(id2, StatusOK, "", now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}

	affected, err := store.MarkInterruptedRunning(now.Add(2 * time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if affected != 1 {
		t.Errorf("affected=%d want 1 (only the build was running)", affected)
	}

	rows, _ := store.Recent(Filter{Name: "build", Limit: 1})
	if rows[0].Status != StatusCancelled || rows[0].Error != "shutdown" {
		t.Errorf("interrupted row=%+v", rows[0])
	}
	rows, _ = store.Recent(Filter{Name: "deploy", Limit: 1})
	if rows[0].Status != StatusOK {
		t.Errorf("deploy row tampered: %+v", rows[0])
	}
	_ = id1
}

func openTestStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "ops.db")
	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func mustBegin(t *testing.T, s *Store, name, kind string, started time.Time) int64 {
	t.Helper()
	id, err := s.Begin(name, kind, nil, started)
	if err != nil {
		t.Fatalf("Begin %s: %v", name, err)
	}
	return id
}
