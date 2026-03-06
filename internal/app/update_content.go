package app

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"osg/internal/config"
	"osg/internal/content"
	"osg/internal/date"
	"osg/internal/frontmatter"
	"osg/internal/logging"
	"osg/internal/publish"
	"osg/internal/slug"
	"osg/internal/vault"
	"osg/internal/wikilink"
)

// pageIndexEntry holds computed info for a single note during the first pass.
type pageIndexEntry struct {
	sourcePath string
	outputPath string
	urlPath    string
	title      string
	fm         map[string]any
	body       []byte
	isDraft    bool
	pub        bool
	dateValue  time.Time
	slugValue  string
	osgPath    string
}

type UpdateStats struct {
	Total    int
	Parsed   int
	Exported int
	Skipped  int
	Drafts   int
	Errors   int
	Images   int
	Links    int
	Cleaned  int
}

func RunUpdateContent(_ context.Context, opts CLIOptions) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	if opts.VaultPath != "" {
		cfg.VaultPath = opts.VaultPath
	}
	if opts.OsgContentDir != "" {
		cfg.ContentDir = opts.OsgContentDir
	}
	if opts.IncludeDrafts != nil {
		cfg.IncludeDrafts = *opts.IncludeDrafts
	}

	vaultPath, err := config.ResolveVaultPath(cfg)
	if err != nil {
		return err
	}

	logger := logging.NewWithWriter(cfg.Logging, opts.Verbose, opts.LogWriter)

	files, err := vault.ListMarkdownFiles(vaultPath)
	if err != nil {
		return err
	}

	if len(files) == 0 {
		logger.Info("no markdown files found", "vault", vaultPath)
		return nil
	}

	// Build image index for resolving osg.image and wikilink references
	imageIndex, err := vault.BuildImageIndex(vaultPath)
	if err != nil {
		logger.Warn("failed to build image index, image resolution disabled", "error", err)
		imageIndex = vault.ImageIndex{}
	}
	logger.Info("image index built", "entries", len(imageIndex))

	stats := UpdateStats{Total: len(files)}
	seen := map[string]string{}

	// First pass: parse all files and build page index for wikilink resolution
	entries := make([]*pageIndexEntry, 0, len(files))
	pageIndex := make(map[string]string) // normalized title -> URL path

	for _, path := range files {
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			logger.Warn("failed to read file", "path", path, "error", readErr)
			stats.Errors++
			continue
		}

		fm, body, _, fmErr := frontmatter.SplitFrontmatter(data)
		if fmErr != nil {
			logger.Warn("failed to parse frontmatter", "path", path, "error", fmErr)
			stats.Errors++
			continue
		}
		stats.Parsed++

		pub, isDraft := publish.ShouldPublish(fm)
		if !pub {
			stats.Skipped++
			continue
		}
		if isDraft && !cfg.IncludeDrafts {
			stats.Drafts++
			continue
		}

		info, statErr := os.Stat(path)
		if statErr != nil {
			logger.Warn("failed to stat file", "path", path, "error", statErr)
			stats.Errors++
			continue
		}

		dateValue := date.Derive(fm, info)
		slugValue := slug.Derive(fm, info.Name())
		osgPath := publish.GetOSGString(fm, "path")
		osgPermalink := publish.GetOSGString(fm, "permalink")

		// Resolve title for permalink placeholders
		// Precedence: osg.title > fm.title > fm.name > filename
		titleForPermalink := publish.GetOSGString(fm, "title")
		if titleForPermalink == "" {
			titleForPermalink = pickValue(fm, "title", "name")
		}
		if titleForPermalink == "" {
			titleForPermalink = strings.TrimSuffix(info.Name(), filepath.Ext(info.Name()))
		}

		// Determine the page language for content path placement.
		pageLang := strings.ToLower(strings.TrimSpace(pickValue(fm, "lang", "language")))

		var outputPath string
		switch {
		case osgPermalink != "":
			// osg.permalink: highest precedence, supports placeholders
			expanded := content.ExpandPermalink(osgPermalink, dateValue, slugValue, titleForPermalink)
			outputPath = filepath.Join(cfg.ContentDir, expanded, "index.md")
		case osgPath != "":
			outputPath = filepath.Join(cfg.ContentDir, filepath.Clean(osgPath), "index.md")
		default:
			outputPath = content.BuildOutputPath(cfg.ContentDir, cfg.ContentLayout, dateValue, slugValue, titleForPermalink)
		}

		// For non-default languages, inject /{lang}/ prefix into the content
		// path so that the build phase places these pages under /en/, /fr/, etc.
		if pageLang != "" && pageLang != cfg.DefaultLanguage && cfg.IsMultilingual() {
			rel, _ := filepath.Rel(cfg.ContentDir, outputPath)
			outputPath = filepath.Join(cfg.ContentDir, pageLang, rel)
		}

		if prev, exists := seen[outputPath]; exists {
			logger.Error("output path collision", "path", outputPath, "first", prev, "second", path)
			stats.Errors++
			continue
		}
		seen[outputPath] = path

		// Compute URL path for this page
		outputDir := filepath.Dir(outputPath)
		urlPath := ""
		if relDir, relErr := filepath.Rel(cfg.ContentDir, outputDir); relErr == nil {
			urlPath = "/" + filepath.ToSlash(relDir) + "/"
		}

		// Get title for index
		title := pickTitle(fm)

		entry := &pageIndexEntry{
			sourcePath: path,
			outputPath: outputPath,
			urlPath:    urlPath,
			title:      title,
			fm:         fm,
			body:       body,
			isDraft:    isDraft,
			pub:        pub,
			dateValue:  dateValue,
			slugValue:  slugValue,
			osgPath:    osgPath,
		}
		entries = append(entries, entry)

		// Add to page index (normalized title -> URL)
		if title != "" && urlPath != "" {
			normalized := wikilink.NormalizeTitle(title)
			if _, exists := pageIndex[normalized]; !exists {
				pageIndex[normalized] = urlPath
			}
		}

		// Also add aliases to the index
		if aliases := pickAliases(fm); len(aliases) > 0 && urlPath != "" {
			for _, alias := range aliases {
				normalized := wikilink.NormalizeTitle(alias)
				if normalized != "" {
					if _, exists := pageIndex[normalized]; !exists {
						pageIndex[normalized] = urlPath
					}
				}
			}
		}
	}

	logger.Info("page index built", "entries", len(pageIndex))

	// Second pass: process and write files with wikilink resolution
	for _, entry := range entries {
		if opts.DryRun {
			logger.Info("dry-run", "source", entry.sourcePath, "dest", entry.outputPath)
			stats.Exported++
			continue
		}

		outputDir := filepath.Dir(entry.outputPath)

		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			logger.Warn("failed to create output directory", "path", entry.outputPath, "error", err)
			stats.Errors++
			continue
		}

		dateISO := date.FormatISO(entry.dateValue)
		outputFM := content.NormalizeFrontmatter(entry.fm, entry.slugValue, dateISO, entry.isDraft, filepath.Base(entry.sourcePath))

		// Resolve the frontmatter image
		if imgRef, ok := outputFM["image"].(string); ok && imgRef != "" {
			if isExternalURL(imgRef) {
				logger.Info("image is external URL", "url", imgRef, "note", entry.sourcePath)
			} else if strings.HasPrefix(imgRef, "/") {
				// Already absolute path
			} else if srcPath, found := imageIndex.Resolve(imgRef); found {
				destName := filepath.Base(srcPath)
				destPath := filepath.Join(outputDir, destName)
				if cpErr := copyFile(srcPath, destPath); cpErr != nil {
					logger.Warn("failed to copy image", "src", srcPath, "dest", destPath, "error", cpErr)
					delete(outputFM, "image")
				} else {
					outputFM["image"] = entry.urlPath + destName
					stats.Images++
					logger.Info("copied image", "src", srcPath, "dest", destPath, "url", outputFM["image"])
				}
			} else {
				logger.Warn("image not found in vault, will use placeholder", "ref", imgRef, "note", entry.sourcePath)
				delete(outputFM, "image")
			}
		}

		body := entry.body

		// Rewrite wikilink images in body
		body = wikilink.RewriteImageLinks(body, func(ref string) (string, bool) {
			srcPath, found := imageIndex.Resolve(ref)
			if !found {
				logger.Warn("wikilink image not found in vault", "ref", ref, "note", entry.sourcePath)
				return "", false
			}

			destName := filepath.Base(srcPath)
			destPath := filepath.Join(outputDir, destName)
			if cpErr := copyFile(srcPath, destPath); cpErr != nil {
				logger.Warn("failed to copy wikilink image", "src", srcPath, "dest", destPath, "error", cpErr)
				return "", false
			}

			stats.Images++
			logger.Info("copied wikilink image", "src", srcPath, "dest", destPath)
			return destName, true
		})

		// Rewrite wikilink text references: [[Note Title]] or [[Note Title|display]]
		body = wikilink.RewriteTextLinks(body, func(title string) (string, bool) {
			normalized := wikilink.NormalizeTitle(title)
			if url, ok := pageIndex[normalized]; ok {
				stats.Links++
				return url, true
			}
			// Not found: return empty to strip [[ ]]
			logger.Debug("wikilink not resolved", "title", title, "note", entry.sourcePath)
			return "", false
		})

		outputBytes, err := content.RenderMarkdown(outputFM, body)
		if err != nil {
			logger.Warn("failed to render output", "path", entry.sourcePath, "error", err)
			stats.Errors++
			continue
		}

		if err := os.WriteFile(entry.outputPath, outputBytes, 0o644); err != nil {
			logger.Warn("failed to write output", "path", entry.outputPath, "error", err)
			stats.Errors++
			continue
		}

		logger.Info("exported", "source", entry.sourcePath, "dest", entry.outputPath)
		stats.Exported++
	}

	// Remove stale content directories that no longer correspond to
	// any vault note.  We walk content/ looking for index.md files
	// whose path is not in the "seen" map (the set of valid outputs
	// produced in this run).
	if !opts.DryRun {
		stats.Cleaned = removeStaleContent(cfg.ContentDir, seen, logger)
	}

	logger.Info("update-content summary",
		"total", stats.Total,
		"parsed", stats.Parsed,
		"exported", stats.Exported,
		"skipped", stats.Skipped,
		"drafts", stats.Drafts,
		"images", stats.Images,
		"links", stats.Links,
		"cleaned", stats.Cleaned,
		"errors", stats.Errors,
	)

	if stats.Errors > 0 {
		return fmt.Errorf("completed with %d errors", stats.Errors)
	}

	return nil
}

