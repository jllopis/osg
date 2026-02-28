package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExpandTilde(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get home dir: %v", err)
	}

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"bare tilde", "~", home},
		{"tilde slash", "~/Documents/vault", filepath.Join(home, "Documents/vault")},
		{"absolute path", "/opt/vault", "/opt/vault"},
		{"relative path", "my-vault", "my-vault"},
		{"tilde in middle", "/some/~path", "/some/~path"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandTilde(tt.in)
			if got != tt.want {
				t.Errorf("ExpandTilde(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveVaultPath_ExpandsTilde(t *testing.T) {
	t.Parallel()

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("cannot get home dir: %v", err)
	}

	cfg := Config{VaultPath: "~/my-vault"}
	got, err := ResolveVaultPath(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(home, "my-vault")
	if got != want {
		t.Errorf("ResolveVaultPath got %q, want %q", got, want)
	}
}

func TestResolveVaultPath_Empty(t *testing.T) {
	t.Parallel()

	cfg := Config{}
	_, err := ResolveVaultPath(cfg)
	if err == nil {
		t.Fatal("expected error for empty vault path")
	}
}

// --- Phase 15: Language config tests ---

func TestIsMultilingual(t *testing.T) {
	t.Parallel()

	mono := Config{DefaultLanguage: "es"}
	if mono.IsMultilingual() {
		t.Fatal("expected monolingual config")
	}

	multi := Config{
		DefaultLanguage: "es",
		Languages:       []LanguageConfig{{Code: "en", Label: "English"}},
	}
	if !multi.IsMultilingual() {
		t.Fatal("expected multilingual config")
	}
}

func TestAllLanguages(t *testing.T) {
	t.Parallel()

	cfg := Config{
		DefaultLanguage: "es",
		Languages:       []LanguageConfig{{Code: "en"}, {Code: "fr"}},
	}
	langs := cfg.AllLanguages()
	if len(langs) != 3 {
		t.Fatalf("expected 3 languages, got %d", len(langs))
	}
	if langs[0] != "es" || langs[1] != "en" || langs[2] != "fr" {
		t.Fatalf("unexpected language order: %v", langs)
	}
}

func TestLanguageLabel(t *testing.T) {
	t.Parallel()

	cfg := Config{
		DefaultLanguage: "es",
		Languages:       []LanguageConfig{{Code: "en", Label: "English"}},
	}
	if got := cfg.LanguageLabel("en"); got != "English" {
		t.Fatalf("expected 'English', got %q", got)
	}
	// Default language returns just the code.
	if got := cfg.LanguageLabel("es"); got != "es" {
		t.Fatalf("expected 'es', got %q", got)
	}
	// Unknown returns the code.
	if got := cfg.LanguageLabel("de"); got != "de" {
		t.Fatalf("expected 'de', got %q", got)
	}
}

func TestLanguageValidation_EmptyCode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
vault_path: /tmp
default_language: es
languages:
  - code: ""
    label: English
`)
	_, err := Load(filepath.Join(dir, "config.yaml"))
	if err == nil {
		t.Fatal("expected error for empty language code")
	}
}

func TestLanguageValidation_DuplicatesDefault(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
vault_path: /tmp
default_language: es
languages:
  - code: es
    label: Castellano
`)
	_, err := Load(filepath.Join(dir, "config.yaml"))
	if err == nil {
		t.Fatal("expected error when language code duplicates default_language")
	}
}

