package ui

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"osg/internal/config"
	"osg/internal/operations"
)

// newTestLogger returns a *slog.Logger that discards all output, so tests
// don't spew log lines into the test runner.
func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// newTestConfig returns a config.Config whose filesystem directories all
// live inside a fresh t.TempDir(), and changes the test's working
// directory into that temp root so any incidental .osg/cache writes (from
// Collect/ComputeStats/LoadAISummaries) land in the temp dir rather than
// polluting the repo. Directory fields are kept as the conventional
// relative names so they resolve under the temp cwd.
func newTestConfig(t *testing.T) config.Config {
	t.Helper()
	root := t.TempDir()
	t.Chdir(root)

	mkdir := func(name string) string {
		p := filepath.Join(root, name)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", p, err)
		}
		return name
	}

	cfg := config.Default()
	cfg.ContentDir = mkdir("content")
	cfg.StaticDir = mkdir("static")
	cfg.PublicDir = mkdir("public")
	cfg.PluginsDir = mkdir("plugins")
	cfg.ThemesDir = mkdir("themes")
	cfg.Theme = "default"
	cfg.BaseURL = "https://example.test"
	cfg.DefaultLanguage = "en"
	return cfg
}

// instantRun is a KindTask body: it writes a single log line and returns
// nil immediately. Used for deterministic, fast-completing operations.
func instantRun(_ context.Context, _ map[string]any, logW io.Writer) error {
	_, _ = io.WriteString(logW, "instant run complete\n")
	return nil
}

// blockingRun is a KindService body: it writes a log line then blocks
// until the context is cancelled (i.e. until Stop / StopAll), returning
// ctx.Err(). Used to model long-running services in tests.
func blockingRun(ctx context.Context, _ map[string]any, logW io.Writer) error {
	_, _ = io.WriteString(logW, "service started\n")
	<-ctx.Done()
	return ctx.Err()
}

// newTestRunner builds an operations.Runner backed by an on-disk SQLite
// store (so History-dependent tests work) covering the canonical set of
// operations: build/deploy/check tasks and a long-running serve service.
// The store is closed via t.Cleanup.
func newTestRunner(t *testing.T) *operations.Runner {
	t.Helper()
	store, err := operations.NewStore(filepath.Join(t.TempDir(), "ops.db"))
	if err != nil {
		t.Fatalf("operations.NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })

	defs := []operations.Definition{
		{Name: "build", Kind: operations.KindTask, Description: "Build the site", Run: instantRun},
		{Name: "deploy", Kind: operations.KindTask, Description: "Deploy the site", Run: instantRun},
		{Name: "check", Kind: operations.KindTask, Description: "Validate content", Run: instantRun},
		{Name: "serve", Kind: operations.KindService, Description: "Local dev server", Addr: ":1313", Run: blockingRun},
	}
	return operations.New(defs, store)
}

// newTestServer constructs a *Server with discarded logs, an isolated
// temp-dir config, and the supplied runner. Fails the test if NewServer
// returns an error (loadTemplates must succeed).
func newTestServer(t *testing.T, runner *operations.Runner) *Server {
	t.Helper()
	srv, err := NewServer(ServerOptions{
		Version: "test",
		Cfg:     newTestConfig(t),
		Logger:  newTestLogger(),
		Runner:  runner,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return srv
}

// writeTempMarkdown writes body to dir/relpath, creating any parent
// directories, and returns the absolute path of the written file.
func writeTempMarkdown(t *testing.T, dir, relpath, body string) string {
	t.Helper()
	abs := filepath.Join(dir, filepath.FromSlash(relpath))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", abs, err)
	}
	if err := os.WriteFile(abs, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", abs, err)
	}
	return abs
}
