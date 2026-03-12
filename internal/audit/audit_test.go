package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	report, err := Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if report.Files != 0 {
		t.Errorf("Files = %d, want 0", report.Files)
	}
	if len(report.Findings) != 0 {
		t.Errorf("Findings = %d, want 0", len(report.Findings))
	}
}

func TestRun_MissingAlt(t *testing.T) {
	dir := t.TempDir()
	html := `<!DOCTYPE html><html><body><img src="photo.jpg"></body></html>`
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	found := false
	for _, f := range report.Findings {
		if f.Category == "accessibility" && strings.Contains(f.Message, "alt") {
			found = true
		}
	}
	if !found {
		t.Error("expected accessibility finding for missing alt")
	}
}

func TestRun_HeadingSkip(t *testing.T) {
	dir := t.TempDir()
	html := `<!DOCTYPE html><html><body><h1>Title</h1><h3>Skipped h2</h3></body></html>`
	if err := os.WriteFile(filepath.Join(dir, "page.html"), []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	found := false
	for _, f := range report.Findings {
		if strings.Contains(f.Message, "Heading level skipped") {
			found = true
		}
	}
	if !found {
		t.Error("expected heading level skip finding")
	}
}

func TestRun_LargeFile(t *testing.T) {
	dir := t.TempDir()
	// Create a file > 500KB.
	data := strings.Repeat("x", 600*1024)
	if err := os.WriteFile(filepath.Join(dir, "big.html"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	found := false
	for _, f := range report.Findings {
		if f.Category == "performance" && strings.Contains(f.Message, "500KB") {
			found = true
		}
	}
	if !found {
		t.Error("expected performance finding for large file")
	}
}

func TestRun_CleanHTML(t *testing.T) {
	dir := t.TempDir()
	html := `<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Title</h1><h2>Sub</h2><img src="photo.jpg" alt="A photo"></body></html>`
	if err := os.WriteFile(filepath.Join(dir, "clean.html"), []byte(html), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, f := range report.Findings {
		if f.Severity == SeverityError {
			t.Errorf("unexpected error finding: %s", f.Message)
		}
	}
}

func TestFormatReport(t *testing.T) {
	report := &Report{
		Files:     5,
		TotalSize: 1024 * 100,
		Findings: []Finding{
			{Severity: SeverityWarning, Category: "accessibility", File: "index.html", Message: "Missing alt"},
		},
	}
	out := FormatReport(report)
	if !strings.Contains(out, "5 files") {
		t.Errorf("output missing file count: %s", out)
	}
	if !strings.Contains(out, "Missing alt") {
		t.Errorf("output missing finding: %s", out)
	}
}

func TestErrorCount(t *testing.T) {
	r := &Report{
		Findings: []Finding{
			{Severity: SeverityError},
			{Severity: SeverityWarning},
			{Severity: SeverityError},
		},
	}
	if got := r.ErrorCount(); got != 2 {
		t.Errorf("ErrorCount = %d, want 2", got)
	}
}
