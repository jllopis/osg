package audit

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Severity levels for audit findings.
const (
	SeverityError   = "error"
	SeverityWarning = "warning"
	SeverityInfo    = "info"
)

// Finding represents a single audit issue.
type Finding struct {
	Severity string `json:"severity"`
	Category string `json:"category"`
	File     string `json:"file"`
	Message  string `json:"message"`
	Fix      string `json:"fix,omitempty"`
}

// Report holds the results of a site audit.
type Report struct {
	Findings  []Finding `json:"findings"`
	Files     int       `json:"files_checked"`
	TotalSize int64     `json:"total_size_bytes"`
}

// ErrorCount returns the number of error-severity findings.
func (r *Report) ErrorCount() int {
	n := 0
	for _, f := range r.Findings {
		if f.Severity == SeverityError {
			n++
		}
	}
	return n
}

// Run performs a site audit on the given public directory.
func Run(publicDir string) (*Report, error) {
	report := &Report{}

	err := filepath.Walk(publicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		report.Files++
		report.TotalSize += info.Size()

		relPath, _ := filepath.Rel(publicDir, path)

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".html", ".htm":
			data, err := os.ReadFile(path)
			if err != nil {
				return nil // skip unreadable files
			}
			checkHTML(relPath, string(data), info.Size(), report)
		}

		// Performance: large files.
		if info.Size() > 500*1024 {
			report.Findings = append(report.Findings, Finding{
				Severity: SeverityWarning,
				Category: "performance",
				File:     relPath,
				Message:  fmt.Sprintf("File is %s (> 500KB)", humanSize(info.Size())),
				Fix:      "Consider compressing or splitting this file",
			})
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk public dir: %w", err)
	}

	sort.Slice(report.Findings, func(i, j int) bool {
		if report.Findings[i].Severity != report.Findings[j].Severity {
			return severityOrder(report.Findings[i].Severity) < severityOrder(report.Findings[j].Severity)
		}
		return report.Findings[i].File < report.Findings[j].File
	})

	return report, nil
}

var (
	reImgTag       = regexp.MustCompile(`(?i)<img\s[^>]*>`)
	reAltAttr      = regexp.MustCompile(`(?i)\balt\s*=\s*"[^"]*"`)
	reEmptyAlt     = regexp.MustCompile(`(?i)\balt\s*=\s*""`)
	reHeading      = regexp.MustCompile(`(?i)<h([1-6])\b`)
	reOpenTag      = regexp.MustCompile(`<(html|head|body|div|section|article|nav|main|aside|footer|header|ul|ol|li|table|tr|td|th|thead|tbody|form|select|option)\b[^/]*>`)
	reCloseTag     = regexp.MustCompile(`</(html|head|body|div|section|article|nav|main|aside|footer|header|ul|ol|li|table|tr|td|th|thead|tbody|form|select|option)>`)
	reInlineScript = regexp.MustCompile(`(?i)<script\b[^>]*>[^<]{500,}</script>`)
)

func checkHTML(relPath, html string, size int64, report *Report) {
	// Check images missing alt attributes.
	imgs := reImgTag.FindAllString(html, -1)
	for _, img := range imgs {
		if !reAltAttr.MatchString(img) {
			report.Findings = append(report.Findings, Finding{
				Severity: SeverityWarning,
				Category: "accessibility",
				File:     relPath,
				Message:  "Image missing alt attribute",
				Fix:      "Add alt=\"description\" to all <img> tags",
			})
			break // one finding per file
		}
		if reEmptyAlt.MatchString(img) {
			// Empty alt is valid for decorative images, just info.
			report.Findings = append(report.Findings, Finding{
				Severity: SeverityInfo,
				Category: "accessibility",
				File:     relPath,
				Message:  "Image has empty alt attribute (decorative?)",
			})
			break
		}
	}

	// Check heading hierarchy — h1 should come before h2, etc.
	headings := reHeading.FindAllStringSubmatch(html, -1)
	if len(headings) > 0 {
		prevLevel := 0
		for _, m := range headings {
			level := int(m[1][0] - '0')
			if level > prevLevel+1 && prevLevel > 0 {
				report.Findings = append(report.Findings, Finding{
					Severity: SeverityWarning,
					Category: "accessibility",
					File:     relPath,
					Message:  fmt.Sprintf("Heading level skipped: h%d → h%d", prevLevel, level),
					Fix:      "Ensure heading levels increase sequentially (h1 → h2 → h3)",
				})
				break
			}
			prevLevel = level
		}
	}

	// Check unclosed tags (basic heuristic).
	opens := reOpenTag.FindAllStringSubmatch(html, -1)
	closes := reCloseTag.FindAllStringSubmatch(html, -1)
	openCounts := map[string]int{}
	for _, m := range opens {
		openCounts[strings.ToLower(m[1])]++
	}
	for _, m := range closes {
		openCounts[strings.ToLower(m[1])]--
	}
	for tag, count := range openCounts {
		if count > 2 { // allow small mismatches from templates
			report.Findings = append(report.Findings, Finding{
				Severity: SeverityError,
				Category: "html",
				File:     relPath,
				Message:  fmt.Sprintf("Potentially unclosed <%s> tag (%d unmatched)", tag, count),
				Fix:      fmt.Sprintf("Check for unclosed <%s> tags", tag),
			})
		}
	}

	// Large inline scripts.
	if reInlineScript.MatchString(html) {
		report.Findings = append(report.Findings, Finding{
			Severity: SeverityInfo,
			Category: "performance",
			File:     relPath,
			Message:  "Large inline script detected (> 500 chars)",
			Fix:      "Consider moving to an external .js file for caching",
		})
	}

	// Page size warning.
	if size > 200*1024 {
		report.Findings = append(report.Findings, Finding{
			Severity: SeverityWarning,
			Category: "performance",
			File:     relPath,
			Message:  fmt.Sprintf("HTML page is %s (> 200KB)", humanSize(size)),
			Fix:      "Consider pagination or lazy loading for large pages",
		})
	}
}

func severityOrder(s string) int {
	switch s {
	case SeverityError:
		return 0
	case SeverityWarning:
		return 1
	case SeverityInfo:
		return 2
	default:
		return 3
	}
}

func humanSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1f KB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// FormatReport returns a human-readable summary of the audit.
func FormatReport(r *Report) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Site Audit: %d files checked (%s total)\n\n",
		r.Files, humanSize(r.TotalSize))

	if len(r.Findings) == 0 {
		b.WriteString("No issues found.\n")
		return b.String()
	}

	counts := map[string]int{}
	for _, f := range r.Findings {
		counts[f.Severity]++
	}

	fmt.Fprintf(&b, "Found %d issues: %d errors, %d warnings, %d info\n\n",
		len(r.Findings), counts[SeverityError], counts[SeverityWarning], counts[SeverityInfo])

	for _, f := range r.Findings {
		icon := "ℹ"
		switch f.Severity {
		case SeverityError:
			icon = "✘"
		case SeverityWarning:
			icon = "!"
		}
		fmt.Fprintf(&b, "  %s [%s] %s: %s\n", icon, f.Category, f.File, f.Message)
		if f.Fix != "" {
			fmt.Fprintf(&b, "    → %s\n", f.Fix)
		}
	}

	return b.String()
}
