package api

import (
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

// --- helpers ---

func testCommentStore(t *testing.T) *CommentStore {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "comments.db")
	cs, err := NewCommentStore(dbPath, 30)
	if err != nil {
		t.Fatalf("NewCommentStore: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	return cs
}

// createTestUser is a convenience that upserts a user and fails on error.
func createTestUser(t *testing.T, cs *CommentStore, provider, pid, name string) *User {
	t.Helper()
	u, err := cs.UpsertUser(provider, pid, name, name+"@example.com", "https://avatar.example.com/"+name)
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	return u
}

// --- NewCommentStore ---

func TestNewCommentStore(t *testing.T) {
	cs := testCommentStore(t)
	if cs.db == nil {
		t.Fatal("expected non-nil db")
	}
}

func TestNewCommentStore_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sub", "deep")
	dbPath := filepath.Join(dir, "comments.db")
	cs, err := NewCommentStore(dbPath, 30)
	if err != nil {
		t.Fatalf("NewCommentStore: %v", err)
	}
	defer func() { _ = cs.Close() }()
}

func TestNewCommentStore_DefaultSessionDays(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "comments.db")
	cs, err := NewCommentStore(dbPath, 0)
	if err != nil {
		t.Fatalf("NewCommentStore: %v", err)
	}
	defer func() { _ = cs.Close() }()
	if cs.authSessionDays != 30 {
		t.Errorf("authSessionDays = %d, want 30", cs.authSessionDays)
	}
}

// --- User CRUD ---

func TestUpsertUser_Create(t *testing.T) {
	cs := testCommentStore(t)

	u, err := cs.UpsertUser("github", "12345", "Alice", "alice@example.com", "https://avatar.url/alice")
	if err != nil {
		t.Fatalf("UpsertUser: %v", err)
	}
	if u.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if u.Provider != "github" {
		t.Errorf("provider = %q, want github", u.Provider)
	}
	if u.ProviderID != "12345" {
		t.Errorf("provider_id = %q, want 12345", u.ProviderID)
	}
	if u.Name != "Alice" {
		t.Errorf("name = %q, want Alice", u.Name)
	}
	if u.Email != "alice@example.com" {
		t.Errorf("email = %q, want alice@example.com", u.Email)
	}
	if u.AvatarURL != "https://avatar.url/alice" {
		t.Errorf("avatar_url = %q, want https://avatar.url/alice", u.AvatarURL)
	}
}

func TestUpsertUser_Update(t *testing.T) {
	cs := testCommentStore(t)

	u1, _ := cs.UpsertUser("github", "12345", "Alice", "alice@a.com", "")
	u2, _ := cs.UpsertUser("github", "12345", "Alice Updated", "alice@b.com", "https://new-avatar")

	if u2.ID != u1.ID {
		t.Errorf("expected same ID after upsert, got %d != %d", u2.ID, u1.ID)
	}
	if u2.Name != "Alice Updated" {
		t.Errorf("name = %q, want Alice Updated", u2.Name)
	}
	if u2.Email != "alice@b.com" {
		t.Errorf("email = %q, want alice@b.com", u2.Email)
	}
	if u2.AvatarURL != "https://new-avatar" {
		t.Errorf("avatar_url = %q, want https://new-avatar", u2.AvatarURL)
	}
}

func TestGetUserByProvider_NotFound(t *testing.T) {
	cs := testCommentStore(t)

	u, err := cs.GetUserByProvider("github", "nonexistent")
	if err != nil {
		t.Fatalf("GetUserByProvider: %v", err)
	}
	if u != nil {
		t.Error("expected nil user for nonexistent provider_id")
	}
}

func TestGetUserByID(t *testing.T) {
	cs := testCommentStore(t)

	created := createTestUser(t, cs, "github", "42", "Bob")

	u, err := cs.GetUserByID(created.ID)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if u == nil {
		t.Fatal("expected non-nil user")
	}
	if u.Name != "Bob" {
		t.Errorf("name = %q, want Bob", u.Name)
	}
}

func TestGetUserByID_NotFound(t *testing.T) {
	cs := testCommentStore(t)

	u, err := cs.GetUserByID(9999)
	if err != nil {
		t.Fatalf("GetUserByID: %v", err)
	}
	if u != nil {
		t.Error("expected nil user for nonexistent ID")
	}
}

// --- Session CRUD ---

