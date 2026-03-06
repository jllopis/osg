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

	// Parse the YAML node and check the plugins_enabled values directly
	// (avoid scanning the whole file, which includes comments with ".wasm").
	doc, err := LoadNode(path)
	if err != nil {
		t.Fatalf("load node: %v", err)
	}
	val, ok := GetNodeValue(doc, "plugins_enabled")
	if !ok {
		t.Fatal("plugins_enabled key not found in YAML")
	}
	if val != "myplugin" {
		t.Errorf("expected plugins_enabled to contain 'myplugin', got %q", val)
	}
	if strings.Contains(val, ".wasm") {
		t.Error("expected .wasm extension to be stripped from value")
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

// --- Interactions config tests ---

func TestDefault_InteractionsDefaults(t *testing.T) {
	t.Parallel()
	cfg := Default()
	if cfg.Interactions.Enabled {
		t.Error("Interactions.Enabled should default to false")
	}
	if cfg.Interactions.Listen != ":8090" {
		t.Errorf("Interactions.Listen = %q, want ':8090'", cfg.Interactions.Listen)
	}
	if cfg.Interactions.DBPath != ".osg/interactions.db" {
		t.Errorf("Interactions.DBPath = %q, want '.osg/interactions.db'", cfg.Interactions.DBPath)
	}
	if cfg.Interactions.ViewDedupHours != 24 {
		t.Errorf("Interactions.ViewDedupHours = %d, want 24", cfg.Interactions.ViewDedupHours)
	}
}

func TestLoad_InteractionsFromYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
interactions:
  enabled: true
  api_url: "https://api.example.com"
  listen: ":9090"
  db_path: "/tmp/test.db"
  cors_origins:
    - "https://example.com"
    - "https://other.com"
  view_dedup_hours: 48
`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !cfg.Interactions.Enabled {
		t.Error("expected Interactions.Enabled = true")
	}
	if cfg.Interactions.APIURL != "https://api.example.com" {
		t.Errorf("APIURL = %q", cfg.Interactions.APIURL)
	}
	if cfg.Interactions.Listen != ":9090" {
		t.Errorf("Listen = %q", cfg.Interactions.Listen)
	}
	if cfg.Interactions.DBPath != "/tmp/test.db" {
		t.Errorf("DBPath = %q", cfg.Interactions.DBPath)
	}
	if len(cfg.Interactions.CORSOrigins) != 2 {
		t.Errorf("CORSOrigins len = %d, want 2", len(cfg.Interactions.CORSOrigins))
	}
	if cfg.Interactions.ViewDedupHours != 48 {
		t.Errorf("ViewDedupHours = %d, want 48", cfg.Interactions.ViewDedupHours)
	}
}

func TestLoad_InteractionsNormalization(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
interactions:
  enabled: true
  listen: ""
  db_path: ""
  view_dedup_hours: -5
`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// Empty listen should default to :8090.
	if cfg.Interactions.Listen != ":8090" {
		t.Errorf("Listen = %q, want ':8090'", cfg.Interactions.Listen)
	}
	// Empty db_path should default.
	if cfg.Interactions.DBPath != ".osg/interactions.db" {
		t.Errorf("DBPath = %q, want '.osg/interactions.db'", cfg.Interactions.DBPath)
	}
	// Negative dedup hours should default to 24.
	if cfg.Interactions.ViewDedupHours != 24 {
		t.Errorf("ViewDedupHours = %d, want 24", cfg.Interactions.ViewDedupHours)
	}
}

// --- Sharing config tests ---

func TestDefault_SharingEnabled(t *testing.T) {
	t.Parallel()
	cfg := Default()
	if !cfg.Sharing {
		t.Error("Sharing should default to true")
	}
}

func TestLoad_SharingDisabled(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `sharing: false`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Sharing {
		t.Error("expected Sharing = false after explicit disable")
	}
}

// --- Comments config tests ---

func TestDefault_CommentsDefaults(t *testing.T) {
	t.Parallel()
	cfg := Default()
	c := cfg.Interactions.Comments

	if c.Enabled {
		t.Error("Comments should default to disabled")
	}
	if c.DBPath != ".osg/comments.db" {
		t.Errorf("DBPath = %q, want .osg/comments.db", c.DBPath)
	}
	if c.AuthSessionDays != 30 {
		t.Errorf("AuthSessionDays = %d, want 30", c.AuthSessionDays)
	}
}