func TestLanguageValidation_LabelDefaultsToCode(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
vault_path: /tmp
default_language: es
languages:
  - code: en
`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Languages[0].Label != "en" {
		t.Fatalf("expected label to default to code 'en', got %q", cfg.Languages[0].Label)
	}
}

// --- UpdatePluginsEnabled tests ---

func TestUpdatePluginsEnabled_WritesFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := UpdatePluginsEnabled(path, []string{"search", "feed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "plugins_enabled") {
		t.Error("expected plugins_enabled in output")
	}
	if !strings.Contains(content, "feed") {
		t.Error("expected 'feed' in output")
	}
	if !strings.Contains(content, "search") {
		t.Error("expected 'search' in output")
	}
}

func TestUpdatePluginsEnabled_Dedup(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := UpdatePluginsEnabled(path, []string{"search", "search", "feed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)
	// Count occurrences of "search" — should be exactly 1 in the plugins_enabled list.
	count := strings.Count(content, "- search")
	if count != 1 {
		t.Errorf("expected 1 occurrence of '- search', got %d in:\n%s", count, content)
	}
}

func TestUpdatePluginsEnabled_StripWasmExtension(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := UpdatePluginsEnabled(path, []string{"myplugin.wasm"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)
	if strings.Contains(content, ".wasm") {
		t.Error("expected .wasm extension to be stripped")
	}
	if !strings.Contains(content, "myplugin") {
		t.Error("expected 'myplugin' in output")
	}
}

func TestUpdatePluginsEnabled_SortsOutput(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := UpdatePluginsEnabled(path, []string{"zebra", "alpha", "middle"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)
	alphaIdx := strings.Index(content, "alpha")
	middleIdx := strings.Index(content, "middle")
	zebraIdx := strings.Index(content, "zebra")
	if alphaIdx > middleIdx || middleIdx > zebraIdx {
		t.Errorf("expected sorted order alpha < middle < zebra in:\n%s", content)
	}
}

func TestUpdatePluginsEnabled_SkipsEmpty(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := UpdatePluginsEnabled(path, []string{"search", "", "  ", "feed"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)
	// Count plugin entries specifically in the plugins_enabled section.
	// The section starts with "plugins_enabled:" and contains "- name" lines.
	inSection := false
	pluginCount := 0
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "plugins_enabled:" {
			inSection = true
			continue
		}
		if inSection {
			if strings.HasPrefix(trimmed, "- ") {
				pluginCount++
			} else {
				inSection = false
			}
		}
	}
	if pluginCount != 2 {
		t.Errorf("expected 2 plugins in plugins_enabled, got %d in:\n%s", pluginCount, content)
	}
}

func TestUpdatePluginsEnabled_ExistingConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	// Write initial config.
	initial := "site_title: My Site\nplugins_enabled:\n  - old\n"
	if err := os.WriteFile(path, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	err := UpdatePluginsEnabled(path, []string{"new-plugin"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "site_title") {
		t.Error("expected existing config to be preserved")
	}
	if strings.Contains(content, "old") {
		t.Error("expected old plugin to be replaced")
	}
	if !strings.Contains(content, "new-plugin") {
		t.Error("expected new-plugin in output")
	}
}

func TestUpdatePluginsEnabled_DefaultPath(t *testing.T) {
	t.Parallel()
	// Use empty string for path — should default to "config.yaml" in cwd.
	// We run in a temp dir to avoid polluting the project root.
	dir := t.TempDir()
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	err := UpdatePluginsEnabled("", []string{"search"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "config.yaml")); err != nil {
		t.Errorf("expected config.yaml to be created in cwd: %v", err)
	}
}

// --- Load tests ---

func TestLoad_MinimalConfig(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
vault_path: /tmp
site_title: Test
`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SiteTitle != "Test" {
		t.Errorf("site_title = %q, want 'Test'", cfg.SiteTitle)
	}
	if cfg.Theme != "default" {
		t.Errorf("theme = %q, want 'default'", cfg.Theme)
	}
	if cfg.ColorScheme != "auto" {
		t.Errorf("color_scheme = %q, want 'auto'", cfg.ColorScheme)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	t.Parallel()
	// Load with a path that doesn't exist — should return defaults, no error.
	cfg, err := Load("/nonexistent/config.yaml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SiteTitle != "OSG" {
		t.Errorf("expected default site_title 'OSG', got %q", cfg.SiteTitle)
	}
}

func TestLoad_EmptyPath(t *testing.T) {
	t.Parallel()
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SiteTitle != "OSG" {
		t.Errorf("expected default site_title 'OSG', got %q", cfg.SiteTitle)
	}
}

func TestLoad_InvalidColorScheme(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
vault_path: /tmp
color_scheme: neon
`)
	_, err := Load(filepath.Join(dir, "config.yaml"))
	if err == nil {
		t.Fatal("expected error for invalid color_scheme")
	}
}

func TestLoad_InvalidSummaryStrategy(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
vault_path: /tmp
summary_strategy: invalid
`)
	_, err := Load(filepath.Join(dir, "config.yaml"))
	if err == nil {
		t.Fatal("expected error for invalid summary_strategy")
	}
}

func TestLoad_InvalidAIProvider(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
vault_path: /tmp
ai:
  provider: nonexistent
`)
	_, err := Load(filepath.Join(dir, "config.yaml"))
	if err == nil {
		t.Fatal("expected error for invalid ai.provider")
	}
}

func TestLoad_AIDefaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
vault_path: /tmp
ai:
  timeout: 0
  concurrency: -1
`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.AI.Timeout != 30 {
		t.Errorf("ai.timeout = %d, want 30 (default)", cfg.AI.Timeout)
	}
	if cfg.AI.Concurrency != 3 {
		t.Errorf("ai.concurrency = %d, want 3 (default)", cfg.AI.Concurrency)
	}
}

func TestLoad_EmptyThemeDefaultsToDefault(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
vault_path: /tmp
theme: ""
`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Theme != "default" {
		t.Errorf("theme = %q, want 'default'", cfg.Theme)
	}
}

func TestLoad_DefaultLanguageFallback(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
vault_path: /tmp
default_language: ""
`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.DefaultLanguage != "es" {
		t.Errorf("default_language = %q, want 'es'", cfg.DefaultLanguage)
	}
}

// --- envKeyTransform tests ---

func TestEnvKeyTransform(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"strip prefix", "OSG_SITE_TITLE", "site_title"},
		{"nested key", "OSG_AI__PROVIDER", "ai.provider"},
		{"no prefix", "SITE_TITLE", "site_title"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := envKeyTransform(tt.in)
			if got != tt.want {
				t.Errorf("envKeyTransform(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

// --- Default tests ---

func TestDefault(t *testing.T) {
	t.Parallel()
	cfg := Default()
	if cfg.SiteTitle != "OSG" {
		t.Errorf("SiteTitle = %q, want 'OSG'", cfg.SiteTitle)
	}
	if cfg.Theme != "default" {
		t.Errorf("Theme = %q, want 'default'", cfg.Theme)
	}
	if cfg.ColorScheme != "auto" {
		t.Errorf("ColorScheme = %q, want 'auto'", cfg.ColorScheme)
	}
	if cfg.DefaultLanguage != "es" {
		t.Errorf("DefaultLanguage = %q, want 'es'", cfg.DefaultLanguage)
	}
	if cfg.AI.Provider != "gemini" {
		t.Errorf("AI.Provider = %q, want 'gemini'", cfg.AI.Provider)
	}
}

func writeYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
	if err != nil {
		t.Fatalf("write yaml: %v", err)
	}
}
