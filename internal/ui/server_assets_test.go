package ui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"osg/internal/audit"
	"osg/internal/operations"
)

// --- loadTemplates -----------------------------------------------------

func TestLoadTemplates(t *testing.T) {
	tpls, err := loadTemplates()
	if err != nil {
		t.Fatalf("loadTemplates: %v", err)
	}
	wantPages := []string{
		"dashboard", "actions", "history", "vault", "page", "plugins",
		"assets", "scheduler", "services", "import", "themes", "audit",
		"fragments",
	}
	for _, name := range wantPages {
		if _, ok := tpls[name]; !ok {
			t.Errorf("loadTemplates: missing template key %q", name)
		}
	}
	if len(tpls) != len(wantPages) {
		t.Errorf("loadTemplates: got %d templates, want %d", len(tpls), len(wantPages))
	}
}

// --- humanSize ---------------------------------------------------------

func TestHumanSize(t *testing.T) {
	tests := []struct {
		name string
		n    int64
		want string
	}{
		{"zero", 0, "0 B"},
		{"bytes", 512, "512 B"},
		{"just below kb", 1023, "1023 B"},
		{"one kb", 1024, "1.0 KB"},
		{"kb fractional", 1536, "1.5 KB"},
		{"just below mb", 1024*1024 - 1, "1024.0 KB"},
		{"one mb", 1024 * 1024, "1.0 MB"},
		{"one gb", 1024 * 1024 * 1024, "1.0 GB"},
		{"gb fractional", 1024 * 1024 * 1024 * 3 / 2, "1.5 GB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := humanSize(tt.n); got != tt.want {
				t.Errorf("humanSize(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

// --- uptimeStr ---------------------------------------------------------

func TestUptimeStr(t *testing.T) {
	now := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		start time.Time
		want  string
	}{
		{"zero start", time.Time{}, "—"},
		{"sub-second", now.Add(-500 * time.Millisecond), "<1s"},
		{"seconds", now.Add(-5 * time.Second), "5s"},
		{"minutes", now.Add(-2*time.Minute - 3*time.Second), "2m 3s"},
		{"hours", now.Add(-3*time.Hour - 15*time.Minute), "3h 15m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := uptimeStr(tt.start, now); got != tt.want {
				t.Errorf("uptimeStr(%v, %v) = %q, want %q", tt.start, now, got, tt.want)
			}
		})
	}
}

// --- collectAssets / walkAssets ---------------------------------------

func TestWalkAssetsEmptyRoot(t *testing.T) {
	if got := walkAssets("", "content", newTestLogger()); got != nil {
		t.Errorf("walkAssets empty root = %v, want nil", got)
	}
	missing := filepath.Join(t.TempDir(), "does-not-exist")
	if got := walkAssets(missing, "content", newTestLogger()); got != nil {
		t.Errorf("walkAssets nonexistent root = %v, want nil", got)
	}
}

func TestWalkAssetsOnlyImages(t *testing.T) {
	root := t.TempDir()
	files := map[string][]byte{
		"a.png":         []byte("PNGDATA1234"), // image
		"b.svg":         []byte("<svg/>"),      // image
		"notes.md":      []byte("# markdown"),  // non-image, skipped
		"sub/c.JPG":     []byte("jpegdata"),    // image, uppercase ext
		"sub/data.json": []byte("{}"),          // non-image, skipped
	}
	for rel, body := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(abs, body, 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	got := walkAssets(root, "content", newTestLogger())
	if len(got) != 3 {
		t.Fatalf("walkAssets counted %d entries, want 3 images: %+v", len(got), got)
	}
	for _, e := range got {
		if e.Source != "content" {
			t.Errorf("entry %q Source = %q, want content", e.Name, e.Source)
		}
		switch e.Format {
		case ".png", ".svg", ".jpg":
		default:
			t.Errorf("entry %q unexpected format %q", e.Name, e.Format)
		}
	}
}

func TestCollectAssetsSummary(t *testing.T) {
	cfg := newTestConfig(t)

	// Two PNGs and one SVG in content; one PNG in static.
	write := func(dir, rel string, size int) {
		abs := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(abs, make([]byte, size), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}
	write(cfg.ContentDir, "one.png", 100)
	write(cfg.ContentDir, "two.png", 200)
	write(cfg.ContentDir, "logo.svg", 50)
	write(cfg.ContentDir, "readme.md", 999) // ignored
	write(cfg.StaticDir, "hero.png", 300)

	entries, summary := collectAssets(cfg, newTestLogger())

	if summary.Total != 4 {
		t.Errorf("summary.Total = %d, want 4", summary.Total)
	}
	if len(entries) != 4 {
		t.Errorf("len(entries) = %d, want 4", len(entries))
	}
	wantSize := int64(100 + 200 + 50 + 300)
	if summary.TotalSize != wantSize {
		t.Errorf("summary.TotalSize = %d, want %d", summary.TotalSize, wantSize)
	}

	// Entries sorted by size desc.
	for i := 1; i < len(entries); i++ {
		if entries[i-1].Size < entries[i].Size {
			t.Errorf("entries not sorted by size desc at %d: %d < %d", i, entries[i-1].Size, entries[i].Size)
		}
	}

	// ByFormat sorted by count desc; .png (3) before .svg (1).
	if len(summary.ByFormat) != 2 {
		t.Fatalf("len(ByFormat) = %d, want 2: %+v", len(summary.ByFormat), summary.ByFormat)
	}
	for i := 1; i < len(summary.ByFormat); i++ {
		if summary.ByFormat[i-1].Count < summary.ByFormat[i].Count {
			t.Errorf("ByFormat not sorted by count desc at %d", i)
		}
	}
	if summary.ByFormat[0].Format != ".png" || summary.ByFormat[0].Count != 3 {
		t.Errorf("ByFormat[0] = %+v, want .png count 3", summary.ByFormat[0])
	}
	if summary.ByFormat[0].Size != int64(100+200+300) {
		t.Errorf("ByFormat[0].Size = %d, want 600", summary.ByFormat[0].Size)
	}
}

// --- servicesFromRunner / serviceFromSnapshot -------------------------

func TestServicesFromRunnerNil(t *testing.T) {
	got := servicesFromRunner(nil)
	if got == nil {
		t.Fatal("servicesFromRunner(nil) = nil, want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Errorf("servicesFromRunner(nil) len = %d, want 0", len(got))
	}
}

func TestServicesFromRunnerOnlyServices(t *testing.T) {
	r := newTestRunner(t)
	got := servicesFromRunner(r)
	// Only "serve" is KindService in the canonical runner.
	if len(got) != 1 {
		t.Fatalf("servicesFromRunner len = %d, want 1: %+v", len(got), got)
	}
	if got[0].Name != "serve" {
		t.Errorf("service name = %q, want serve", got[0].Name)
	}
	if got[0].Addr != ":1313" {
		t.Errorf("service addr = %q, want :1313", got[0].Addr)
	}
}

func TestServiceFromSnapshotActive(t *testing.T) {
	started := time.Date(2026, 5, 29, 10, 0, 0, 0, time.UTC)
	snap := operations.Snapshot{
		Definition: operations.Definition{Name: "serve", Description: "dev", Addr: ":1313", Kind: operations.KindService},
		State:      operations.StateRunning,
		Active:     &operations.Run{StartedAt: started, LastError: "boom"},
		LastRun:    &operations.HistoryRun{Error: "old error"},
		LogTail:    []string{"line1", "line2"},
	}
	s := serviceFromSnapshot(snap)
	if s.Name != "serve" || s.Description != "dev" || s.Addr != ":1313" {
		t.Errorf("definition fields not copied: %+v", s)
	}
	if s.State != operations.StateRunning {
		t.Errorf("State = %q, want running", s.State)
	}
	if !s.StartedAt.Equal(started) {
		t.Errorf("StartedAt = %v, want %v (from Active)", s.StartedAt, started)
	}
	if s.LastError != "boom" {
		t.Errorf("LastError = %q, want boom (from Active)", s.LastError)
	}
	if len(s.LogTail) != 2 {
		t.Errorf("LogTail = %v, want 2 lines", s.LogTail)
	}
}

func TestServiceFromSnapshotLastRunOnly(t *testing.T) {
	snap := operations.Snapshot{
		Definition: operations.Definition{Name: "serve", Kind: operations.KindService},
		State:      operations.StateError,
		Active:     nil,
		LastRun:    &operations.HistoryRun{Error: "last error msg"},
	}
	s := serviceFromSnapshot(snap)
	if !s.StartedAt.IsZero() {
		t.Errorf("StartedAt = %v, want zero (no Active)", s.StartedAt)
	}
	if s.LastError != "last error msg" {
		t.Errorf("LastError = %q, want last error msg (from LastRun)", s.LastError)
	}
}

// --- rebuildSnapshotFromRunner ----------------------------------------

func TestRebuildSnapshotNilRunner(t *testing.T) {
	snap := rebuildSnapshotFromRunner(nil)
	if snap.Available {
		t.Errorf("nil runner: Available = true, want false")
	}
}

func TestRebuildSnapshotNoBuildDef(t *testing.T) {
	store, err := operations.NewStore(filepath.Join(t.TempDir(), "ops.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	r := operations.New([]operations.Definition{
		{Name: "deploy", Kind: operations.KindTask, Run: instantRun},
	}, store)

	snap := rebuildSnapshotFromRunner(r)
	if snap.Available {
		t.Errorf("no build def: Available = true, want false")
	}
}

func TestRebuildSnapshotWithBuildDef(t *testing.T) {
	r := newTestRunner(t) // has a "build" task
	snap := rebuildSnapshotFromRunner(r)
	if !snap.Available {
		t.Errorf("build def present: Available = false, want true")
	}
	// Not running before any trigger.
	if snap.Running {
		t.Errorf("Running = true before trigger, want false")
	}
}

func TestRebuildSnapshotReflectsRunningState(t *testing.T) {
	store, err := operations.NewStore(filepath.Join(t.TempDir(), "ops.db"))
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	// "build" as a blocking service-like task so we can observe Running.
	r := operations.New([]operations.Definition{
		{Name: "build", Kind: operations.KindService, Run: blockingRun},
	}, store)

	if _, err := r.Trigger("build", nil); err != nil {
		t.Fatalf("Trigger: %v", err)
	}
	t.Cleanup(func() { _ = r.Stop("build") })

	var snap RebuildSnapshot
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		snap = rebuildSnapshotFromRunner(r)
		if snap.Running {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !snap.Available {
		t.Errorf("Available = false, want true")
	}
	if !snap.Running {
		t.Errorf("Running = false after Trigger, want true")
	}
}

// --- collectThemes -----------------------------------------------------

func TestCollectThemesEmbeddedDefault(t *testing.T) {
	cfg := newTestConfig(t) // Theme="default", ThemesDir is empty temp dir
	themes := collectThemes(cfg)
	var def *ThemeView
	for i := range themes {
		if themes[i].Name == "default" {
			def = &themes[i]
		}
	}
	if def == nil {
		t.Fatalf("collectThemes missing embedded default: %+v", themes)
	}
	if !def.Active {
		t.Errorf("default theme Active = false, want true when cfg.Theme=default")
	}
	if def.SourceLabel != "embedded" {
		t.Errorf("default SourceLabel = %q, want embedded", def.SourceLabel)
	}
}

func TestCollectThemesDefaultNotActive(t *testing.T) {
	cfg := newTestConfig(t)
	cfg.Theme = "custom"
	themes := collectThemes(cfg)
	for _, th := range themes {
		if th.Name == "default" && th.Active {
			t.Errorf("default marked Active when cfg.Theme=custom")
		}
	}
}

// --- auditFindings / pillClassForSeverity -----------------------------

func TestAuditFindingsNil(t *testing.T) {
	if got := auditFindings(nil); got != nil {
		t.Errorf("auditFindings(nil) = %v, want nil", got)
	}
}

func TestAuditFindingsFromReport(t *testing.T) {
	report := &audit.Report{
		Findings: []audit.Finding{
			{Severity: audit.SeverityError, Category: "links", File: "a.html", Message: "broken", Fix: "fix it"},
			{Severity: audit.SeverityWarning, Category: "a11y", File: "b.html", Message: "no alt"},
		},
	}
	got := auditFindings(report)
	if len(got) != 2 {
		t.Fatalf("auditFindings len = %d, want 2", len(got))
	}
	if got[0].Severity != audit.SeverityError || got[0].PillClass != "is-error" {
		t.Errorf("finding[0] = %+v, want error/is-error", got[0])
	}
	if got[0].Category != "links" || got[0].File != "a.html" || got[0].Message != "broken" || got[0].Fix != "fix it" {
		t.Errorf("finding[0] fields not copied: %+v", got[0])
	}
	if got[1].PillClass != "is-warn" {
		t.Errorf("finding[1] PillClass = %q, want is-warn", got[1].PillClass)
	}
}

func TestPillClassForSeverity(t *testing.T) {
	tests := []struct {
		sev  string
		want string
	}{
		{audit.SeverityError, "is-error"},
		{audit.SeverityWarning, "is-warn"},
		{audit.SeverityInfo, "is-running"},
		{"", "is-idle"},
		{"unknown", "is-idle"},
	}
	for _, tt := range tests {
		t.Run(tt.sev, func(t *testing.T) {
			if got := pillClassForSeverity(tt.sev); got != tt.want {
				t.Errorf("pillClassForSeverity(%q) = %q, want %q", tt.sev, got, tt.want)
			}
		})
	}
}
