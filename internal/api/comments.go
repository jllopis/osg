package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
)

// CommentHandlers groups the HTTP handlers for the comment system.
type CommentHandlers struct {
	store  *CommentStore
	logger *slog.Logger
}

// NewCommentHandlers creates the comment handler group.
func NewCommentHandlers(store *CommentStore, logger *slog.Logger) *CommentHandlers {
	return &CommentHandlers{store: store, logger: logger}
}

// CommentsResponse is the JSON response for GET /api/v1/comments.
type CommentsResponse struct {
	Comments []*Comment `json:"comments"`
	User     any        `json:"user"` // *User or nil
}

// CreateCommentRequest is the JSON body for POST /api/v1/comments.
type CreateCommentRequest struct {
	PagePath string `json:"page_path"`
	ParentID int64  `json:"parent_id,omitempty"`
	Body     string `json:"body"`
}

// Validate checks that a CreateCommentRequest has valid fields.
func (r CreateCommentRequest) Validate() error {
	if strings.TrimSpace(r.PagePath) == "" {
		return fmt.Errorf("page_path is required")
	}
	if !strings.HasPrefix(r.PagePath, "/") {
		return fmt.Errorf("page_path must start with /")
	}
	body := strings.TrimSpace(r.Body)
	if body == "" {
		return fmt.Errorf("body is required")
	}
	if len(body) > 10000 {
		return fmt.Errorf("body too long (max 10000 chars)")
	}
	return nil
}

// HandleList returns threaded comments for a page.
// GET /api/v1/comments?page=/path/to/page/
func (h *CommentHandlers) HandleList(w http.ResponseWriter, r *http.Request) {
	pagePath := r.URL.Query().Get("page")
	if pagePath == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "page query parameter is required"})
		return
	}
	if !strings.HasPrefix(pagePath, "/") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "page must start with /"})
		return
	}

	comments, err := h.store.ListComments(pagePath)
	if err != nil {
		h.logger.Error("list comments", "error", err, "page", pagePath)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	if comments == nil {
		comments = []*Comment{}
	}

	// Include current user if authenticated.
	user := getUserFromRequest(r, h.store)
	var userView any
	if user != nil {
		userView = map[string]any{
			"id":         user.ID,
			"name":       user.Name,
			"avatar_url": user.AvatarURL,
		}
	}

	writeJSON(w, http.StatusOK, CommentsResponse{
		Comments: comments,
		User:     userView,
	})
}

// HandleCreate creates a new comment. Requires authentication.
// POST /api/v1/comments
func (h *CommentHandlers) HandleCreate(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r, h.store)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	var req CreateCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON"})
		return
	}
	if err := req.Validate(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// If replying to a parent, verify it exists and belongs to the same page.
	if req.ParentID > 0 {
		parent, err := h.store.GetComment(req.ParentID)
		if err != nil {
			h.logger.Error("get parent comment", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
			return
		}
		if parent == nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parent comment not found"})
			return
		}
		if parent.PagePath != req.PagePath {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "parent comment belongs to a different page"})
			return
		}
	}

	comment, err := h.store.CreateComment(req.PagePath, user.ID, req.ParentID, strings.TrimSpace(req.Body))
	if err != nil {
		h.logger.Error("create comment", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusCreated, comment)
}

// HandleDelete soft-deletes a comment. Users can only delete their own.
// DELETE /api/v1/comments/{id}
func (h *CommentHandlers) HandleDelete(w http.ResponseWriter, r *http.Request) {
	user := getUserFromRequest(r, h.store)
	if user == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return
	}

	idStr := r.PathValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid comment id"})
		return
	}

	comment, err := h.store.GetComment(id)
	if err != nil {
		h.logger.Error("get comment for delete", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}
	if comment == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "comment not found"})
		return
	}

	// Only the comment author can delete it.
	if comment.UserID != user.ID {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "cannot delete another user's comment"})
		return
	}

	if err := h.store.SoftDeleteComment(id); err != nil {
		h.logger.Error("soft delete comment", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal error"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
