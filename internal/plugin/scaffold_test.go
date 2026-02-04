package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldRust(t *testing.T) {
	t.Parallel()

	base := t.TempDir()
	if err := ScaffoldRust(base, "My Plugin"); err != nil {
		t.Fatalf("scaffold failed: %v", err)
	}

	root := filepath.Join(base, "My Plugin")
	files := []string{
		filepath.Join(root, "Cargo.toml"),
		filepath.Join(root, "README.md"),
		filepath.Join(root, "src", "lib.rs"),
		filepath.Join(root, "build.sh"),
	}

	for _, file := range files {
		if _, err := os.Stat(file); err != nil {
			t.Fatalf("expected file %s: %v", file, err)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read file %s: %v", file, err)
		}
		content := string(data)
		if strings.Contains(content, "{{name}}") || strings.Contains(content, "{{crate_name}}") {
			t.Fatalf("placeholders not replaced in %s", file)
		}
	}
}
