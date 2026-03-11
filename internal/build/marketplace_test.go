package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateMarketplace(t *testing.T) {
	dir := t.TempDir()
	indexPath := filepath.Join(dir, "plugins-index.json")
	if err := os.WriteFile(indexPath, []byte(`{
  "plugins": [
    {
      "name": "search",
      "description": "Full-text search",
      "author": "jllopis",
      "repo": "github.com/jllopis/osg-search",
      "version": "0.1.0",
      "hooks": ["build.finished"]
    },
    {
      "name": "mermaid",
      "description": "Mermaid diagrams",
      "author": "jllopis",
      "repo": "github.com/jllopis/osg-mermaid",
      "version": "0.1.0",
      "hooks": ["content.transform", "build.finished"]
    }
  ]
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	publicDir := filepath.Join(dir, "public")
	if err := GenerateMarketplace(indexPath, publicDir); err != nil {
		t.Fatalf("GenerateMarketplace: %v", err)
	}

	outPath := filepath.Join(publicDir, "plugins", "index.html")
	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	html := string(data)
	for _, want := range []string{
		"OSG Plugin Marketplace",
		"search",
		"mermaid",
		"Full-text search",
		"Mermaid diagrams",
		"build.finished",
		"content.transform",
		"osg plugin install github.com/jllopis/osg-search",
		"osg plugin install github.com/jllopis/osg-mermaid",
	} {
		if !strings.Contains(html, want) {
			t.Errorf("output missing %q", want)
		}
	}
}

func TestGenerateMarketplace_MissingIndex(t *testing.T) {
	err := GenerateMarketplace("/nonexistent/plugins-index.json", t.TempDir())
	if err == nil {
		t.Fatal("expected error for missing index file")
	}
}
