package tui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// renderHeader renders a 1-line compact header bar spanning the full width.
func (m Model) renderHeader() string {
	style := headerStyle.Width(m.width)

	// Left: OSG + site title.
	left := lipgloss.NewStyle().Bold(true).Foreground(nordFrost1).Render("OSG")
	if m.options.SiteTitle != "" {
		left += "  " + lipgloss.NewStyle().Foreground(nordFg0).Render(m.options.SiteTitle)
	}

	// Center/right: serve badge + API badge + build stats.
	var right string
	if m.serveRunning {
		right = serveBadge(true) + " " + defaultAddr(m.options.ServeAddr)
		if m.serveMode == "api" {
			right += "  " + lipgloss.NewStyle().Foreground(nordFrost2).Render("+api")
		}
	}
	if m.apiRunning {
		if right != "" {
			right += "  "
		}
		right += apiBadge(true) + " " + defaultAPIAddr(m.options.APIAddr)
	}

	if m.lastBuild != nil {
		stats := fmt.Sprintf("build: %d pages", m.lastBuild.Total)
		if m.lastBuild.Errors > 0 {
			stats += fmt.Sprintf(", %d errors", m.lastBuild.Errors)
		}
		if right != "" {
			right += "  |  "
		}
		right += lipgloss.NewStyle().Foreground(nordFg0).Render(stats)
	}

	if m.hasRunning() {
		spinner := m.spinner.View()
		if right != "" {
			right = spinner + " " + right
		} else {
			right = spinner
		}
	}

	// Fill the gap between left and right.
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right) - 4 // padding
	if gap < 1 {
		gap = 1
	}
	filler := lipgloss.NewStyle().Render(spaces(gap))

	return style.Render(left + filler + right)
}

// spaces returns a string of n space characters.
func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
