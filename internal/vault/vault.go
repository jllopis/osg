package vault

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

// imageExtensions lists file extensions recognized as images.
var imageExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".gif":  true,
	".svg":  true,
	".webp": true,
	".bmp":  true,
	".avif": true,
}

// ImageIndex maps image filenames to their full filesystem paths.
// When multiple files share the same basename, the first one found wins
// (Walk order is lexicographic within each directory).
type ImageIndex map[string]string

// Resolve looks up an image reference (as used in Obsidian wikilinks or
// osg.image frontmatter). It tries the full reference first (which may
// include a relative path like "folder/image.png"), then falls back to
// just the basename.
func (idx ImageIndex) Resolve(ref string) (fullPath string, ok bool) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", false
	}
	// Try exact match (ref could be "Attachments/photo.png")
	if p, found := idx[ref]; found {
		return p, true
	}
	// Try basename only
	base := filepath.Base(ref)
	if p, found := idx[base]; found {
		return p, true
	}
	return "", false
}

// BuildImageIndex walks the vault directory and builds an index of all
// image files. Keys are basenames; if a vault-relative path is unique it
// is also stored. Dot-prefixed directories are skipped.
func BuildImageIndex(root string) (ImageIndex, error) {
	if strings.TrimSpace(root) == "" {
		return nil, fmt.Errorf("vault path is empty")
	}

	idx := ImageIndex{}

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

		ext := strings.ToLower(filepath.Ext(name))
		if !imageExtensions[ext] {
			return nil
		}

		// Store by basename (first wins)
		if _, exists := idx[name]; !exists {
			idx[name] = path
		}

		// Also store by vault-relative path for explicit references
		rel, relErr := filepath.Rel(root, path)
		if relErr == nil {
			rel = filepath.ToSlash(rel)
			if _, exists := idx[rel]; !exists {
				idx[rel] = path
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("walk vault for images: %w", err)
	}

	return idx, nil
}

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
