package publish

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func writeVaultFile(t *testing.T, dir, rel, content string) string {
	t.Helper()
	full := filepath.Join(dir, rel)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return full
}

func TestPromoteDueDrafts_FlipsDueDraft(t *testing.T) {
	vault := t.TempDir()
	path := writeVaultFile(t, vault, "posts/due-draft.md",
		`---
title: My post
osg:
  publish: draft
  publish_at: 2024-01-01
---

Body content.
`)

	promoted, err := PromoteDueDrafts(vault, time.Now(), nil)
	if err != nil {
		t.Fatalf("PromoteDueDrafts: %v", err)
	}
	if len(promoted) != 1 || promoted[0] != "posts/due-draft.md" {
		t.Errorf("unexpected promoted list: %v", promoted)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	if !strings.Contains(s, "publish: true") {
		t.Errorf("expected publish: true in output, got:\n%s", s)
	}
	if strings.Contains(s, "publish: draft") {
		t.Errorf("draft flag should have been replaced, got:\n%s", s)
	}
	// Body must survive intact (atomic write must not damage it).
	if !strings.Contains(s, "Body content.") {
		t.Errorf("body lost during rewrite, got:\n%s", s)
	}
}

func TestPromoteDueDrafts_LeavesFutureDraftAlone(t *testing.T) {
	vault := t.TempDir()
	path := writeVaultFile(t, vault, "posts/future.md",
		`---
title: Tomorrow
osg:
  publish: draft
  publish_at: 2099-01-01
---

Body.
`)

	promoted, err := PromoteDueDrafts(vault, time.Now(), nil)
	if err != nil {
		t.Fatalf("PromoteDueDrafts: %v", err)
	}
	if len(promoted) != 0 {
		t.Errorf("future draft must not be promoted; got %v", promoted)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "publish: draft") {
		t.Errorf("future draft was modified; content:\n%s", got)
	}
}

func TestPromoteDueDrafts_LeavesDraftWithoutPublishAt(t *testing.T) {
	vault := t.TempDir()
	path := writeVaultFile(t, vault, "posts/just-draft.md",
		`---
title: Just a draft
osg:
  publish: draft
---

Body.
`)

	promoted, err := PromoteDueDrafts(vault, time.Now(), nil)
	if err != nil {
		t.Fatalf("PromoteDueDrafts: %v", err)
	}
	if len(promoted) != 0 {
		t.Errorf("plain draft (no publish_at) must not be promoted; got %v", promoted)
	}
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "publish: draft") {
		t.Errorf("plain draft was modified; content:\n%s", got)
	}
}

func TestPromoteDueDrafts_LeavesNonDraftAlone(t *testing.T) {
	vault := t.TempDir()
	path := writeVaultFile(t, vault, "posts/published.md",
		`---
title: Live
osg:
  publish: true
  publish_at: 2024-01-01
---

Body.
`)
	original, _ := os.ReadFile(path)

	promoted, err := PromoteDueDrafts(vault, time.Now(), nil)
	if err != nil {
		t.Fatalf("PromoteDueDrafts: %v", err)
	}
	if len(promoted) != 0 {
		t.Errorf("non-draft must not be promoted; got %v", promoted)
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(original) {
		t.Errorf("non-draft file was modified")
	}
}

func TestPromoteDueDrafts_EmptyVaultIsNoop(t *testing.T) {
	promoted, err := PromoteDueDrafts("", time.Now(), nil)
	if err != nil {
		t.Errorf("empty vault path should be a no-op, got error: %v", err)
	}
	if len(promoted) != 0 {
		t.Errorf("expected empty result, got %v", promoted)
	}
}

func TestPromoteDueDrafts_MissingVaultIsNoop(t *testing.T) {
	promoted, err := PromoteDueDrafts(filepath.Join(t.TempDir(), "no-such-dir"), time.Now(), nil)
	if err != nil {
		t.Errorf("missing vault path should be a no-op, got error: %v", err)
	}
	if len(promoted) != 0 {
		t.Errorf("expected empty result, got %v", promoted)
	}
}

func TestPromoteDueDrafts_SkipsDotDirs(t *testing.T) {
	vault := t.TempDir()
	// Should be skipped (dot-dir).
	writeVaultFile(t, vault, ".obsidian/template.md",
		`---
osg:
  publish: draft
  publish_at: 2024-01-01
---
template body
`)
	// Should be promoted (visible dir).
	writeVaultFile(t, vault, "posts/visible.md",
		`---
osg:
  publish: draft
  publish_at: 2024-01-01
---
real body
`)
	promoted, err := PromoteDueDrafts(vault, time.Now(), nil)
	if err != nil {
		t.Fatalf("PromoteDueDrafts: %v", err)
	}
	if len(promoted) != 1 || promoted[0] != "posts/visible.md" {
		t.Errorf("dot-dir contents should be skipped; promoted = %v", promoted)
	}
}
