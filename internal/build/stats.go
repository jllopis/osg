package build

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"osg/internal/config"
	"osg/internal/site"
	"osg/internal/vault"
)

// SiteStats holds aggregate statistics about the site.
type SiteStats struct {
	TotalPosts       int
	Drafts           int
	Scheduled        int
	Published        int
	Sections         int
	Images           int
	OutputSize       int64
	OutputFiles      int
	NextScheduled    time.Time // earliest future PublishAt across all pages
	SectionBreakdown []SectionStats
	Monthly          []MonthlyStats
}

// SectionStats holds per-section statistics.
type SectionStats struct {
	Name    string
	Posts   int
	NoTags  int
	NoImage int
}

// MonthlyStats holds publication count per month.
type MonthlyStats struct {
	Month string // "2025-01"
	Count int
}

// ComputeStats scans content and public directories to produce site statistics.
func ComputeStats(cfg config.Config) (*SiteStats, error) {
	files, err := vault.ListMarkdownFiles(cfg.ContentDir)
	if err != nil {
		return nil, fmt.Errorf("list content: %w", err)
	}

	siteIndex := site.New()
	for _, fp := range files {
		page, section, err := site.ParseFile(cfg.ContentDir, cfg.BaseURL, fp)
		if err != nil {
			continue
		}
		if page != nil {
			if page.Lang == "" {
				page.Lang = cfg.DefaultLanguage
			}
			siteIndex.AddPage(page)
		}
		if section != nil {
			siteIndex.AddSection(section)
		}
	}
	siteIndex.BuildHierarchy()

	stats := &SiteStats{
		TotalPosts: len(siteIndex.Pages),
		Sections:   len(siteIndex.Sections),
	}

	// Count drafts/published and collect per-section stats.
	sectionMap := map[string]*SectionStats{}
	monthMap := map[string]int{}

	now := time.Now()
	for _, p := range siteIndex.Pages {
		switch {
		// Scheduled-with-future-date wins over draft so a draft that
		// has been scheduled to auto-publish counts toward Scheduled
		// (and updates NextScheduled). Without this swap the scheduler
		// service couldn't see the post: ComputeStats would classify
		// it as a plain draft and NextScheduled would never advance.
		case !p.PublishAt.IsZero() && p.PublishAt.After(now):
			stats.Scheduled++
			if stats.NextScheduled.IsZero() || p.PublishAt.Before(stats.NextScheduled) {
				stats.NextScheduled = p.PublishAt
			}
		case p.Draft:
			stats.Drafts++
		default:
			stats.Published++
		}

		// Per-section breakdown.
		secName := sectionName(p.Path)
		ss, ok := sectionMap[secName]
		if !ok {
			ss = &SectionStats{Name: secName}
			sectionMap[secName] = ss
		}
		ss.Posts++
		if len(p.Taxonomies) == 0 || allEmpty(p.Taxonomies) {
			ss.NoTags++
		}
		if p.Image == "" {
			ss.NoImage++
		}

		// Monthly histogram.
		if !p.Date.IsZero() {
			month := p.Date.Format("2006-01")
			monthMap[month]++
		}
	}

	// Sort sections by post count descending.
	for _, ss := range sectionMap {
		stats.SectionBreakdown = append(stats.SectionBreakdown, *ss)
	}
	sort.Slice(stats.SectionBreakdown, func(i, j int) bool {
		return stats.SectionBreakdown[i].Posts > stats.SectionBreakdown[j].Posts
	})

	// Sort months chronologically.
	for month, count := range monthMap {
		stats.Monthly = append(stats.Monthly, MonthlyStats{Month: month, Count: count})
	}
	sort.Slice(stats.Monthly, func(i, j int) bool {
		return stats.Monthly[i].Month < stats.Monthly[j].Month
	})

	// Count images in content + static + public.
	stats.Images = countImages(cfg.ContentDir) + countImages(cfg.StaticDir)

	// Output size.
	stats.OutputSize, stats.OutputFiles = dirSize(cfg.PublicDir)

	return stats, nil
}

// FormatStats returns a human-readable multi-line summary.
func FormatStats(s *SiteStats) string {
	var b strings.Builder
	b.WriteString("=== Site Statistics ===\n")
	_, _ = fmt.Fprintf(&b, "  Posts: %d (%d published, %d drafts)\n", s.TotalPosts, s.Published, s.Drafts)
	_, _ = fmt.Fprintf(&b, "  Sections: %d\n", s.Sections)
	_, _ = fmt.Fprintf(&b, "  Images: %d\n", s.Images)
	if s.OutputFiles > 0 {
		_, _ = fmt.Fprintf(&b, "  Output: %d files (%s)\n", s.OutputFiles, humanSize(s.OutputSize))
	} else {
		b.WriteString("  Output: not built yet\n")
	}

	if len(s.SectionBreakdown) > 0 {
		b.WriteString("\n--- By Section ---\n")
		for _, ss := range s.SectionBreakdown {
			line := fmt.Sprintf("  %-20s %d posts", ss.Name, ss.Posts)
			warnings := []string{}
			if ss.NoTags > 0 {
				warnings = append(warnings, fmt.Sprintf("%d no tags", ss.NoTags))
			}
			if ss.NoImage > 0 {
				warnings = append(warnings, fmt.Sprintf("%d no image", ss.NoImage))
			}
			if len(warnings) > 0 {
				line += "  (" + strings.Join(warnings, ", ") + ")"
			}
			b.WriteString(line + "\n")
		}
	}

	if len(s.Monthly) > 0 {
		b.WriteString("\n--- Publication History ---\n")
		// Show sparkline-style histogram for the last 12 months.
		maxCount := 0
		for _, ms := range s.Monthly {
			if ms.Count > maxCount {
				maxCount = ms.Count
			}
		}
		// Show last 12 months (or all if fewer).
		start := 0
		if len(s.Monthly) > 12 {
			start = len(s.Monthly) - 12
		}
		for _, ms := range s.Monthly[start:] {
			bar := sparkBar(ms.Count, maxCount, 20)
			_, _ = fmt.Fprintf(&b, "  %s %s %d\n", ms.Month, bar, ms.Count)
		}
	}

	return b.String()
}

func sectionName(pagePath string) string {
	parts := strings.Split(strings.Trim(pagePath, "/"), "/")
	if len(parts) <= 1 {
		return "(root)"
	}
	return parts[0]
}

func allEmpty(taxonomies map[string][]string) bool {
	for _, terms := range taxonomies {
		if len(terms) > 0 {
			return false
		}
	}
	return true
}

func countImages(dir string) int {
	if dir == "" {
		return 0
	}
	count := 0
	imageExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".webp": true, ".svg": true, ".avif": true, ".bmp": true,
	}
	_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if imageExts[ext] {
			count++
		}
		return nil
	})
	return count
}

func dirSize(dir string) (int64, int) {
	if dir == "" {
		return 0, 0
	}
	var size int64
	count := 0
	_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err == nil {
			size += info.Size()
		}
		count++
		return nil
	})
	return size, count
}

func sparkBar(value, maxVal, width int) string {
	if maxVal <= 0 || width <= 0 {
		return ""
	}
	filled := (value * width) / maxVal
	if filled < 1 && value > 0 {
		filled = 1
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