func TestCreateSession(t *testing.T) {
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	token, err := cs.CreateSession(user.ID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if len(token) != 64 { // 32 random bytes -> 64 hex chars
		t.Errorf("token length = %d, want 64", len(token))
	}
}

func TestValidateSession_Valid(t *testing.T) {
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	token, _ := cs.CreateSession(user.ID)
	u, err := cs.ValidateSession(token)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if u == nil {
		t.Fatal("expected non-nil user")
	}
	if u.ID != user.ID {
		t.Errorf("user ID = %d, want %d", u.ID, user.ID)
	}
}

func TestValidateSession_InvalidToken(t *testing.T) {
	cs := testCommentStore(t)

	u, err := cs.ValidateSession("nonexistent-token")
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if u != nil {
		t.Error("expected nil user for invalid token")
	}
}

func TestValidateSession_ExpiredToken(t *testing.T) {
	// Use a store with 0 session days (defaults to 30), but we manually
	// set the expires_at in the past.
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	token, _ := cs.CreateSession(user.ID)

	// Force-expire the session.
	past := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339Nano)
	_, _ = cs.db.Exec(`UPDATE sessions SET expires_at = ? WHERE token = ?`, past, token)

	u, err := cs.ValidateSession(token)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if u != nil {
		t.Error("expected nil user for expired session")
	}
}

func TestDeleteSession(t *testing.T) {
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	token, _ := cs.CreateSession(user.ID)
	if err := cs.DeleteSession(token); err != nil {
		t.Fatalf("DeleteSession: %v", err)
	}

	u, _ := cs.ValidateSession(token)
	if u != nil {
		t.Error("expected nil user after session deletion")
	}
}

func TestCleanExpiredSessions(t *testing.T) {
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	token1, _ := cs.CreateSession(user.ID)
	token2, _ := cs.CreateSession(user.ID)

	// Expire token1.
	past := time.Now().UTC().Add(-1 * time.Hour).Format(time.RFC3339Nano)
	_, _ = cs.db.Exec(`UPDATE sessions SET expires_at = ? WHERE token = ?`, past, token1)

	if err := cs.CleanExpiredSessions(); err != nil {
		t.Fatalf("CleanExpiredSessions: %v", err)
	}

	// token1 should be gone.
	u1, _ := cs.ValidateSession(token1)
	if u1 != nil {
		t.Error("expected expired session to be cleaned")
	}
	// token2 should still work.
	u2, _ := cs.ValidateSession(token2)
	if u2 == nil {
		t.Error("expected valid session to survive cleanup")
	}
}

// --- Comment CRUD ---

func TestCreateComment_TopLevel(t *testing.T) {
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	c, err := cs.CreateComment("/posts/hello/", user.ID, 0, "Hello world!")
	if err != nil {
		t.Fatalf("CreateComment: %v", err)
	}
	if c.ID == 0 {
		t.Error("expected non-zero ID")
	}
	if c.Body != "Hello world!" {
		t.Errorf("body = %q, want Hello world!", c.Body)
	}
	if c.Author == nil {
		t.Fatal("expected non-nil author")
	}
	if c.Author.Name != "Alice" {
		t.Errorf("author name = %q, want Alice", c.Author.Name)
	}
	if c.ParentID.Valid {
		t.Error("expected null parent_id for top-level comment")
	}
}

func TestCreateComment_Reply(t *testing.T) {
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	parent, _ := cs.CreateComment("/posts/hello/", user.ID, 0, "Parent")
	child, err := cs.CreateComment("/posts/hello/", user.ID, parent.ID, "Reply")
	if err != nil {
		t.Fatalf("CreateComment reply: %v", err)
	}
	if !child.ParentID.Valid || child.ParentID.Int64 != parent.ID {
		t.Errorf("parent_id = %v, want %d", child.ParentID, parent.ID)
	}
}

func TestGetComment(t *testing.T) {
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	created, _ := cs.CreateComment("/posts/hello/", user.ID, 0, "Test")
	fetched, err := cs.GetComment(created.ID)
	if err != nil {
		t.Fatalf("GetComment: %v", err)
	}
	if fetched == nil {
		t.Fatal("expected non-nil comment")
	}
	if fetched.Body != "Test" {
		t.Errorf("body = %q, want Test", fetched.Body)
	}
}

func TestGetComment_NotFound(t *testing.T) {
	cs := testCommentStore(t)

	c, err := cs.GetComment(9999)
	if err != nil {
		t.Fatalf("GetComment: %v", err)
	}
	if c != nil {
		t.Error("expected nil for nonexistent comment")
	}
}

