package ui

import (
	"context"
	"testing"
)

// Within the TTL, collectState must serve the cached snapshot instead of
// re-scanning the vault; invalidateState forces a fresh Collect.
func TestCollectState_CacheAndInvalidate(t *testing.T) {
	srv := newTestServer(t, nil)
	ctx := context.Background()

	st1 := srv.collectState(ctx)
	if got := len(st1.Pages); got != 0 {
		t.Fatalf("initial pages = %d, want 0", got)
	}

	// New content appears on disk after the snapshot was taken.
	writeTempMarkdown(t, srv.opts.Cfg.ContentDir, "posts/new.md",
		"---\ntitle: New\npublish: true\n---\n\nbody\n")

	if got := len(srv.collectState(ctx).Pages); got != 0 {
		t.Errorf("cached pages = %d, want 0 (snapshot must be reused within TTL)", got)
	}

	srv.invalidateState()
	if got := len(srv.collectState(ctx).Pages); got != 1 {
		t.Errorf("pages after invalidate = %d, want 1", got)
	}
}

// A negative StateTTL disables caching entirely.
func TestCollectState_NegativeTTLDisablesCache(t *testing.T) {
	srv := newTestServer(t, nil)
	srv.opts.StateTTL = -1
	ctx := context.Background()

	_ = srv.collectState(ctx)
	writeTempMarkdown(t, srv.opts.Cfg.ContentDir, "posts/new.md",
		"---\ntitle: New\npublish: true\n---\n\nbody\n")
	if got := len(srv.collectState(ctx).Pages); got != 1 {
		t.Errorf("pages = %d, want 1 (no caching with negative TTL)", got)
	}
}
