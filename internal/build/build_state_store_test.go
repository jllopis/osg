package build

import (
	"testing"
	"time"

	"osg/internal/config"
)

func newTestBuildStateCfg(t *testing.T) config.Config {
	t.Helper()
	return config.Config{BuildCacheDir: t.TempDir()}
}

func TestBuildStateStore_RoundTrip(t *testing.T) {
	cfg := newTestBuildStateCfg(t)
	store, err := OpenBuildStateStore(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()

	publishAt := time.Now().Add(48 * time.Hour).UTC().Truncate(time.Second)
	entries := []DeferredEntry{
		{Path: "/2026/02/draft-post/", Source: "drafts/draft-post.md", Reason: DeferredReasonDraft},
		{Path: "/2026/06/scheduled/", Source: "posts/scheduled.md", Reason: DeferredReasonScheduled, PublishAt: publishAt},
	}
	if err := store.ReplaceDeferred(entries); err != nil {
		t.Fatalf("replace: %v", err)
	}
	got, err := store.LoadDeferred()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(got))
	}
	// Order: by path ascending.
	if got[0].Path != "/2026/02/draft-post/" || got[0].Reason != DeferredReasonDraft {
		t.Errorf("first row mismatch: %+v", got[0])
	}
	if got[1].Path != "/2026/06/scheduled/" || got[1].Reason != DeferredReasonScheduled {
		t.Errorf("second row mismatch: %+v", got[1])
	}
	if !got[1].PublishAt.Equal(publishAt) {
		t.Errorf("PublishAt = %v, want %v", got[1].PublishAt, publishAt)
	}
	if got[1].RecordedAt.IsZero() {
		t.Error("RecordedAt should be auto-stamped on insert")
	}
}

func TestBuildStateStore_ReplaceWipesPrevious(t *testing.T) {
	cfg := newTestBuildStateCfg(t)
	store, err := OpenBuildStateStore(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()

	if err := store.ReplaceDeferred([]DeferredEntry{
		{Path: "/old/", Source: "old.md", Reason: DeferredReasonDraft},
	}); err != nil {
		t.Fatalf("replace 1: %v", err)
	}
	if err := store.ReplaceDeferred([]DeferredEntry{
		{Path: "/new/", Source: "new.md", Reason: DeferredReasonDraft},
	}); err != nil {
		t.Fatalf("replace 2: %v", err)
	}
	got, err := store.LoadDeferred()
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(got) != 1 || got[0].Path != "/new/" {
		t.Errorf("expected only /new/ after second replace, got %+v", got)
	}
}

func TestBuildStateStore_ReplaceEmptyClears(t *testing.T) {
	cfg := newTestBuildStateCfg(t)
	store, err := OpenBuildStateStore(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer func() { _ = store.Close() }()

	_ = store.ReplaceDeferred([]DeferredEntry{
		{Path: "/will-go/", Source: "x.md", Reason: DeferredReasonDraft},
	})
	if err := store.ReplaceDeferred(nil); err != nil {
		t.Fatalf("replace nil: %v", err)
	}
	got, _ := store.LoadDeferred()
	if len(got) != 0 {
		t.Errorf("expected empty after replace(nil), got %d rows", len(got))
	}
}

func TestLoadDeferredPaths_HandlesMissingStore(t *testing.T) {
	// Point cache dir at a non-existent path; LoadDeferredPaths must
	// not panic, must return an empty slice, and must not block the
	// deploy from proceeding.
	cfg := config.Config{BuildCacheDir: ""}
	if got := LoadDeferredPaths(cfg, nil); len(got) != 0 {
		t.Errorf("expected empty slice from fresh store, got %v", got)
	}
}

func TestLoadDeferredPaths_ReadsRows(t *testing.T) {
	cfg := newTestBuildStateCfg(t)
	store, err := OpenBuildStateStore(cfg)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	_ = store.ReplaceDeferred([]DeferredEntry{
		{Path: "/a/", Source: "a.md", Reason: DeferredReasonDraft},
		{Path: "/b/", Source: "b.md", Reason: DeferredReasonScheduled},
	})
	_ = store.Close()

	paths := LoadDeferredPaths(cfg, nil)
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
	if paths[0] != "/a/" || paths[1] != "/b/" {
		t.Errorf("unexpected paths: %v", paths)
	}
}
