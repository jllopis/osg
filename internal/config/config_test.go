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