func TestLoad_CommentsFromYAML(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
interactions:
  enabled: true
  comments:
    enabled: true
    db_path: "data/my-comments.db"
    auth_session_days: 7
    auth_callback_url: "https://blog.example.com"
    providers:
      - provider: github
        client_id: "gh-id-123"
        client_secret: "gh-secret-456"
      - provider: google
        client_id: "g-id-789"
        client_secret: "g-secret-012"
`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	c := cfg.Interactions.Comments
	if !c.Enabled {
		t.Error("expected comments enabled")
	}
	if c.DBPath != "data/my-comments.db" {
		t.Errorf("DBPath = %q", c.DBPath)
	}
	if c.AuthSessionDays != 7 {
		t.Errorf("AuthSessionDays = %d, want 7", c.AuthSessionDays)
	}
	if c.AuthCallbackURL != "https://blog.example.com" {
		t.Errorf("AuthCallbackURL = %q", c.AuthCallbackURL)
	}
	if len(c.Providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(c.Providers))
	}
	if c.Providers[0].Provider != "github" || c.Providers[0].ClientID != "gh-id-123" {
		t.Errorf("provider[0] = %+v", c.Providers[0])
	}
	if c.Providers[1].Provider != "google" || c.Providers[1].ClientID != "g-id-789" {
		t.Errorf("provider[1] = %+v", c.Providers[1])
	}
}

func TestLoad_CommentsNormalization(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
interactions:
  enabled: true
  comments:
    enabled: true
    db_path: ""
    auth_session_days: -1
    providers:
      - provider: "  GITHUB  "
        client_id: "  id  "
        client_secret: "  secret  "
`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	c := cfg.Interactions.Comments

	// Empty db_path should default.
	if c.DBPath != ".osg/comments.db" {
		t.Errorf("DBPath = %q, want .osg/comments.db", c.DBPath)
	}
	// Negative session days should default to 30.
	if c.AuthSessionDays != 30 {
		t.Errorf("AuthSessionDays = %d, want 30", c.AuthSessionDays)
	}
	// Provider name should be lowercased and trimmed.
	if c.Providers[0].Provider != "github" {
		t.Errorf("provider = %q, want github", c.Providers[0].Provider)
	}
	// Client ID/secret should be trimmed.
	if c.Providers[0].ClientID != "id" {
		t.Errorf("client_id = %q, want id", c.Providers[0].ClientID)
	}
	if c.Providers[0].ClientSecret != "secret" {
		t.Errorf("client_secret = %q, want secret", c.Providers[0].ClientSecret)
	}
}

func TestLoad_CommentsInvalidProvider(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
interactions:
  enabled: true
  comments:
    enabled: true
    providers:
      - provider: twitter
        client_id: id
        client_secret: secret
`)
	_, err := Load(filepath.Join(dir, "config.yaml"))
	if err == nil {
		t.Fatal("expected error for invalid provider")
	}
	if !strings.Contains(err.Error(), "must be github or google") {
		t.Errorf("error = %q, want 'must be github or google'", err.Error())
	}
}

func TestLoad_CommentsMissingClientID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
interactions:
  enabled: true
  comments:
    enabled: true
    providers:
      - provider: github
        client_id: ""
        client_secret: secret
`)
	_, err := Load(filepath.Join(dir, "config.yaml"))
	if err == nil {
		t.Fatal("expected error for missing client_id")
	}
	if !strings.Contains(err.Error(), "client_id is required") {
		t.Errorf("error = %q, want 'client_id is required'", err.Error())
	}
}

func TestLoad_CommentsMissingClientSecret(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `
interactions:
  enabled: true
  comments:
    enabled: true
    providers:
      - provider: github
        client_id: id
        client_secret: ""
`)
	_, err := Load(filepath.Join(dir, "config.yaml"))
	if err == nil {
		t.Fatal("expected error for missing client_secret")
	}
	if !strings.Contains(err.Error(), "client_secret is required") {
		t.Errorf("error = %q, want 'client_secret is required'", err.Error())
	}
}

// --- TUI log modifier config tests ---

func TestDefault_TUILogModifier(t *testing.T) {
	t.Parallel()
	cfg := Default()
	if cfg.TUILogModifier != "shift" {
		t.Errorf("TUILogModifier = %q, want 'shift'", cfg.TUILogModifier)
	}
}

func TestLoad_TUILogModifierAlt(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `tui_log_modifier: alt`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TUILogModifier != "alt" {
		t.Errorf("TUILogModifier = %q, want 'alt'", cfg.TUILogModifier)
	}
}

func TestLoad_TUILogModifierShift(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `tui_log_modifier: shift`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TUILogModifier != "shift" {
		t.Errorf("TUILogModifier = %q, want 'shift'", cfg.TUILogModifier)
	}
}

func TestLoad_TUILogModifierEmptyDefaultsToShift(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `tui_log_modifier: ""`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TUILogModifier != "shift" {
		t.Errorf("TUILogModifier = %q, want 'shift' (default for empty)", cfg.TUILogModifier)
	}
}

func TestLoad_TUILogModifierInvalid(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `tui_log_modifier: ctrl`)
	_, err := Load(filepath.Join(dir, "config.yaml"))
	if err == nil {
		t.Fatal("expected error for invalid tui_log_modifier")
	}
	if !strings.Contains(err.Error(), "must be alt or shift") {
		t.Errorf("error = %q, want 'must be alt or shift'", err.Error())
	}
}

func TestLoad_TUILogModifierNormalizesCase(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeYAML(t, dir, "config.yaml", `tui_log_modifier: "  SHIFT  "`)
	cfg, err := Load(filepath.Join(dir, "config.yaml"))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.TUILogModifier != "shift" {
		t.Errorf("TUILogModifier = %q, want 'shift' (normalized)", cfg.TUILogModifier)
	}
}
