package tui

import "github.com/charmbracelet/lipgloss"

type hint struct{ key, desc string }

// renderHintBar renders the bottom hint/status bar.
// The hints change based on the current mode: normal, log panel focused,
// or running tasks/services.
func (m Model) renderHintBar() string {
	style := hintBarStyle.Width(m.width)

	var hints []hint

	modPrefix := logModHintPrefix(m)

	if m.logFocus {
		// Log panel has focus — show navigation hints.
		hints = []hint{
			{modPrefix + "+↑↓", "scroll"},
			{modPrefix + "+←→", "tab"},
			{"F7", "close"},
			{"Esc", "unfocus"},
		}
	} else {
		// Normal mode — base hints.
		hints = []hint{
			{"/help", "commands"},
			{"ctrl+c", "quit"},
			{"tab", "sidebar"},
		}

		// Service toggles with running status.
		if m.serveRunning {
			hints = append(hints, hint{"F5", "stop serve"})
		} else {
			hints = append(hints, hint{"F5", "serve"})
		}
		if m.apiRunning {
			hints = append(hints, hint{"F6", "stop api"})
		} else {
			hints = append(hints, hint{"F6", "api"})
		}

		hints = append(hints, hint{"F7", "logs"})
		hints = append(hints, hint{"F8", "config"})

		if m.logPanel.Visible() {
			hints = append(hints, hint{modPrefix + "+↑↓", "log scroll"})
			hints = append(hints, hint{modPrefix + "+←→", "log tab"})
		}
	}

	var parts []string
	for _, h := range hints {
		parts = append(parts,
			hintKeyStyle.Render(h.key)+" "+hintDescStyle.Render(h.desc))
	}

	// Status on the right side.
	status := m.statusText()
	statusText := lipgloss.NewStyle().Foreground(nordFrost2).Render(status)

	left := ""
	for i, p := range parts {
		if i > 0 {
			left += "  "
		}
		left += p
	}

	gap := m.width - lipgloss.Width(left) - lipgloss.Width(statusText) - 4
	if gap < 1 {
		gap = 1
	}

	return style.Render(left + spaces(gap) + statusText)
}

// statusText returns the right-side status string.
func (m Model) statusText() string {
	var parts []string

	if m.serveRunning {
		mode := "static"
		if m.serveMode == "api" {
			mode = "serve+api"
		}
		parts = append(parts, mode+" "+m.options.ServeAddr)
	}
	if m.apiRunning {
		parts = append(parts, "api "+defaultAPIAddr(m.options.APIAddr))
	}

	if len(parts) > 0 {
		return joinStr(parts, " | ")
	}

	if m.lastAction != "" {
		return m.lastAction
	}
	return "idle"
}

// joinStr joins strings with a separator.
func joinStr(parts []string, sep string) string {
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += sep
		}
		result += p
	}
	return result
}

// logModHintPrefix returns the short hint-bar prefix for the configured
// log modifier key: "A" for alt, "S" for shift.
func logModHintPrefix(m Model) string {
	switch m.options.LogModifier {
	case "alt":
		return "A"
	default:
		return "S"
	}
}