// removeStaleContent walks contentDir looking for index.md files that were
// not produced in the current run (not present in validPaths).  For each
// stale file it removes the entire parent directory (which also removes
// co-located images).  Section index files (_index.md) are left untouched.
func removeStaleContent(contentDir string, validPaths map[string]string, logger *slog.Logger) int {
	cleaned := 0

	_ = filepath.WalkDir(contentDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // best-effort
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "index.md" {
			return nil // skip images and other assets
		}
		// Never touch section index files.
		if d.Name() == "_index.md" {
			return nil
		}

		if _, ok := validPaths[path]; ok {
			return nil // this file was just written
		}

		// Stale: remove the entire directory (index.md + co-located images).
		dir := filepath.Dir(path)
		if logger != nil {
			logger.Info("removing stale content", "path", dir)
		}
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			if logger != nil {
				logger.Warn("failed to remove stale content", "path", dir, "error", rmErr)
			}
		} else {
			cleaned++
		}
		return filepath.SkipDir
	})

	return cleaned
}

// pickValue returns the first non-empty string value for the given keys.
func pickValue(fm map[string]any, keys ...string) string {
	if fm == nil {
		return ""
	}
	for _, key := range keys {
		if val, ok := fm[key].(string); ok {
			val = strings.TrimSpace(val)
			if val != "" {
				return val
			}
		}
	}
	return ""
}

// pickTitle extracts title from frontmatter (title or name field)
func pickTitle(fm map[string]any) string {
	if fm == nil {
		return ""
	}
	if title, ok := fm["title"].(string); ok {
		return strings.TrimSpace(title)
	}
	if name, ok := fm["name"].(string); ok {
		return strings.TrimSpace(name)
	}
	return ""
}

// pickAliases extracts aliases from frontmatter (as string slice)
func pickAliases(fm map[string]any) []string {
	if fm == nil {
		return nil
	}
	raw, ok := fm["aliases"]
	if !ok {
		return nil
	}

	// Handle []any (YAML list)
	if list, ok := raw.([]any); ok {
		var result []string
		for _, item := range list {
			if s, ok := item.(string); ok {
				s = strings.TrimSpace(s)
				if s != "" {
					result = append(result, s)
				}
			}
		}
		return result
	}

	// Handle []string
	if list, ok := raw.([]string); ok {
		var result []string
		for _, s := range list {
			s = strings.TrimSpace(s)
			if s != "" {
				result = append(result, s)
			}
		}
		return result
	}

	return nil
}

// isExternalURL returns true if the string looks like an external URL.
func isExternalURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
