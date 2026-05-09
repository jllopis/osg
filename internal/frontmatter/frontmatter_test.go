package frontmatter

import (
	"bytes"
	"strings"
	"testing"
)

func TestSplitFrontmatterWithYAML(t *testing.T) {
	input := []byte("---\nfoo: bar\npublish: true\n---\nBody line 1\nBody line 2\n")

	fm, body, hasFM, err := SplitFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !hasFM {
		t.Fatalf("expected frontmatter")
	}
	if fm["foo"] != "bar" {
		t.Fatalf("expected foo=bar, got %v", fm["foo"])
	}
	if fm["publish"] != true {
		t.Fatalf("expected publish=true, got %v", fm["publish"])
	}

	expectedBody := []byte("Body line 1\nBody line 2\n")
	if !bytes.Equal(body, expectedBody) {
		t.Fatalf("body mismatch: %q", string(body))
	}
}

func TestSplitFrontmatterNoHeader(t *testing.T) {
	input := []byte("No frontmatter here\nSecond line\n")

	fm, body, hasFM, err := SplitFrontmatter(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hasFM {
		t.Fatalf("did not expect frontmatter")
	}
	if len(fm) != 0 {
		t.Fatalf("expected empty frontmatter")
	}
	if !bytes.Equal(body, input) {
		t.Fatalf("body mismatch")
	}
}

func TestSplitFrontmatterMissingEnd(t *testing.T) {
	input := []byte("---\nfoo: bar\n")

	_, _, _, err := SplitFrontmatter(input)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestUpdateField_AddsNewOSGSummary(t *testing.T) {
	src := `---
title: Test post
date: 2026-01-01
osg:
  publish: true
---
Body content.
`
	out, err := UpdateField([]byte(src), "osg.summary", "Hand-written summary.")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "summary: Hand-written summary.") {
		t.Errorf("expected new osg.summary key, got:\n%s", s)
	}
	if !strings.Contains(s, "publish: true") {
		t.Errorf("existing osg.publish must be preserved, got:\n%s", s)
	}
	if !strings.Contains(s, "title: Test post") {
		t.Errorf("title preserved, got:\n%s", s)
	}
	if !strings.HasSuffix(s, "Body content.\n") {
		t.Errorf("body preserved, got:\n%s", s)
	}
}

func TestUpdateField_ReplacesExisting(t *testing.T) {
	src := `---
title: T
osg:
  summary: old summary
---
body
`
	out, err := UpdateField([]byte(src), "osg.summary", "new summary")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "old summary") {
		t.Errorf("old summary should have been replaced, got:\n%s", s)
	}
	if !strings.Contains(s, "new summary") {
		t.Errorf("new summary missing, got:\n%s", s)
	}
}

func TestUpdateField_DeleteOnEmpty(t *testing.T) {
	src := `---
title: T
osg:
  summary: drop me
  publish: true
---
body
`
	out, err := UpdateField([]byte(src), "osg.summary", "")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if strings.Contains(s, "drop me") {
		t.Errorf("summary should have been deleted, got:\n%s", s)
	}
	if !strings.Contains(s, "publish: true") {
		t.Errorf("sibling osg.publish must survive, got:\n%s", s)
	}
}

func TestUpdateField_CreatesMappingIfMissing(t *testing.T) {
	src := `---
title: T
---
body
`
	out, err := UpdateField([]byte(src), "osg.summary", "added")
	if err != nil {
		t.Fatal(err)
	}
	s := string(out)
	if !strings.Contains(s, "osg:") || !strings.Contains(s, "summary: added") {
		t.Errorf("expected osg.summary block to be created, got:\n%s", s)
	}
}
