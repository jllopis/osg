package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// LoadNode
// ---------------------------------------------------------------------------

func TestLoadNode(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		content := "site_title: Test\n# a comment\nbase_url: https://example.com\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}

		doc, err := LoadNode(path)
		if err != nil {
			t.Fatalf("LoadNode: %v", err)
		}
		if doc.Kind != yaml.DocumentNode {
			t.Fatalf("Kind = %d; want DocumentNode", doc.Kind)
		}
	})

	t.Run("nonexistent file returns empty doc", func(t *testing.T) {
		doc, err := LoadNode("/tmp/nonexistent-osg-test.yaml")
		if err != nil {
			t.Fatalf("LoadNode: %v", err)
		}
		if doc.Kind != yaml.DocumentNode {
			t.Fatalf("Kind = %d; want DocumentNode", doc.Kind)
		}
		root := rootMapping(doc)
		if root == nil || root.Kind != yaml.MappingNode {
			t.Fatal("expected mapping root")
		}
	})

	t.Run("invalid YAML returns error", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "bad.yaml")
		if err := os.WriteFile(path, []byte(":\n  :\n    - [invalid"), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := LoadNode(path)
		if err == nil {
			t.Fatal("expected error for invalid YAML")
		}
	})
}

// ---------------------------------------------------------------------------
// GetNodeValue / SetNodeValue
// ---------------------------------------------------------------------------

func TestGetSetNodeValue(t *testing.T) {
	doc := parseTestDoc(t, `
site_title: "OSG"
base_url: ""
ai:
  provider: gemini
  timeout: 30
`)

	t.Run("get top-level", func(t *testing.T) {
		val, ok := GetNodeValue(doc, "site_title")
		if !ok {
			t.Fatal("key not found")
		}
		if val != "OSG" {
			t.Errorf("val = %q; want \"OSG\"", val)
		}
	})

	t.Run("get nested", func(t *testing.T) {
		val, ok := GetNodeValue(doc, "ai.provider")
		if !ok {
			t.Fatal("key not found")
		}
		if val != "gemini" {
			t.Errorf("val = %q; want \"gemini\"", val)
		}
	})

	t.Run("get nonexistent", func(t *testing.T) {
		_, ok := GetNodeValue(doc, "nonexistent")
		if ok {
			t.Error("expected not found")
		}
	})

	t.Run("set existing top-level", func(t *testing.T) {
		SetNodeValue(doc, "site_title", "New Title")
		val, _ := GetNodeValue(doc, "site_title")
		if val != "New Title" {
			t.Errorf("val = %q; want \"New Title\"", val)
		}
	})

	t.Run("set existing nested", func(t *testing.T) {
		SetNodeValue(doc, "ai.provider", "anthropic")
		val, _ := GetNodeValue(doc, "ai.provider")
		if val != "anthropic" {
			t.Errorf("val = %q; want \"anthropic\"", val)
		}
	})

	t.Run("set new key creates it", func(t *testing.T) {
		SetNodeValue(doc, "theme", "dark")
		val, ok := GetNodeValue(doc, "theme")
		if !ok {
			t.Fatal("key not found after set")
		}
		if val != "dark" {
			t.Errorf("val = %q; want \"dark\"", val)
		}
	})

	t.Run("set new nested key creates intermediate", func(t *testing.T) {
		SetNodeValue(doc, "logging.level", "debug")
		val, ok := GetNodeValue(doc, "logging.level")
		if !ok {
			t.Fatal("key not found after set")
		}
		if val != "debug" {
			t.Errorf("val = %q; want \"debug\"", val)
		}
	})
}

// ---------------------------------------------------------------------------
// SetNodeSequence
// ---------------------------------------------------------------------------

func TestSetNodeSequence(t *testing.T) {
	doc := parseTestDoc(t, `
plugins_enabled:
  - search
`)

	t.Run("replace sequence", func(t *testing.T) {
		SetNodeSequence(doc, "plugins_enabled", []string{"feed", "search"})
		val, ok := GetNodeValue(doc, "plugins_enabled")
		if !ok {
			t.Fatal("key not found")
		}
		if val != "feed, search" {
			t.Errorf("val = %q; want \"feed, search\"", val)
		}
	})

	t.Run("create new sequence", func(t *testing.T) {
		SetNodeSequence(doc, "image_widths", []string{"640", "1200"})
		val, ok := GetNodeValue(doc, "image_widths")
		if !ok {
			t.Fatal("key not found")
		}
		if val != "640, 1200" {
			t.Errorf("val = %q; want \"640, 1200\"", val)
		}
	})
}

