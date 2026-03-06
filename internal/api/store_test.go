package api

import (
	"os"
	"path/filepath"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	store, err := NewStore(dbPath, 24)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func TestNewStore(t *testing.T) {
	store := testStore(t)
	if store.db == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestNewStore_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub", "dir")
	dbPath := filepath.Join(dir, "test.db")
	store, err := NewStore(dbPath, 24)
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer func() { _ = store.Close() }()

	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected directory to exist: %v", err)
	}
}

func TestRecordView_Basic(t *testing.T) {
	store := testStore(t)

	stats, err := store.RecordView("/posts/hello", "fp1")
	if err != nil {
		t.Fatalf("RecordView: %v", err)
	}
	if stats.Views != 1 {
		t.Errorf("views = %d, want 1", stats.Views)
	}
	if stats.Unique != 1 {
		t.Errorf("unique = %d, want 1", stats.Unique)
	}
}

func TestRecordView_DedupSameFingerprint(t *testing.T) {
	store := testStore(t)

	// Same fingerprint, same page — daily dedup means second insert is ignored.
	_, _ = store.RecordView("/posts/hello", "fp1")
	stats, err := store.RecordView("/posts/hello", "fp1")
	if err != nil {
		t.Fatalf("RecordView: %v", err)
	}
	// INSERT OR IGNORE means only 1 row exists (deduped by day).
	if stats.Views != 1 {
		t.Errorf("views = %d, want 1 (deduped)", stats.Views)
	}
	if stats.Unique != 1 {
		t.Errorf("unique = %d, want 1", stats.Unique)
	}
}

func TestRecordView_DifferentFingerprints(t *testing.T) {
	store := testStore(t)

	_, _ = store.RecordView("/posts/hello", "fp1")
	stats, err := store.RecordView("/posts/hello", "fp2")
	if err != nil {
		t.Fatalf("RecordView: %v", err)
	}
	if stats.Views != 2 {
		t.Errorf("views = %d, want 2", stats.Views)
	}
	if stats.Unique != 2 {
		t.Errorf("unique = %d, want 2", stats.Unique)
	}
}

func TestRecordView_DifferentPages(t *testing.T) {
	store := testStore(t)

	_, _ = store.RecordView("/posts/a", "fp1")
	_, _ = store.RecordView("/posts/b", "fp1")

	statsA, _ := store.GetStats("/posts/a", "fp1")
	statsB, _ := store.GetStats("/posts/b", "fp1")

	if statsA.Views != 1 {
		t.Errorf("page a views = %d, want 1", statsA.Views)
	}
	if statsB.Views != 1 {
		t.Errorf("page b views = %d, want 1", statsB.Views)
	}
}

func TestVote_Like(t *testing.T) {
	store := testStore(t)

	stats, err := store.Vote("/posts/hello", "fp1", 1)
	if err != nil {
		t.Fatalf("Vote: %v", err)
	}
	if stats.Likes != 1 {
		t.Errorf("likes = %d, want 1", stats.Likes)
	}
	if stats.Dislikes != 0 {
		t.Errorf("dislikes = %d, want 0", stats.Dislikes)
	}
	if stats.UserVote != 1 {
		t.Errorf("user_vote = %d, want 1", stats.UserVote)
	}
}

func TestVote_Dislike(t *testing.T) {
	store := testStore(t)

	stats, err := store.Vote("/posts/hello", "fp1", -1)
	if err != nil {
		t.Fatalf("Vote: %v", err)
	}
	if stats.Likes != 0 {
		t.Errorf("likes = %d, want 0", stats.Likes)
	}
	if stats.Dislikes != 1 {
		t.Errorf("dislikes = %d, want 1", stats.Dislikes)
	}
	if stats.UserVote != -1 {
		t.Errorf("user_vote = %d, want -1", stats.UserVote)
	}
}

func TestVote_ChangeVote(t *testing.T) {
	store := testStore(t)

	_, _ = store.Vote("/posts/hello", "fp1", 1)
	stats, err := store.Vote("/posts/hello", "fp1", -1)
	if err != nil {
		t.Fatalf("Vote: %v", err)
	}
	if stats.Likes != 0 {
		t.Errorf("likes = %d, want 0", stats.Likes)
	}
	if stats.Dislikes != 1 {
		t.Errorf("dislikes = %d, want 1", stats.Dislikes)
	}
	if stats.UserVote != -1 {
		t.Errorf("user_vote = %d, want -1", stats.UserVote)
	}
}

func TestVote_Retract(t *testing.T) {
	store := testStore(t)

	_, _ = store.Vote("/posts/hello", "fp1", 1)
	stats, err := store.Vote("/posts/hello", "fp1", 0)
	if err != nil {
		t.Fatalf("Vote: %v", err)
	}
	if stats.Likes != 0 {
		t.Errorf("likes = %d, want 0", stats.Likes)
	}
	if stats.UserVote != 0 {
		t.Errorf("user_vote = %d, want 0", stats.UserVote)
	}
}

func TestVote_MultipleUsers(t *testing.T) {
	store := testStore(t)

	_, _ = store.Vote("/posts/hello", "fp1", 1)
	_, _ = store.Vote("/posts/hello", "fp2", 1)
	_, _ = store.Vote("/posts/hello", "fp3", -1)

	stats, _ := store.GetStats("/posts/hello", "fp3")
	if stats.Likes != 2 {
		t.Errorf("likes = %d, want 2", stats.Likes)
	}
	if stats.Dislikes != 1 {
		t.Errorf("dislikes = %d, want 1", stats.Dislikes)
	}
	if stats.UserVote != -1 {
		t.Errorf("user_vote = %d, want -1", stats.UserVote)
	}
}

func TestGetStats_NoData(t *testing.T) {
	store := testStore(t)

	stats, err := store.GetStats("/nonexistent", "fp1")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.Views != 0 || stats.Unique != 0 || stats.Likes != 0 || stats.Dislikes != 0 || stats.UserVote != 0 {
		t.Errorf("expected all zeros, got %+v", stats)
	}
}

func TestGetStats_ViewsAndVotes(t *testing.T) {
	store := testStore(t)

	_, _ = store.RecordView("/posts/hello", "fp1")
	_, _ = store.RecordView("/posts/hello", "fp2")
	_, _ = store.Vote("/posts/hello", "fp1", 1)
	_, _ = store.Vote("/posts/hello", "fp2", -1)

	stats, err := store.GetStats("/posts/hello", "fp1")
	if err != nil {
		t.Fatalf("GetStats: %v", err)
	}
	if stats.Views != 2 {
		t.Errorf("views = %d, want 2", stats.Views)
	}
	if stats.Unique != 2 {
		t.Errorf("unique = %d, want 2", stats.Unique)
	}
	if stats.Likes != 1 {
		t.Errorf("likes = %d, want 1", stats.Likes)
	}
	if stats.Dislikes != 1 {
		t.Errorf("dislikes = %d, want 1", stats.Dislikes)
	}
	if stats.UserVote != 1 {
		t.Errorf("user_vote = %d, want 1", stats.UserVote)
	}
}
