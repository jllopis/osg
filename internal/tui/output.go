package tui

import (
	"fmt"
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
	}

	line := fmt.Sprintf("  %s %s %s", stamp, label, text)

	// Add structured field detail for exported messages.
	if msg.Text == "exported" {
		source := asString(msg.Fields["source"])
		if source != "" {
			detail := lipgloss.NewStyle().Foreground(nordBg3).Render(
				fmt.Sprintf("           from %s", source))
			return []string{line, detail}
		}
	}

	return []string{line}
}
