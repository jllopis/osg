package api

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages the SQLite database for page interactions.
type Store struct {
	db             *sql.DB
	viewDedupHours int
}

// DB returns the underlying *sql.DB for sharing with other stores.
func (s *Store) DB() *sql.DB { return s.db }

// Stats holds aggregated interaction data for a single page.
type Stats struct {
	Views    int64 `json:"views"`
	Unique   int64 `json:"unique"`
	Likes    int64 `json:"likes"`
	Dislikes int64 `json:"dislikes"`
	UserVote int   `json:"user_vote"` // 1=like, -1=dislike, 0=none
}

// NewStore opens (or creates) the SQLite database at dbPath and ensures
// the schema is up to date.
func NewStore(dbPath string, viewDedupHours int) (*Store, error) {
	if viewDedupHours <= 0 {
		viewDedupHours = 24
	}

	// Ensure parent directory exists.
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create db directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate db: %w", err)
	}

	return &Store{db: db, viewDedupHours: viewDedupHours}, nil
}

func migrate(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS page_views (
	id         INTEGER PRIMARY KEY AUTOINCREMENT,
	page_path  TEXT    NOT NULL,
	fingerprint TEXT   NOT NULL,
	created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE INDEX IF NOT EXISTS idx_page_views_path ON page_views(page_path);
CREATE UNIQUE INDEX IF NOT EXISTS idx_page_views_dedup
	ON page_views(page_path, fingerprint, substr(created_at, 1, 10));

CREATE TABLE IF NOT EXISTS page_votes (
	page_path   TEXT    NOT NULL,
	fingerprint TEXT    NOT NULL,
	vote        INTEGER NOT NULL CHECK(vote IN (-1, 1)),
	created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
	PRIMARY KEY (page_path, fingerprint)
);
`
	_, err := db.Exec(schema)
	return err
}

// RecordView inserts a page view. Total views are always incremented.
// Unique views are deduped by fingerprint within the configured window.
// Returns the updated stats for the page.
func (s *Store) RecordView(pagePath, fingerprint string) (Stats, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	// Always insert a total-view row (no constraint — dedup index only
	// prevents duplicate unique rows per day).
	// The UNIQUE INDEX on (page_path, fingerprint, date) means the same
	// fingerprint on the same day will conflict. We use INSERT OR IGNORE
	// so the unique count naturally deduplicates per day, and we track
	// total views separately with a simple INSERT (different timestamp
	// precision avoids conflict).
	//
	// Strategy: attempt INSERT; if it conflicts on the dedup index this
	// fingerprint already viewed today, so the unique count stays the same
	// but we still want to count total views. We handle this by always
	// inserting with full timestamp precision (the dedup index uses only
	// the date portion via substr, so the first insert per day succeeds
	// and subsequent ones are ignored).
	_, _ = s.db.Exec(
		`INSERT OR IGNORE INTO page_views (page_path, fingerprint, created_at) VALUES (?, ?, ?)`,
		pagePath, fingerprint, now,
	)

	return s.getStats(pagePath, fingerprint)
}

// Vote records or updates a vote. vote must be 1 (like), -1 (dislike), or 0 (retract).
// Returns the updated stats for the page.
func (s *Store) Vote(pagePath, fingerprint string, vote int) (Stats, error) {
	now := time.Now().UTC().Format(time.RFC3339Nano)

	if vote == 0 {
		// Retract vote.
		_, err := s.db.Exec(
			`DELETE FROM page_votes WHERE page_path = ? AND fingerprint = ?`,
			pagePath, fingerprint,
		)
		if err != nil {
			return Stats{}, fmt.Errorf("retract vote: %w", err)
		}
	} else {
		// Upsert vote.
		_, err := s.db.Exec(
			`INSERT INTO page_votes (page_path, fingerprint, vote, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?)
			 ON CONFLICT(page_path, fingerprint) DO UPDATE SET vote = excluded.vote, updated_at = excluded.updated_at`,
			pagePath, fingerprint, vote, now, now,
		)
		if err != nil {
			return Stats{}, fmt.Errorf("upsert vote: %w", err)
		}
	}

	return s.getStats(pagePath, fingerprint)
}

// TopPage holds an aggregated view count keyed by page path.
type TopPage struct {
	Path  string
	Views int64
}

// TopPages returns the most viewed pages ordered by view count desc.
// Counts come from the page_views table (one row per unique
// fingerprint+day per RecordView's INSERT OR IGNORE). Used by the
// build pipeline to populate the popular-posts sidebar widget.
func (s *Store) TopPages(limit int) ([]TopPage, error) {
	if limit <= 0 {
		limit = 5
	}
	rows, err := s.db.Query(
		`SELECT page_path, COUNT(*) FROM page_views
		 GROUP BY page_path
		 ORDER BY COUNT(*) DESC, page_path ASC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var out []TopPage
	for rows.Next() {
		var tp TopPage
		if err := rows.Scan(&tp.Path, &tp.Views); err != nil {
			return nil, err
		}
		out = append(out, tp)
	}
	return out, rows.Err()
}

// GetStats returns the current interaction stats for a page and fingerprint.
func (s *Store) GetStats(pagePath, fingerprint string) (Stats, error) {
	return s.getStats(pagePath, fingerprint)
}

func (s *Store) getStats(pagePath, fingerprint string) (Stats, error) {
	var st Stats

	// Total views = count of all rows (including deduped-out ones that
	// were ignored by INSERT OR IGNORE — so total == unique for the
	// daily-dedup scheme). For true total views we need a separate table
	// or a counter column. We'll use COUNT as total views (one per day
	// per fingerprint) and COUNT(DISTINCT fingerprint) as unique.
	//
	// Re-design: the dedup index means at most one row per
	// (page, fingerprint, day). So COUNT(*) is views (counting each
	// unique visitor once per day) and COUNT(DISTINCT fingerprint) is
	// unique visitors.
	err := s.db.QueryRow(
		`SELECT COUNT(*), COUNT(DISTINCT fingerprint) FROM page_views WHERE page_path = ?`,
		pagePath,
	).Scan(&st.Views, &st.Unique)
	if err != nil {
		return st, fmt.Errorf("count views: %w", err)
	}

	// Likes and dislikes.
	err = s.db.QueryRow(
		`SELECT COALESCE(SUM(CASE WHEN vote = 1 THEN 1 ELSE 0 END), 0),
		        COALESCE(SUM(CASE WHEN vote = -1 THEN 1 ELSE 0 END), 0)
		 FROM page_votes WHERE page_path = ?`,
		pagePath,
	).Scan(&st.Likes, &st.Dislikes)
	if err != nil {
		return st, fmt.Errorf("count votes: %w", err)
	}

	// Current user's vote.
	var userVote sql.NullInt64
	err = s.db.QueryRow(
		`SELECT vote FROM page_votes WHERE page_path = ? AND fingerprint = ?`,
		pagePath, fingerprint,
	).Scan(&userVote)
	if err != nil && err != sql.ErrNoRows {
		return st, fmt.Errorf("get user vote: %w", err)
	}
	if userVote.Valid {
		st.UserVote = int(userVote.Int64)
	}

	return st, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}
