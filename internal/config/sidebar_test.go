package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSidebarWidgetsNormalisation(t *testing.T) {
	cases := []struct {
		name    string
		yaml    string
		want    []string
		wantErr bool
	}{
		{"empty", "", nil, false},
		{
			"trim and lowercase",
			"sidebar_widgets:\n  - \" Author \"\n  - NEWSLETTER\n",
			[]string{"author", "newsletter"},
			false,
		},
		{
			"dedupe",
			"sidebar_widgets:\n  - author\n  - Author\n  - author\n",
			[]string{"author"},
			false,
		},
		{
			"order preserved",
			"sidebar_widgets:\n  - popular\n  - author\n  - newsletter\n",
			[]string{"popular", "author", "newsletter"},
			false,
		},
		{
			"unknown rejected",
			"sidebar_widgets:\n  - author\n  - twitter-feed\n",
			nil,
			true,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(path, []byte(c.yaml), 0o644); err != nil {
				t.Fatal(err)
			}
			cfg, err := Load(path)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !equalStrings(cfg.SidebarWidgets, c.want) {
				t.Errorf("got %v want %v", cfg.SidebarWidgets, c.want)
			}
		})
	}
}

func TestNewsletterActionTrimmed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	_ = os.WriteFile(path, []byte("newsletter_action: \"  https://example.com/sub  \"\n"), 0o644)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.NewsletterAction != "https://example.com/sub" {
		t.Errorf("not trimmed: %q", cfg.NewsletterAction)
	}
	if strings.Contains(cfg.NewsletterAction, " ") {
		t.Errorf("contains spaces: %q", cfg.NewsletterAction)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
