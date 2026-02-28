package config

import (
	"os"
	"path/filepath"
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

func writeYAML(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644)
	if err != nil {
		t.Fatalf("write yaml: %v", err)
	}
}
