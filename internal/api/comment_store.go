package api

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// CommentStore manages the separate SQLite database for comments,
// users, and sessions. It is intentionally separate from the
// interactions Store for future portability to PostgreSQL.
type CommentStore struct {
	db              *sql.DB
	authSessionDays int
}

// User represents an authenticated user from an OAuth2 provider.
type User struct {
	ID         int64  `json:"id"`
	Provider   string `json:"provider"`
	ProviderID string `json:"provider_id"`
	Name       string `json:"name"`
	Email      string `json:"email,omitempty"`
	AvatarURL  string `json:"avatar_url,omitempty"`
	CreatedAt  string `json:"created_at"`
}

// Session ties a cookie token to a user.
type Session struct {
	Token     string
	UserID    int64
	CreatedAt string
	ExpiresAt string
}

// Comment represents a single comment (flat, before tree building).
type Comment struct {
	ID        int64          `json:"id"`
	PagePath  string         `json:"page_path,omitempty"`
	UserID    int64          `json:"-"`
	ParentID  sql.NullInt64  `json:"-"`
	Body      string         `json:"body"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at,omitempty"`
	Deleted   bool           `json:"-"`
	Author    *CommentAuthor `json:"author"`
	Replies   []*Comment     `json:"replies,omitempty"`
}

// CommentAuthor is the public view of a comment's author.
type CommentAuthor struct {
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// NewCommentStore opens (or creates) the comments SQLite database.
func NewCommentStore(dbPath string, authSessionDays int) (*CommentStore, error) {
	if authSessionDays <= 0 {
		authSessionDays = 30
	}

	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create comments db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=ON")
	if err != nil {
		return nil, fmt.Errorf("open comments db: %w", err)
	}

	if err := migrateComments(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate comments db: %w", err)
	}

	return &CommentStore{db: db, authSessionDays: authSessionDays}, nil
}

func migrateComments(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS users (
	id          INTEGER PRIMARY KEY AUTOINCREMENT,
	provider    TEXT    NOT NULL,
	provider_id TEXT    NOT NULL,
	name        TEXT    NOT NULL DEFAULT '',
	email       TEXT    NOT NULL DEFAULT '',
	avatar_url  TEXT    NOT NULL DEFAULT '',
	created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	UNIQUE(provider, provider_id)
);

CREATE TABLE IF NOT EXISTS sessions (
	token      TEXT PRIMARY KEY,
	user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	expires_at TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_expires ON sessions(expires_at);

CREATE TABLE IF NOT EXISTS comments (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	page_path  TEXT    NOT NULL,
	user_id    INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
	parent_id  INTEGER REFERENCES comments(id) ON DELETE CASCADE,
	body       TEXT    NOT NULL,
	created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	updated_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	deleted    INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_comments_page ON comments(page_path);
CREATE INDEX IF NOT EXISTS idx_comments_parent ON comments(parent_id);
`
	_, err := db.Exec(schema)
	return err
}

// --- User operations ---

// UpsertUser creates or updates a user record, returning the user.
func (s *CommentStore) UpsertUser(provider, providerID, name, email, avatarURL string) (*User, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(
		`INSERT INTO users (provider, provider_id, name, email, avatar_url, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(provider, provider_id) DO UPDATE SET
			name = excluded.name,
			email = excluded.email,
			avatar_url = excluded.avatar_url`,
		provider, providerID, name, email, avatarURL, now,
	)
	if err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}

	return s.GetUserByProvider(provider, providerID)
}

// GetUserByProvider looks up a user by provider+provider_id.
func (s *CommentStore) GetUserByProvider(provider, providerID string) (*User, error) {
	var u User
	err := s.db.QueryRow(
		`SELECT id, provider, provider_id, name, email, avatar_url, created_at
		 FROM users WHERE provider = ? AND provider_id = ?`,
		provider, providerID,
	).Scan(&u.ID, &u.Provider, &u.ProviderID, &u.Name, &u.Email, &u.AvatarURL, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &u, nil
}

// GetUserByID looks up a user by internal ID.
func (s *CommentStore) GetUserByID(id int64) (*User, error) {
	var u User
	err := s.db.QueryRow(
		`SELECT id, provider, provider_id, name, email, avatar_url, created_at
		 FROM users WHERE id = ?`,
		id,
	).Scan(&u.ID, &u.Provider, &u.ProviderID, &u.Name, &u.Email, &u.AvatarURL, &u.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	return &u, nil
}

// --- Session operations ---

// generateToken creates a cryptographically random hex token.
func generateToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// CreateSession creates a new session for a user and returns the token.
func (s *CommentStore) CreateSession(userID int64) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", fmt.Errorf("generate session token: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(time.Duration(s.authSessionDays) * 24 * time.Hour)

	_, err = s.db.Exec(
		`INSERT INTO sessions (token, user_id, created_at, expires_at) VALUES (?, ?, ?, ?)`,
		token, userID, now.Format(time.RFC3339Nano), expiresAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return "", fmt.Errorf("create session: %w", err)
	}

	return token, nil
}

// ValidateSession returns the user for a valid, non-expired session token.
// Returns nil, nil if the session is invalid or expired.
func (s *CommentStore) ValidateSession(token string) (*User, error) {
	var userID int64
	var expiresAt string
	err := s.db.QueryRow(
		`SELECT user_id, expires_at FROM sessions WHERE token = ?`,
		token,
	).Scan(&userID, &expiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("validate session: %w", err)
	}

	expires, err := time.Parse(time.RFC3339Nano, expiresAt)
	if err != nil {
		return nil, fmt.Errorf("parse expires_at: %w", err)
	}
	if time.Now().UTC().After(expires) {
		// Expired — clean up.
		_, _ = s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
		return nil, nil
	}

	return s.GetUserByID(userID)
}

// DeleteSession removes a session by token.
func (s *CommentStore) DeleteSession(token string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE token = ?`, token)
	return err
}

// CleanExpiredSessions removes all expired sessions.
func (s *CommentStore) CleanExpiredSessions() error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at < ?`, now)
	return err
}

// --- Comment operations ---

// CreateComment inserts a new comment. parentID is 0 for top-level comments.
func (s *CommentStore) CreateComment(pagePath string, userID int64, parentID int64, body string) (*Comment, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	var parent sql.NullInt64
	if parentID > 0 {
		parent = sql.NullInt64{Int64: parentID, Valid: true}
	}

	res, err := s.db.Exec(
		`INSERT INTO comments (page_path, user_id, parent_id, body, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		pagePath, userID, parent, body, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("create comment: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get comment id: %w", err)
	}

	return s.GetComment(id)
}

// GetComment returns a single comment by ID, with author info.
func (s *CommentStore) GetComment(id int64) (*Comment, error) {
	var c Comment
	var parentID sql.NullInt64
	var userName, avatarURL string

	err := s.db.QueryRow(
		`SELECT c.id, c.page_path, c.user_id, c.parent_id, c.body,
		        c.created_at, c.updated_at, c.deleted,
		        u.name, u.avatar_url
		 FROM comments c
		 JOIN users u ON c.user_id = u.id
		 WHERE c.id = ?`,
		id,
	).Scan(&c.ID, &c.PagePath, &c.UserID, &parentID, &c.Body,
		&c.CreatedAt, &c.UpdatedAt, &c.Deleted,
		&userName, &avatarURL)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get comment: %w", err)
	}

	c.ParentID = parentID
	c.Author = &CommentAuthor{Name: userName, AvatarURL: avatarURL}
	return &c, nil
}

// SoftDeleteComment marks a comment as deleted (body cleared, author anonymous).
func (s *CommentStore) SoftDeleteComment(id int64) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	_, err := s.db.Exec(
		`UPDATE comments SET deleted = 1, body = '', updated_at = ? WHERE id = ?`,
		now, id,
	)
	return err
}