func TestSoftDeleteComment(t *testing.T) {
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	c, _ := cs.CreateComment("/posts/hello/", user.ID, 0, "To delete")
	if err := cs.SoftDeleteComment(c.ID); err != nil {
		t.Fatalf("SoftDeleteComment: %v", err)
	}

	fetched, _ := cs.GetComment(c.ID)
	if fetched == nil {
		t.Fatal("expected comment to still exist after soft delete")
	}
	if !fetched.Deleted {
		t.Error("expected deleted = true")
	}
	if fetched.Body != "" {
		t.Errorf("body should be empty after soft delete, got %q", fetched.Body)
	}
}

// --- ListComments + tree building ---

func TestListComments_Empty(t *testing.T) {
	cs := testCommentStore(t)

	comments, err := cs.ListComments("/empty-page/")
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if len(comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(comments))
	}
}

func TestListComments_FlatList(t *testing.T) {
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	_, _ = cs.CreateComment("/posts/hello/", user.ID, 0, "First")
	_, _ = cs.CreateComment("/posts/hello/", user.ID, 0, "Second")
	_, _ = cs.CreateComment("/posts/hello/", user.ID, 0, "Third")

	comments, err := cs.ListComments("/posts/hello/")
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if len(comments) != 3 {
		t.Fatalf("expected 3 root comments, got %d", len(comments))
	}
	if comments[0].Body != "First" {
		t.Errorf("first comment = %q, want First", comments[0].Body)
	}
}

func TestListComments_NestedTree(t *testing.T) {
	cs := testCommentStore(t)
	alice := createTestUser(t, cs, "github", "1", "Alice")
	bob := createTestUser(t, cs, "github", "2", "Bob")

	root, _ := cs.CreateComment("/posts/hello/", alice.ID, 0, "Root comment")
	reply1, _ := cs.CreateComment("/posts/hello/", bob.ID, root.ID, "Reply to root")
	_, _ = cs.CreateComment("/posts/hello/", alice.ID, reply1.ID, "Reply to reply")

	comments, err := cs.ListComments("/posts/hello/")
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 root comment, got %d", len(comments))
	}
	if len(comments[0].Replies) != 1 {
		t.Fatalf("expected 1 reply to root, got %d", len(comments[0].Replies))
	}
	if len(comments[0].Replies[0].Replies) != 1 {
		t.Fatalf("expected 1 nested reply, got %d", len(comments[0].Replies[0].Replies))
	}
	if comments[0].Replies[0].Replies[0].Body != "Reply to reply" {
		t.Errorf("nested reply body = %q, want Reply to reply", comments[0].Replies[0].Replies[0].Body)
	}
}

func TestListComments_DifferentPages(t *testing.T) {
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	_, _ = cs.CreateComment("/posts/a/", user.ID, 0, "Page A")
	_, _ = cs.CreateComment("/posts/b/", user.ID, 0, "Page B")

	commentsA, _ := cs.ListComments("/posts/a/")
	commentsB, _ := cs.ListComments("/posts/b/")

	if len(commentsA) != 1 {
		t.Errorf("page A: expected 1 comment, got %d", len(commentsA))
	}
	if len(commentsB) != 1 {
		t.Errorf("page B: expected 1 comment, got %d", len(commentsB))
	}
}

func TestListComments_SoftDeletedLeafPruned(t *testing.T) {
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	c, _ := cs.CreateComment("/posts/hello/", user.ID, 0, "To delete")
	_ = cs.SoftDeleteComment(c.ID)

	comments, err := cs.ListComments("/posts/hello/")
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	// Deleted leaf with no replies should be pruned.
	if len(comments) != 0 {
		t.Errorf("expected 0 comments (deleted leaf pruned), got %d", len(comments))
	}
}

func TestListComments_SoftDeletedWithRepliesPreserved(t *testing.T) {
	cs := testCommentStore(t)
	alice := createTestUser(t, cs, "github", "1", "Alice")
	bob := createTestUser(t, cs, "github", "2", "Bob")

	root, _ := cs.CreateComment("/posts/hello/", alice.ID, 0, "Root")
	_, _ = cs.CreateComment("/posts/hello/", bob.ID, root.ID, "Reply")
	_ = cs.SoftDeleteComment(root.ID)

	comments, err := cs.ListComments("/posts/hello/")
	if err != nil {
		t.Fatalf("ListComments: %v", err)
	}
	// Root should be preserved (has replies) but anonymised.
	if len(comments) != 1 {
		t.Fatalf("expected 1 root comment, got %d", len(comments))
	}
	if comments[0].Body != "" {
		t.Errorf("deleted comment body should be empty, got %q", comments[0].Body)
	}
	if comments[0].Author.Name != "" {
		t.Errorf("deleted comment author should be anonymous, got %q", comments[0].Author.Name)
	}
	if len(comments[0].Replies) != 1 {
		t.Errorf("expected 1 reply under deleted root, got %d", len(comments[0].Replies))
	}
	if comments[0].Replies[0].Body != "Reply" {
		t.Errorf("reply body = %q, want Reply", comments[0].Replies[0].Body)
	}
}

