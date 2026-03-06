package api

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// --- helpers ---

func testCommentHandlers(t *testing.T) (*CommentHandlers, *CommentStore) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "comments.db")
	cs, err := NewCommentStore(dbPath, 30)
	if err != nil {
		t.Fatalf("NewCommentStore: %v", err)
	}
	t.Cleanup(func() { _ = cs.Close() })
	h := NewCommentHandlers(cs, slog.Default())
	return h, cs
}

// authenticatedRequest adds an osg_session cookie for the given user.
func authenticatedRequest(t *testing.T, cs *CommentStore, userID int64, method, path string, body any) *http.Request {
	t.Helper()
	token, err := cs.CreateSession(userID)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.AddCookie(&http.Cookie{Name: "osg_session", Value: token})
	return req
}

// --- CreateCommentRequest.Validate ---

func TestCreateCommentRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		req     CreateCommentRequest
		wantErr string
	}{
		{"valid", CreateCommentRequest{PagePath: "/posts/hello/", Body: "Hello"}, ""},
		{"empty page", CreateCommentRequest{PagePath: "", Body: "Hello"}, "page_path is required"},
		{"no leading slash", CreateCommentRequest{PagePath: "posts/hello/", Body: "Hello"}, "page_path must start with /"},
		{"empty body", CreateCommentRequest{PagePath: "/a/", Body: ""}, "body is required"},
		{"whitespace body", CreateCommentRequest{PagePath: "/a/", Body: "   "}, "body is required"},
		{"body too long", CreateCommentRequest{PagePath: "/a/", Body: strings.Repeat("x", 10001)}, "body too long"},
		{"body at limit", CreateCommentRequest{PagePath: "/a/", Body: strings.Repeat("x", 10000)}, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("error = %q, want containing %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

// --- HandleList ---

func TestHandleList_MissingPage(t *testing.T) {
	h, _ := testCommentHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/comments", nil)
	rec := httptest.NewRecorder()
	h.HandleList(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleList_InvalidPage(t *testing.T) {
	h, _ := testCommentHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/comments?page=no-slash", nil)
	rec := httptest.NewRecorder()
	h.HandleList(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleList_EmptyPage(t *testing.T) {
	h, _ := testCommentHandlers(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/comments?page=/empty/", nil)
	rec := httptest.NewRecorder()
	h.HandleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp CommentsResponse
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Comments) != 0 {
		t.Errorf("expected 0 comments, got %d", len(resp.Comments))
	}
	if resp.User != nil {
		t.Error("expected nil user when not authenticated")
	}
}

func TestHandleList_WithComments(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")
	_, _ = cs.CreateComment("/posts/hello/", user.ID, 0, "Comment 1")
	_, _ = cs.CreateComment("/posts/hello/", user.ID, 0, "Comment 2")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/comments?page=/posts/hello/", nil)
	rec := httptest.NewRecorder()
	h.HandleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp CommentsResponse
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if len(resp.Comments) != 2 {
		t.Errorf("expected 2 comments, got %d", len(resp.Comments))
	}
}

func TestHandleList_IncludesAuthenticatedUser(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	req := authenticatedRequest(t, cs, user.ID, http.MethodGet, "/api/v1/comments?page=/posts/hello/", nil)
	rec := httptest.NewRecorder()
	h.HandleList(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	var resp CommentsResponse
	_ = json.NewDecoder(rec.Body).Decode(&resp)
	if resp.User == nil {
		t.Error("expected non-nil user when authenticated")
	}
}

// --- HandleCreate ---

func TestHandleCreate_Success(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	body := CreateCommentRequest{PagePath: "/posts/hello/", Body: "Great post!"}
	req := authenticatedRequest(t, cs, user.ID, http.MethodPost, "/api/v1/comments", body)
	rec := httptest.NewRecorder()
	h.HandleCreate(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}

	var c Comment
	_ = json.NewDecoder(rec.Body).Decode(&c)
	if c.Body != "Great post!" {
		t.Errorf("body = %q, want Great post!", c.Body)
	}
	if c.Author == nil || c.Author.Name != "Alice" {
		t.Error("expected author Alice")
	}
}

func TestHandleCreate_Unauthenticated(t *testing.T) {
	h, _ := testCommentHandlers(t)

	b, _ := json.Marshal(CreateCommentRequest{PagePath: "/posts/hello/", Body: "Hi"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/comments", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.HandleCreate(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestHandleCreate_InvalidJSON(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	req := authenticatedRequest(t, cs, user.ID, http.MethodPost, "/api/v1/comments", nil)
	// Override body with invalid JSON.
	req.Body = http.NoBody
	req = httptest.NewRequest(http.MethodPost, "/api/v1/comments", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	token, _ := cs.CreateSession(user.ID)
	req.AddCookie(&http.Cookie{Name: "osg_session", Value: token})
	rec := httptest.NewRecorder()
	h.HandleCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCreate_ValidationError(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	body := CreateCommentRequest{PagePath: "", Body: "Hello"}
	req := authenticatedRequest(t, cs, user.ID, http.MethodPost, "/api/v1/comments", body)
	rec := httptest.NewRecorder()
	h.HandleCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCreate_Reply(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	parent, _ := cs.CreateComment("/posts/hello/", user.ID, 0, "Parent")

	body := CreateCommentRequest{PagePath: "/posts/hello/", ParentID: parent.ID, Body: "Reply"}
	req := authenticatedRequest(t, cs, user.ID, http.MethodPost, "/api/v1/comments", body)
	rec := httptest.NewRecorder()
	h.HandleCreate(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}
}

func TestHandleCreate_ReplyToNonexistent(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	body := CreateCommentRequest{PagePath: "/posts/hello/", ParentID: 9999, Body: "Reply"}
	req := authenticatedRequest(t, cs, user.ID, http.MethodPost, "/api/v1/comments", body)
	rec := httptest.NewRecorder()
	h.HandleCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCreate_ReplyWrongPage(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	parent, _ := cs.CreateComment("/posts/page-a/", user.ID, 0, "Parent on page A")

	body := CreateCommentRequest{PagePath: "/posts/page-b/", ParentID: parent.ID, Body: "Reply on page B"}
	req := authenticatedRequest(t, cs, user.ID, http.MethodPost, "/api/v1/comments", body)
	rec := httptest.NewRecorder()
	h.HandleCreate(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleCreate_TrimsBody(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	body := CreateCommentRequest{PagePath: "/posts/hello/", Body: "  trimmed  "}
	req := authenticatedRequest(t, cs, user.ID, http.MethodPost, "/api/v1/comments", body)
	rec := httptest.NewRecorder()
	h.HandleCreate(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", rec.Code)
	}

	var c Comment
	_ = json.NewDecoder(rec.Body).Decode(&c)
	if c.Body != "trimmed" {
		t.Errorf("body = %q, want trimmed", c.Body)
	}
}

// --- HandleDelete ---

func TestHandleDelete_Success(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")
	comment, _ := cs.CreateComment("/posts/hello/", user.ID, 0, "To delete")

	req := authenticatedRequest(t, cs, user.ID, http.MethodDelete, "/api/v1/comments/"+idStr(comment.ID), nil)
	req.SetPathValue("id", idStr(comment.ID))
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	// Verify soft-deleted.
	c, _ := cs.GetComment(comment.ID)
	if !c.Deleted {
		t.Error("expected comment to be soft-deleted")
	}
}

func TestHandleDelete_Unauthenticated(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")
	comment, _ := cs.CreateComment("/posts/hello/", user.ID, 0, "Comment")

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/comments/"+idStr(comment.ID), nil)
	req.SetPathValue("id", idStr(comment.ID))
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", rec.Code)
	}
}

func TestHandleDelete_NotFound(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	req := authenticatedRequest(t, cs, user.ID, http.MethodDelete, "/api/v1/comments/9999", nil)
	req.SetPathValue("id", "9999")
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rec.Code)
	}
}

func TestHandleDelete_WrongUser(t *testing.T) {
	h, cs := testCommentHandlers(t)
	alice := createTestUser(t, cs, "github", "1", "Alice")
	bob := createTestUser(t, cs, "github", "2", "Bob")

	comment, _ := cs.CreateComment("/posts/hello/", alice.ID, 0, "Alice's comment")

	req := authenticatedRequest(t, cs, bob.ID, http.MethodDelete, "/api/v1/comments/"+idStr(comment.ID), nil)
	req.SetPathValue("id", idStr(comment.ID))
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", rec.Code)
	}
}

func TestHandleDelete_InvalidID(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	req := authenticatedRequest(t, cs, user.ID, http.MethodDelete, "/api/v1/comments/abc", nil)
	req.SetPathValue("id", "abc")
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

func TestHandleDelete_ZeroID(t *testing.T) {
	h, cs := testCommentHandlers(t)
	user := createTestUser(t, cs, "github", "1", "Alice")

	req := authenticatedRequest(t, cs, user.ID, http.MethodDelete, "/api/v1/comments/0", nil)
	req.SetPathValue("id", "0")
	rec := httptest.NewRecorder()
	h.HandleDelete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", rec.Code)
	}
}

// --- helper ---

func idStr(id int64) string {
	return strconv.FormatInt(id, 10)
}
