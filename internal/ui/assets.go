package ui

import (
	"io/fs"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"osg/internal/config"
)

// AssetEntry is the template-friendly view of one source asset.
type AssetEntry struct {
	Name    string
	Path    string // relative to its scan root (vault/static)
	Source  string // "content" or "static"
	Size    int64
	Format  string // ".jpg", ".png", ".webp", ".svg"...
	ModTime time.Time
}

// AssetSummary is the aggregate over all assets.
type AssetSummary struct {
	Total     int
	TotalSize int64
	ByFormat  []FormatCount // sorted by count desc
}

type FormatCount struct {
	Format string
	Count  int
	Size   int64
}

var imageExts = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".avif": true,
	".svg":  true,
	".bmp":  true,
}

// collectAssets walks the content and static directories looking for
// image files. The result is read-only inventory: optimization happens
// during `osg build`, this surface is for visibility only.
func collectAssets(cfg config.Config, logger *slog.Logger) ([]AssetEntry, AssetSummary) {
	out := []AssetEntry{}
	out = append(out, walkAssets(cfg.ContentDir, "content", logger)...)
	out = append(out, walkAssets(cfg.StaticDir, "static", logger)...)

	sort.Slice(out, func(i, j int) bool {
		if out[i].Size == out[j].Size {
			return out[i].Path < out[j].Path
		}
		return out[i].Size > out[j].Size
	})

	summary := AssetSummary{Total: len(out)}
	formatMap := map[string]*FormatCount{}
	for _, a := range out {
		summary.TotalSize += a.Size
		fc, ok := formatMap[a.Format]
		if !ok {
			fc = &FormatCount{Format: a.Format}
			formatMap[a.Format] = fc
		}
		fc.Count++
		fc.Size += a.Size
	}
	for _, fc := range formatMap {
		summary.ByFormat = append(summary.ByFormat, *fc)
	}
	sort.Slice(summary.ByFormat, func(i, j int) bool {
		if summary.ByFormat[i].Count == summary.ByFormat[j].Count {
			return summary.ByFormat[i].Format < summary.ByFormat[j].Format
		}
		return summary.ByFormat[i].Count > summary.ByFormat[j].Count
	})
	return out, summary
}

func walkAssets(root, label string, logger *slog.Logger) []AssetEntry {
	if root == "" {
		return nil
	}
	var out []AssetEntry
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable subtrees
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if !imageExts[ext] {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		out = append(out, AssetEntry{
			Name:    d.Name(),
			Path:    rel,
			Source:  label,
			Size:    info.Size(),
			Format:  ext,
			ModTime: info.ModTime(),
		})
		return nil
	})
	if err != nil && logger != nil {
		logger.Warn("asset walk failed", "root", root, "error", err)
	}
	return out
}
