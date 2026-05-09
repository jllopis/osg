package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func sampleParams(t *testing.T) serviceParams {
	t.Helper()
	dir := t.TempDir()
	return serviceParams{
		Label:   "osg-ui",
		Exec:    "/usr/local/bin/osg",
		Workdir: dir,
		Config:  filepath.Join(dir, "config.yaml"),
		LogOut:  filepath.Join(dir, ".osg/logs/osg-ui.out.log"),
		LogErr:  filepath.Join(dir, ".osg/logs/osg-ui.err.log"),
	}
}

// ---------------------------------------------------------------------------
// linuxUnitContent
// ---------------------------------------------------------------------------

func TestLinuxUnitContent_StructureAndKeys(t *testing.T) {
	p := sampleParams(t)
	out := linuxUnitContent(p)

	for _, want := range []string{
		"[Unit]",
		"Description=OSG local web dashboard",
		"After=network.target",
		"[Service]",
		"Type=simple",
		"ExecStart=/usr/local/bin/osg ui -c " + p.Config,
		"WorkingDirectory=" + p.Workdir,
		"Restart=on-failure",
		"RestartSec=5",
		"StandardOutput=append:" + p.LogOut,
		"StandardError=append:" + p.LogErr,
		"[Install]",
		"WantedBy=default.target",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing line %q in unit:\n%s", want, out)
		}
	}
}

func TestLinuxUnitContent_QuotesPathsWithSpaces(t *testing.T) {
	p := sampleParams(t)
	p.Workdir = "/Users/x/Site With Space"
	p.Config = filepath.Join(p.Workdir, "config.yaml")
	out := linuxUnitContent(p)
	if !strings.Contains(out, `WorkingDirectory="/Users/x/Site With Space"`) {
		t.Errorf("path with space must be quoted; output:\n%s", out)
	}
	if !strings.Contains(out, `ExecStart=/usr/local/bin/osg ui -c "/Users/x/Site With Space/config.yaml"`) {
		t.Errorf("config path with space must be quoted in ExecStart; output:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// darwinPlistContent
// ---------------------------------------------------------------------------

func TestDarwinPlistContent_StructureAndKeys(t *testing.T) {
	p := sampleParams(t)
	out := darwinPlistContent(p)

	for _, want := range []string{
		`<?xml version="1.0" encoding="UTF-8"?>`,
		"<!DOCTYPE plist",
		`<plist version="1.0">`,
		"<key>Label</key>",
		"<string>com.jllopis.osg-ui</string>",
		"<key>ProgramArguments</key>",
		"<string>/usr/local/bin/osg</string>",
		"<string>ui</string>",
		"<string>-c</string>",
		"<string>" + p.Config + "</string>",
		"<key>WorkingDirectory</key>",
		"<key>RunAtLoad</key>",
		"<true/>",
		"<key>KeepAlive</key>",
		"<key>StandardOutPath</key>",
		"<string>" + p.LogOut + "</string>",
		"<key>StandardErrorPath</key>",
		"<string>" + p.LogErr + "</string>",
		"</plist>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing fragment %q in plist:\n%s", want, out)
		}
	}
}

func TestDarwinPlistContent_EscapesSpecialChars(t *testing.T) {
	p := sampleParams(t)
	p.Workdir = "/Users/x/Site & <stuff>"
	out := darwinPlistContent(p)
	if !strings.Contains(out, "Site &amp; &lt;stuff&gt;") {
		t.Errorf("plist must XML-escape & < > inside <string>; output:\n%s", out)
	}
	if strings.Contains(out, "Site & <stuff>") {
		t.Errorf("raw special chars leaked into plist; output:\n%s", out)
	}
}

// ---------------------------------------------------------------------------
// resolveServiceParams
// ---------------------------------------------------------------------------

func TestResolveServiceParams_DefaultsFromCwd(t *testing.T) {
	cwd, _ := os.Getwd()
	defer func() { _ = os.Chdir(cwd) }()

	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}

	got, err := resolveServiceParams(CLIOptions{ConfigPath: "config.yaml"}, ServiceInstallOptions{})
	if err != nil {
		t.Fatalf("resolveServiceParams: %v", err)
	}
	// Resolve once to handle macOS /var → /private/var symlink: both
	// the workdir and the config base must agree on the canonical
	// path. Direct string comparison would flap between platforms.
	wantBase, _ := filepath.EvalSymlinks(got.Workdir)
	gotConfigBase, _ := filepath.EvalSymlinks(filepath.Dir(got.Config))
	if wantBase == "" || gotConfigBase == "" {
		t.Skip("filesystem doesn't support EvalSymlinks; skipping path equality")
	}
	if gotConfigBase != wantBase {
		t.Errorf("Config dir = %q, want %q (defaults must use the resolved cwd)",
			gotConfigBase, wantBase)
	}
	if filepath.Base(got.Config) != "config.yaml" {
		t.Errorf("Config base = %q, want config.yaml", filepath.Base(got.Config))
	}
	if got.Exec == "" {
		t.Error("Exec must be populated from os.Executable when option is empty")
	}
	if got.LogOut == "" || got.LogErr == "" {
		t.Errorf("log paths must be populated, got out=%q err=%q", got.LogOut, got.LogErr)
	}
}