// --- buildCommentTree unit tests ---

func TestBuildCommentTree_Empty(t *testing.T) {
	result := buildCommentTree(nil)
	if len(result) != 0 {
		t.Errorf("expected 0 roots, got %d", len(result))
	}
}

func TestBuildCommentTree_OrphanedParent(t *testing.T) {
	// Comment with a parent_id that doesn't exist should become a root.
	flat := []*Comment{
		{
			ID:       1,
			ParentID: sql.NullInt64{Int64: 999, Valid: true},
			Body:     "Orphan",
			Author:   &CommentAuthor{Name: "Alice"},
		},
	}
	result := buildCommentTree(flat)
	if len(result) != 1 {
		t.Fatalf("expected 1 root, got %d", len(result))
	}
	if result[0].Body != "Orphan" {
		t.Errorf("body = %q, want Orphan", result[0].Body)
	}
}

func TestPruneDeletedLeaves_DeepNesting(t *testing.T) {
	// root -> reply -> deleted-leaf
	// The deleted leaf is pruned, root and reply remain.
	root := &Comment{ID: 1, Body: "Root", Author: &CommentAuthor{Name: "A"}}
	reply := &Comment{ID: 2, Body: "Reply", Author: &CommentAuthor{Name: "B"}}
	deletedLeaf := &Comment{ID: 3, Deleted: true, Body: "", Author: &CommentAuthor{Name: ""}}

	reply.Replies = []*Comment{deletedLeaf}
	root.Replies = []*Comment{reply}

	result := pruneDeletedLeaves([]*Comment{root})
	if len(result) != 1 {
		t.Fatalf("expected 1 root, got %d", len(result))
	}
	if len(result[0].Replies) != 1 {
		t.Fatalf("expected 1 reply, got %d", len(result[0].Replies))
	}
	if len(result[0].Replies[0].Replies) != 0 {
		t.Errorf("expected deleted leaf pruned, got %d replies", len(result[0].Replies[0].Replies))
	}
}

func TestPruneDeletedLeaves_DeletedChain(t *testing.T) {
	// deleted-root -> deleted-reply (all leaves deleted → everything pruned)
	root := &Comment{ID: 1, Deleted: true, Body: "", Author: &CommentAuthor{Name: ""}}
	reply := &Comment{ID: 2, Deleted: true, Body: "", Author: &CommentAuthor{Name: ""}}
	root.Replies = []*Comment{reply}

	result := pruneDeletedLeaves([]*Comment{root})
	if len(result) != 0 {
		t.Errorf("expected all pruned, got %d", len(result))
	}
}

// --- generateToken ---

func TestGenerateToken_UniqueAndLength(t *testing.T) {
	t1, err := generateToken()
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	t2, _ := generateToken()

	if len(t1) != 64 {
		t.Errorf("token length = %d, want 64", len(t1))
	}
	if t1 == t2 {
		t.Error("two generated tokens should be different")
	}
}

// --- Close ---

func TestCommentStore_Close(t *testing.T) {
	cs := testCommentStore(t)
	if err := cs.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// After close, operations should fail.
	_, err := cs.ListComments("/test/")
	if err == nil {
		t.Error("expected error after Close")
	}
}

// --- Edge: session with deleted user (FK CASCADE) ---

func TestSession_CascadeDeleteUser(t *testing.T) {
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")
	token, _ := cs.CreateSession(user.ID)

	// Delete the user directly.
	_, _ = cs.db.Exec(`DELETE FROM users WHERE id = ?`, user.ID)

	// Session should no longer resolve.
	u, err := cs.ValidateSession(token)
	if err != nil {
		t.Fatalf("ValidateSession: %v", err)
	}
	if u != nil {
		t.Error("expected nil user after cascade delete")
	}
}

// --- Edge: comment creation timestamps ---

func TestCreateComment_HasTimestamps(t *testing.T) {
	cs := testCommentStore(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	c, _ := cs.CreateComment("/posts/hello/", user.ID, 0, "Test")
	if c.CreatedAt == "" {
		t.Error("expected non-empty created_at")
	}
	if c.UpdatedAt == "" {
		t.Error("expected non-empty updated_at")
	}

	// Parse to ensure valid RFC3339.
	if _, err := time.Parse(time.RFC3339Nano, c.CreatedAt); err != nil {
		t.Errorf("created_at not valid RFC3339Nano: %v", err)
	}
}
