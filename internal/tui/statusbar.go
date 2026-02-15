package tui

import "github.com/charmbracelet/lipgloss"

// renderHintBar renders the bottom hint/status bar.
func (m Model) renderHintBar() string {
	style := hintBarStyle.Width(m.width)

	hints := []struct{ key, desc string }{
		{"/help", "commands"},
		{"ctrl+c", "quit"},
		{"tab", "sidebar"},
		{"ctrl+l", "clear"},
		{"up/down", "scroll"},
	}

	var parts []string
	for _, h := range hints {
		parts = append(parts,
			hintKeyStyle.Render(h.key)+" "+hintDescStyle.Render(h.desc))
	}

	// Status on the right side.
	status := "idle"
	if m.lastAction != "" {
		status = m.lastAction
	}
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