func TestResolveServiceParams_HonoursExplicitFields(t *testing.T) {
	tmp := t.TempDir()
	got, err := resolveServiceParams(CLIOptions{}, ServiceInstallOptions{
		Workdir: tmp,
		Config:  "/etc/osg/config.yaml",
		Exec:    "/opt/osg/bin/osg",
	})
	if err != nil {
		t.Fatalf("resolveServiceParams: %v", err)
	}
	if got.Exec != "/opt/osg/bin/osg" {
		t.Errorf("Exec = %q, want /opt/osg/bin/osg", got.Exec)
	}
	if got.Config != "/etc/osg/config.yaml" {
		t.Errorf("Config = %q, want /etc/osg/config.yaml", got.Config)
	}
	wantWorkdir, _ := filepath.EvalSymlinks(tmp)
	gotWorkdir, _ := filepath.EvalSymlinks(got.Workdir)
	if gotWorkdir != wantWorkdir {
		t.Errorf("Workdir = %q, want %q", gotWorkdir, wantWorkdir)
	}
}

func TestResolveServiceParams_RelativeConfigJoinedToWorkdir(t *testing.T) {
	tmp := t.TempDir()
	got, err := resolveServiceParams(CLIOptions{}, ServiceInstallOptions{
		Workdir: tmp,
		Config:  "config.yaml",
	})
	if err != nil {
		t.Fatalf("resolveServiceParams: %v", err)
	}
	want, _ := filepath.EvalSymlinks(filepath.Join(tmp, "config.yaml"))
	gotConfig, _ := filepath.EvalSymlinks(got.Config)
	if gotConfig != "" && gotConfig != want {
		// EvalSymlinks of a non-existent file returns "" — fall back
		// to direct comparison in that case.
		t.Errorf("Config = %q, want %q (joined with workdir)", got.Config, want)
	}
	if got.Config != filepath.Join(got.Workdir, "config.yaml") {
		t.Errorf("Config not joined with workdir: %q vs %q", got.Config, filepath.Join(got.Workdir, "config.yaml"))
	}
}

// ---------------------------------------------------------------------------
// shellEscape & xmlEscape
// ---------------------------------------------------------------------------

func TestShellEscape_PathsWithoutSpecialsPassThrough(t *testing.T) {
	if got := shellEscape("/usr/bin/osg"); got != "/usr/bin/osg" {
		t.Errorf("got %q, want unchanged", got)
	}
}

func TestShellEscape_PathsWithSpacesQuoted(t *testing.T) {
	if got := shellEscape("/Users/x/My Site"); got != `"/Users/x/My Site"` {
		t.Errorf("got %q, want quoted", got)
	}
}

func TestXMLEscape(t *testing.T) {
	cases := map[string]string{
		"a&b":   "a&amp;b",
		"<x>":   "&lt;x&gt;",
		`"q"`:   "&quot;q&quot;",
		"plain": "plain",
	}
	for in, want := range cases {
		if got := xmlEscape(in); got != want {
			t.Errorf("xmlEscape(%q) = %q, want %q", in, got, want)
		}
	}
}