// ---------------------------------------------------------------------------
// DeleteNodeKey
// ---------------------------------------------------------------------------

func TestDeleteNodeKey(t *testing.T) {
	doc := parseTestDoc(t, `
site_title: "OSG"
theme: default
ai:
  provider: gemini
  timeout: 30
`)

	t.Run("delete top-level", func(t *testing.T) {
		DeleteNodeKey(doc, "theme")
		_, ok := GetNodeValue(doc, "theme")
		if ok {
			t.Error("key should be deleted")
		}
	})

	t.Run("delete nested", func(t *testing.T) {
		DeleteNodeKey(doc, "ai.timeout")
		_, ok := GetNodeValue(doc, "ai.timeout")
		if ok {
			t.Error("key should be deleted")
		}
		// Provider should still exist.
		val, ok := GetNodeValue(doc, "ai.provider")
		if !ok {
			t.Fatal("ai.provider should still exist")
		}
		if val != "gemini" {
			t.Errorf("val = %q; want \"gemini\"", val)
		}
	})

	t.Run("delete nonexistent is no-op", func(t *testing.T) {
		DeleteNodeKey(doc, "nonexistent")
		// Should not panic.
	})
}

// ---------------------------------------------------------------------------
// Comment preservation (round-trip)
// ---------------------------------------------------------------------------

func TestCommentPreservation(t *testing.T) {
	input := `# Site config
site_title: "OSG" # inline comment
# This is the base URL
base_url: "https://example.com"
`
	doc := parseTestDoc(t, input)

	// Modify a value.
	SetNodeValue(doc, "site_title", "New Title")

	// Round-trip through marshal.
	out, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	output := string(out)

	// The inline comment should be preserved.
	if !strings.Contains(output, "# inline comment") {
		t.Errorf("inline comment lost:\n%s", output)
	}
	// The head comment should be preserved.
	if !strings.Contains(output, "# This is the base URL") {
		t.Errorf("head comment lost:\n%s", output)
	}
	// The top comment should be preserved.
	if !strings.Contains(output, "# Site config") {
		t.Errorf("top comment lost:\n%s", output)
	}
	// The new value should be there.
	if !strings.Contains(output, "New Title") {
		t.Errorf("new value not present:\n%s", output)
	}
}

// ---------------------------------------------------------------------------
// SaveNode round-trip
// ---------------------------------------------------------------------------

func TestSaveNodeRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	input := "site_title: OSG\nbase_url: \"\"\n"
	if err := os.WriteFile(path, []byte(input), 0o644); err != nil {
		t.Fatal(err)
	}

	doc, err := LoadNode(path)
	if err != nil {
		t.Fatalf("LoadNode: %v", err)
	}

	SetNodeValue(doc, "site_title", "Changed")
	if err := SaveNode(path, doc); err != nil {
		t.Fatalf("SaveNode: %v", err)
	}

	// Re-read and verify.
	doc2, err := LoadNode(path)
	if err != nil {
		t.Fatalf("LoadNode after save: %v", err)
	}
	val, ok := GetNodeValue(doc2, "site_title")
	if !ok || val != "Changed" {
		t.Errorf("site_title = %q; want \"Changed\"", val)
	}
}

// ---------------------------------------------------------------------------
// guessScalarTag
// ---------------------------------------------------------------------------

func TestGuessScalarTag(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{"true", "!!bool"},
		{"false", "!!bool"},
		{"42", "!!int"},
		{"3.14", "!!float"},
		{"hello", "!!str"},
		{"", "!!str"},
	}
	for _, tc := range tests {
		got := guessScalarTag(tc.value)
		if got != tc.want {
			t.Errorf("guessScalarTag(%q) = %q; want %q", tc.value, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func parseTestDoc(t *testing.T, content string) *yaml.Node {
	t.Helper()
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatalf("parse test doc: %v", err)
	}
	return &doc
}
