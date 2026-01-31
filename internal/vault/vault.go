package vault

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

func ListMarkdownFiles(root string) ([]string, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("vault path is empty")
	}

	var files []string

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		name := d.Name()
		if d.IsDir() {
			if strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if strings.HasPrefix(name, ".") {
			return nil
		}

		if strings.EqualFold(filepath.Ext(name), ".md") {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk vault: %w", err)
	}

	return files, nil
}
