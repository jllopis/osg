package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSandboxPayload_TranslatesPathsInsideSandbox(t *testing.T) {
	root := t.TempDir()
	m := &Manager{sandboxDir: root}

	got := m.sandboxPayload(map[string]any{
		"public_dir": root,
		"config": map[string]any{
			"public_dir": filepath.Join(root, "public"),
			"site_title": "t",
		},
	})
	if got["public_dir"] != "/" {
		t.Errorf("public_dir = %v, want /", got["public_dir"])
	}
	cfg := got["config"].(map[string]any)
	if cfg["public_dir"] != "/public" {
		t.Errorf("config.public_dir = %v, want /public", cfg["public_dir"])
	}
	if cfg["site_title"] != "t" {
		t.Errorf("unrelated keys must be preserved")
	}
}

func TestSandboxPayload_LeavesOutsidePathsAndNoSandboxAlone(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	m := &Manager{sandboxDir: root}

	got := m.sandboxPayload(map[string]any{
		"config": map[string]any{"public_dir": outside},
	})
	if cfg := got["config"].(map[string]any); cfg["public_dir"] != outside {
		t.Errorf("paths outside the sandbox must not be rewritten, got %v", cfg["public_dir"])
	}

	noSandbox := &Manager{}
	payload := map[string]any{"public_dir": root}
	if got := noSandbox.sandboxPayload(payload); got["public_dir"] != root {
		t.Errorf("without a sandbox the payload must pass through unchanged")
	}
}

// A plugin sandboxed to one directory must not be able to write into a
// different host directory named in the payload.
func TestEmit_PluginCannotEscapeSandbox(t *testing.T) {
	t.Parallel()
	sandbox := filepath.Join(t.TempDir(), "public")
	m, ctx := setupPluginSandbox(t, "search", sandbox)
	defer func() { _ = m.Close(ctx) }()

	outside := t.TempDir()
	_ = m.Emit(ctx, "build.finished", map[string]any{
		"config": map[string]any{"public_dir": outside},
		"site":   map[string]any{"pages": []any{}},
	})

	if _, err := os.Stat(filepath.Join(outside, "search.json")); !os.IsNotExist(err) {
		t.Fatalf("plugin escaped the sandbox: search.json exists outside (stat err = %v)", err)
	}
}