// ListComments returns threaded comments for a page path.
// Soft-deleted comments are included (with anonymised data) to preserve
// thread structure, but only if they have non-deleted replies.
func (s *CommentStore) ListComments(pagePath string) ([]*Comment, error) {
	rows, err := s.db.Query(
		`SELECT c.id, c.page_path, c.user_id, c.parent_id, c.body,
		        c.created_at, c.updated_at, c.deleted,
		        u.name, u.avatar_url
		 FROM comments c
		 JOIN users u ON c.user_id = u.id
		 WHERE c.page_path = ?
		 ORDER BY c.created_at ASC`,
		pagePath,
	)
	if err != nil {
		return nil, fmt.Errorf("list comments: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var flat []*Comment
	for rows.Next() {
		var c Comment
		var parentID sql.NullInt64
		var userName, avatarURL string

		if err := rows.Scan(&c.ID, &c.PagePath, &c.UserID, &parentID, &c.Body,
			&c.CreatedAt, &c.UpdatedAt, &c.Deleted,
			&userName, &avatarURL); err != nil {
			return nil, fmt.Errorf("scan comment: %w", err)
		}

		c.ParentID = parentID
		if c.Deleted {
			c.Author = &CommentAuthor{Name: ""}
			c.Body = ""
		} else {
			c.Author = &CommentAuthor{Name: userName, AvatarURL: avatarURL}
		}

		flat = append(flat, &c)
	}

	return buildCommentTree(flat), nil
}

// buildCommentTree converts a flat, chronologically-sorted list of comments
// into a nested tree. Two passes: first build a map, then link children.
func buildCommentTree(flat []*Comment) []*Comment {
	byID := make(map[int64]*Comment, len(flat))
	for _, c := range flat {
		c.Replies = nil // ensure clean
		byID[c.ID] = c
	}

	var roots []*Comment
	for _, c := range flat {
		if c.ParentID.Valid {
			parent, ok := byID[c.ParentID.Int64]
			if ok {
				parent.Replies = append(parent.Replies, c)
				continue
			}
		}
		roots = append(roots, c)
	}

	// Prune deleted leaf comments (deleted with no replies).
	roots = pruneDeletedLeaves(roots)
	return roots
}

// pruneDeletedLeaves recursively removes deleted comments that have no replies.
func pruneDeletedLeaves(comments []*Comment) []*Comment {
	var result []*Comment
	for _, c := range comments {
		c.Replies = pruneDeletedLeaves(c.Replies)
		// Keep if not deleted, or if it has replies (preserves thread structure).
		if !c.Deleted || len(c.Replies) > 0 {
			result = append(result, c)
		}
	}
	return result
}

// Close closes the underlying database connection.
func (s *CommentStore) Close() error {
	return s.db.Close()
}
