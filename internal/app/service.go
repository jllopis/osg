package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ServiceLabel is the canonical base identifier used across platforms.
// When ServiceInstallOptions.Name is empty the unit file uses this as
// the literal label; when Name is set it becomes the suffix
// (`osg-ui-<name>`), enabling multiple parallel installs in the same
// user account.
const ServiceLabel = "osg-ui"

// serviceLabel returns the platform-neutral label for a given install
// name. Empty name yields the historical bare ServiceLabel so existing
// single-instance setups keep working without changes; non-empty name
// produces a sanitised suffix so two installs in different workdirs
// can coexist as `osg-ui-blogA`, `osg-ui-blogB`, etc.
func serviceLabel(name string) string {
	n := sanitiseServiceName(name)
	if n == "" {
		return ServiceLabel
	}
	return ServiceLabel + "-" + n
}

// sanitiseServiceName lowercases the input, keeps `[a-z0-9-]`, and
// collapses every other run into a single dash so the result is safe
// to drop into systemd unit names, launchd labels, file paths and
// log filenames. Returns "" when the input contains no usable
// characters.
func sanitiseServiceName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	lastDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		case r == '-':
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		default:
			if !lastDash && b.Len() > 0 {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}

// ServiceInstallOptions configures the unit file installer.
type ServiceInstallOptions struct {
	// Name is the optional instance suffix. Empty produces the
	// historical "osg-ui" label (single-instance default); non-empty
	// produces "osg-ui-<sanitised>" so several blogs/sites can run
	// as concurrent services. Pair each install with a different
	// ui.addr in its config.yaml to avoid bind conflicts.
	Name string
	// Workdir is the working directory the service runs in. Defaults
	// to the current directory at install time so subsequent runs
	// resolve config.yaml / vault paths the way the user expects.
	Workdir string
	// Config is the path to config.yaml passed to `osg ui` via -c.
	// Empty falls back to the global -c flag (CLIOptions.ConfigPath)
	// or the literal "config.yaml" if neither is set.
	Config string
	// Exec is the absolute path to the osg binary the unit should
	// execute. Empty resolves via os.Executable() at install time.
	Exec string
	// NoStart, when true, skips the load+start step after writing the
	// unit file. The user can later run `osg service start`.
	NoStart bool
}

// serviceParams is the resolved, absolute view of the install
// inputs. Built once by resolveServiceParams and consumed by both
// the platform-specific writer and the pure content generators.
type serviceParams struct {
	Label   string
	Exec    string
	Workdir string
	Config  string
	LogOut  string
	LogErr  string
}

// resolveServiceParams turns the user-supplied options into absolute
// paths the platform writers can drop straight into the unit file.
// Errors only on the operations that genuinely require a working
// system (os.Executable, os.Getwd) so a typo-driven failure is
// easy to diagnose.
func resolveServiceParams(opts CLIOptions, sopts ServiceInstallOptions) (serviceParams, error) {
	p := serviceParams{Label: serviceLabel(sopts.Name)}

	exec := strings.TrimSpace(sopts.Exec)
	if exec == "" {
		path, err := os.Executable()
		if err != nil {
			return p, fmt.Errorf("resolve osg binary path: %w", err)
		}
		exec = path
	}
	absExec, err := filepath.Abs(exec)
	if err != nil {
		return p, fmt.Errorf("absolutise osg binary path: %w", err)
	}
	p.Exec = absExec

	wd := strings.TrimSpace(sopts.Workdir)
	if wd == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return p, fmt.Errorf("resolve working directory: %w", err)
		}
		wd = cwd
	}
	absWd, err := filepath.Abs(wd)
	if err != nil {
		return p, fmt.Errorf("absolutise working directory: %w", err)
	}
	p.Workdir = absWd

	cfg := strings.TrimSpace(sopts.Config)
	if cfg == "" {
		cfg = strings.TrimSpace(opts.ConfigPath)
	}
	if cfg == "" {
		cfg = "config.yaml"
	}
	if !filepath.IsAbs(cfg) {
		cfg = filepath.Join(p.Workdir, cfg)
	}
	p.Config = cfg

	// Logs live next to the rest of OSG state, using the resolved
	// label so multi-instance installs (osg-ui-blogA, osg-ui-blogB)
	// get distinct log files even when they share a workdir prefix.
	logsDir := filepath.Join(p.Workdir, ".osg", "logs")
	p.LogOut = filepath.Join(logsDir, p.Label+".out.log")
	p.LogErr = filepath.Join(logsDir, p.Label+".err.log")
	return p, nil
}

