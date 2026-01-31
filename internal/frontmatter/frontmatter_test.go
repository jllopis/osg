package frontmatter

import (
	"bytes"
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
