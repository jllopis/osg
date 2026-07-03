package operations

import (
	"path/filepath"
	"testing"
)

// TestStoreDSNPragmasApplied guards against the modernc/sqlite DSN regression:
// the driver ignores mattn-style keys (_journal_mode, _busy_timeout) and only
// honours the _pragma= form. If the DSN reverts, WAL and the busy timeout
// silently stop applying and concurrent writers hit "database is locked".
func TestStoreDSNPragmasApplied(t *testing.T) {
	t.Parallel()
	st, err := NewStore(filepath.Join(t.TempDir(), "ops.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	defer func() { _ = st.Close() }()

	var journalMode string
	if err := st.db.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if journalMode != "wal" {
		t.Errorf("journal_mode = %q, want wal", journalMode)
	}

	var busyTimeout int
	if err := st.db.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if busyTimeout != 5000 {
		t.Errorf("busy_timeout = %d, want 5000", busyTimeout)
	}
}