// linuxUnitContent renders a systemd user unit file for the dashboard.
// Pure function (no I/O) so it is reusable from tests on any platform.
//
// Restart=on-failure restarts the dashboard if it crashes; a clean
// exit (e.g. Ctrl+C) does not trigger a restart so `osg service stop`
// behaves like the user expects.
func linuxUnitContent(p serviceParams) string {
	var b strings.Builder
	b.WriteString("[Unit]\n")
	b.WriteString("Description=OSG local web dashboard (osg ui)\n")
	b.WriteString("After=network.target\n")
	b.WriteString("\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	fmt.Fprintf(&b, "ExecStart=%s ui -c %s\n", shellEscape(p.Exec), shellEscape(p.Config))
	fmt.Fprintf(&b, "WorkingDirectory=%s\n", shellEscape(p.Workdir))
	b.WriteString("Restart=on-failure\n")
	b.WriteString("RestartSec=5\n")
	// Subset of the user's environment that OSG actually needs. Full
	// env is inherited by systemd by default; this just guarantees
	// PATH is sane even when the user's shell rc isn't sourced.
	b.WriteString("Environment=PATH=/usr/local/bin:/usr/bin:/bin\n")
	fmt.Fprintf(&b, "StandardOutput=append:%s\n", shellEscape(p.LogOut))
	fmt.Fprintf(&b, "StandardError=append:%s\n", shellEscape(p.LogErr))
	b.WriteString("\n")
	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=default.target\n")
	return b.String()
}

// darwinPlistContent renders a LaunchAgent property list for the
// dashboard. Pure function (no I/O). KeepAlive=true respawns on
// crash; RunAtLoad=true starts the agent at login (no manual launchctl).
func darwinPlistContent(p serviceParams) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n")
	b.WriteString("<dict>\n")
	b.WriteString("\t<key>Label</key>\n")
	fmt.Fprintf(&b, "\t<string>com.jllopis.%s</string>\n", xmlEscape(p.Label))
	b.WriteString("\t<key>ProgramArguments</key>\n")
	b.WriteString("\t<array>\n")
	fmt.Fprintf(&b, "\t\t<string>%s</string>\n", xmlEscape(p.Exec))
	b.WriteString("\t\t<string>ui</string>\n")
	b.WriteString("\t\t<string>-c</string>\n")
	fmt.Fprintf(&b, "\t\t<string>%s</string>\n", xmlEscape(p.Config))
	b.WriteString("\t</array>\n")
	b.WriteString("\t<key>WorkingDirectory</key>\n")
	fmt.Fprintf(&b, "\t<string>%s</string>\n", xmlEscape(p.Workdir))
	b.WriteString("\t<key>RunAtLoad</key>\n\t<true/>\n")
	b.WriteString("\t<key>KeepAlive</key>\n\t<true/>\n")
	b.WriteString("\t<key>StandardOutPath</key>\n")
	fmt.Fprintf(&b, "\t<string>%s</string>\n", xmlEscape(p.LogOut))
	b.WriteString("\t<key>StandardErrorPath</key>\n")
	fmt.Fprintf(&b, "\t<string>%s</string>\n", xmlEscape(p.LogErr))
	b.WriteString("</dict>\n")
	b.WriteString("</plist>\n")
	return b.String()
}

// xmlEscape escapes the five characters that matter inside a plist
// <string>. Unicode passes through untouched.
func xmlEscape(s string) string {
	r := strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		"\"", "&quot;",
		"'", "&apos;",
	)
	return r.Replace(s)
}

// shellEscape wraps a path in double quotes when it contains spaces
// so the systemd ExecStart line keeps its argv intact. Paths without
// spaces are returned unchanged so the unit file stays readable.
func shellEscape(s string) string {
	if !strings.ContainsAny(s, " \t\"\\") {
		return s
	}
	return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
}

// ensureLogsDir creates the log directory referenced by the unit
// file. Best-effort: failures here only mean the systemd/launchd
// process will create it (or refuse to start); we surface them
// as warnings via the caller's logger.
func ensureLogsDir(p serviceParams) error {
	return os.MkdirAll(filepath.Dir(p.LogOut), 0o755)
}
