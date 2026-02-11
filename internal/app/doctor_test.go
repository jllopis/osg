package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDoctorDevAllowsEmptyBaseURL(t *testing.T) {
	t.Parallel()

	cfgPath := writeDoctorConfig(t, "dev", "")
	if err := RunDoctor(nil, CLIOptions{ConfigPath: cfgPath}); err != nil {
		t.Fatalf("doctor dev should not error: %v", err)
	}
}

func TestDoctorProdRequiresBaseURL(t *testing.T) {
	t.Parallel()

	cfgPath := writeDoctorConfig(t, "prod", "")
	if err := RunDoctor(nil, CLIOptions{ConfigPath: cfgPath}); err == nil {
		t.Fatalf("doctor prod should error on empty base_url")
	}
}

func writeDoctorConfig(t *testing.T, profile string, baseURL string) string {
	t.Helper()

	root := t.TempDir()
	vault := filepath.Join(root, "vault")
	content := filepath.Join(root, "content")
	public := filepath.Join(root, "public")
	templates := filepath.Join(root, "templates")
	staticDir := filepath.Join(root, "static")
	themes := filepath.Join(root, "themes", "default", "templates")

	for _, dir := range []string{vault, content, public, templates, staticDir, themes} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}

	config := []byte(
		`vault_path: "` + vault + `"
base_url: "` + baseURL + `"
theme: default
content_dir: "` + content + `"
public_dir: "` + public + `"
templates_dir: "` + templates + `"
static_dir: "` + staticDir + `"
themes_dir: "` + filepath.Join(root, "themes") + `"
plugins_dir: "` + filepath.Join(root, "plugins") + `"
plugins_enabled: []
sass_dir: "` + filepath.Join(root, "sass") + `"
content_layout: "{date}/{slug}"
include_drafts: false
compile_sass: false
tui_prefix: space
tui_prefix_ms: 600
serve_watch: false
serve_live_reload: false
serve_debounce_ms: 300
build_incremental: true
build_cache_dir: "` + filepath.Join(root, ".osg/cache") + `"
clean_public: true
doctor_profile: ` + profile + `
logging:
  level: info
  format: json
`)

	cfgPath := filepath.Join(root, "config.yaml")
	if err := os.WriteFile(cfgPath, config, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return cfgPath
}
