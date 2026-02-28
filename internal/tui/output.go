package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// renderOutput builds the full text content for the scrollable viewport.
func (m Model) renderOutput() string {
	if len(m.messages) == 0 {
		return timestampStyle.Render("  No output yet. Type /help for commands.")
	}

	var b strings.Builder
	for _, msg := range m.messages {
		lines := formatOutputLine(msg)
		for _, line := range lines {
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

// formatOutputLine renders a single Message as one or more display lines.
func formatOutputLine(msg Message) []string {
	stamp := timestampStyle.Render(msg.Time.Format("15:04:05"))
	label := labelStyle(msg.Label).Render(fmt.Sprintf("%-8s", msg.Label))

	text := msg.Text
	// Enrich known structured messages.
	switch msg.Text {
	case "exported":
		dest := asString(msg.Fields["dest"])
		if dest != "" {
			text = "Exported -> " + dest
		}
	case "update-content summary":
		text = fmt.Sprintf("Update content: exported %d, skipped %d, drafts %d, errors %d",
			asInt(msg.Fields["exported"]),
			asInt(msg.Fields["skipped"]),
			asInt(msg.Fields["drafts"]),
			asInt(msg.Fields["errors"]),
		)
	case "build incremental":
		mode := asString(msg.Fields["mode"])
		if mode == "" {
			mode = "partial"
		}
		text = fmt.Sprintf("Build incremental: %s (changed %d, removed %d)",
			mode, asInt(msg.Fields["changed"]), asInt(msg.Fields["removed"]))
	case "build summary":
		text = fmt.Sprintf("Build: rendered %d, cached %d, errors %d",
			asInt(msg.Fields["rendered"]),
			asInt(msg.Fields["cached"]),
			asInt(msg.Fields["errors"]),
		)
	case "initial build complete":
		text = "Initial build complete"
	case "watch enabled":
		text = fmt.Sprintf("Watch enabled (debounce %dms, live reload %s)",
			asInt(msg.Fields["debounce_ms"]),
			asString(msg.Fields["live_reload"]),
		)

	// ---- Doctor messages ----
	case "doctor starting":
		profile := asString(msg.Fields["profile"])
		if profile != "" {
			text = fmt.Sprintf("Doctor starting (profile: %s)", profile)
		}
	case "doctor summary":
		text = fmt.Sprintf("Doctor summary: %d warning(s), %d error(s)",
			asInt(msg.Fields["warnings"]),
			asInt(msg.Fields["errors"]),
		)
	case "broken wikilink":
		target := asString(msg.Fields["target"])
		ref := asString(msg.Fields["referenced_by"])
		text = fmt.Sprintf("Broken wikilink: [[%s]]  (in %s)", target, ref)
	case "optional template not found (built-in fallback used)":
		tmpl := asString(msg.Fields["template"])
		if tmpl != "" {
			text = fmt.Sprintf("Optional template not found: %s (built-in fallback used)", tmpl)
		}
	case "required template missing":
		tmpl := asString(msg.Fields["template"])
		text = fmt.Sprintf("Required template missing: %s", tmpl)
	case "empty section (no .md files)":
		section := asString(msg.Fields["section"])
		path := asString(msg.Fields["path"])
		text = fmt.Sprintf("Empty section: %s (%s)", section, path)
	case "large image file":
		path := asString(msg.Fields["path"])
		size := asString(msg.Fields["size_mb"])
		text = fmt.Sprintf("Large image: %s (%s MB)", path, size)
	case "plugin enabled but not installed":
		plugin := asString(msg.Fields["plugin"])
		text = fmt.Sprintf("Plugin enabled but not installed: %s", plugin)
	case "base_url is empty":
		text = "base_url is empty (required for production deployments)"
	case "base_url invalid":
		errMsg := asString(msg.Fields["error"])
		text = fmt.Sprintf("base_url invalid: %s", errMsg)
	case "theme not found":
		themeName := asString(msg.Fields["theme"])
		path := asString(msg.Fields["path"])
		text = fmt.Sprintf("Theme not found: %s (expected at %s)", themeName, path)
	case "theme templates not found":
		path := asString(msg.Fields["path"])
		text = fmt.Sprintf("Theme templates not found: %s", path)
	case "theme.yaml parse error":
		themeName := asString(msg.Fields["theme"])
		errMsg := asString(msg.Fields["error"])
		text = fmt.Sprintf("theme.yaml parse error in %s: %s", themeName, errMsg)
	case "theme parent chain error":
		themeName := asString(msg.Fields["theme"])
		errMsg := asString(msg.Fields["error"])
		text = fmt.Sprintf("Theme parent chain error in %s: %s", themeName, errMsg)
	case "taxonomy paginate_by should be > 0":
		name := asString(msg.Fields["name"])
		text = fmt.Sprintf("Taxonomy '%s': paginate_by should be > 0", name)
	case "taxonomy paginate_path is empty":
		name := asString(msg.Fields["name"])
		text = fmt.Sprintf("Taxonomy '%s': paginate_path is empty", name)
	case "duplicate taxonomies":
		names := asString(msg.Fields["names"])
		text = fmt.Sprintf("Duplicate taxonomies: %s", names)
	case "unknown summary_strategy":
		val := asString(msg.Fields["value"])
		text = fmt.Sprintf("Unknown summary_strategy: %s", val)
	case "serve_live_reload enabled but serve_watch is false":
		text = "serve_live_reload enabled but serve_watch is false"
	case "serve_debounce_ms should be > 0":
		text = "serve_debounce_ms should be > 0 when serve_watch is enabled"
	case "sass binary not found in PATH":
		text = "sass binary not found in PATH (compile_sass is enabled)"
	}

	line := fmt.Sprintf("  %s %s %s", stamp, label, text)
	lines := []string{line}

	// Add structured field detail for exported messages.
	if msg.Text == "exported" {
		source := asString(msg.Fields["source"])
		if source != "" {
			detail := lipgloss.NewStyle().Foreground(nordBg3).Render(
				fmt.Sprintf("           from %s", source))
			lines = append(lines, detail)
		}
	}

	// Add "fix" hint for doctor messages (WARN/ERROR with a fix field).
	if fix := asString(msg.Fields["fix"]); fix != "" {
		fixLine := lipgloss.NewStyle().Foreground(nordFrost0).Render(
			fmt.Sprintf("                    fix: %s", fix))
		lines = append(lines, fixLine)
	}

	// For messages with extra fields not already rendered inline, append
	// them as detail lines.  Skip fields already consumed above.
	if isDoctorDetailMessage(msg.Text) {
		details := formatExtraFields(msg.Fields)
		for _, d := range details {
			detailLine := lipgloss.NewStyle().Foreground(nordBg3).Render(
				fmt.Sprintf("                    %s", d))
			lines = append(lines, detailLine)
		}
	}

	return lines
}

// isDoctorDetailMessage returns true for doctor messages whose extra fields
// should be shown as detail lines (path does not exist, plugins_enabled, etc.).
func isDoctorDetailMessage(text string) bool {
	switch text {
	case "plugins_enabled set but plugins_dir is empty",
		"plugins_enabled set but plugins_dir missing",
		"no taxonomies configured",
		"taxonomy name is empty",
		"theme is empty",
		"theme.yaml not found (optional, recommended for metadata)":
		return true
	}
	// Generic: any message ending with "does not exist".
	return strings.HasSuffix(text, "does not exist")
}

// doctorConsumedFields lists field keys that are already rendered inline
// by the switch cases above or by the fix-line logic.
var doctorConsumedFields = map[string]bool{
	"fix": true, "target": true, "referenced_by": true,
	"template": true, "section": true, "path": true,
	"size_mb": true, "plugin": true, "theme": true,
	"error": true, "name": true, "names": true, "value": true,
	"profile": true, "warnings": true, "errors": true,
	// build/update/serve fields already handled:
	"dest": true, "source": true, "exported": true, "skipped": true,
	"drafts": true, "rendered": true, "cached": true, "changed": true,
	"removed": true, "mode": true, "debounce_ms": true, "live_reload": true,
}

// formatExtraFields returns display strings for fields not yet consumed inline.
func formatExtraFields(fields map[string]any) []string {
	var out []string
	keys := make([]string, 0, len(fields))
	for k := range fields {
		if !doctorConsumedFields[k] {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		out = append(out, fmt.Sprintf("%s: %v", k, fields[k]))
	}
	return out
}
