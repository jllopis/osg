package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model.
func (m Model) View() string {
	if m.width == 0 {
		return "OSG loading..."
	}

	// Config editor takes over the full screen when active.
	if m.configActive && m.configScreen != nil {
		return m.configScreen.View()
	}

	header := m.renderHeader()
	hintBar := m.renderHintBar()

	// Body height = total - header - input line - hint bar.
	bodyHeight := m.height - lipgloss.Height(header) - 1 - lipgloss.Height(hintBar)
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// Main viewport.
	vpView := m.viewport.View()

	// Sidebar.
	var body string
	if m.sidebarVisible {
		sidebar := m.renderSidebar(bodyHeight)
		body = lipgloss.JoinHorizontal(lipgloss.Top, vpView, sidebar)
	} else {
		body = vpView
	}

	// Input line.
	inputLine := m.renderInputLine()

	// Log panel (between body and input when visible).
	logView := m.logPanel.View()

	// Autocomplete overlay (appears above input if visible).
	if m.acVisible && len(m.acMatches) > 0 {
		acPopup := m.renderAutocomplete()
		if logView != "" {
			return lipgloss.JoinVertical(lipgloss.Left,
				header, body, logView, acPopup, inputLine, hintBar)
		}
		return lipgloss.JoinVertical(lipgloss.Left,
			header, body, acPopup, inputLine, hintBar)
	}

	if logView != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			header, body, logView, inputLine, hintBar)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header, body, inputLine, hintBar)
}

// renderInputLine renders the command input line.
func (m Model) renderInputLine() string {
	return lipgloss.NewStyle().
		Width(m.width).
		Padding(0, 1).
		Render(m.input.View())
}

// renderAutocomplete renders the autocomplete popup.
func (m Model) renderAutocomplete() string {
	maxShow := 8
	matches := m.acMatches
	if len(matches) > maxShow {
		matches = matches[:maxShow]
	}

	var lines []string
	for i, cmd := range matches {
		name := cmd.Name
		hint := acHintStyle.Render("  " + cmd.Hint)
		if i == m.acSelected {
			lines = append(lines, acSelectedStyle.Render(fmt.Sprintf(" %-12s", name))+hint)
		} else {
			lines = append(lines, acNormalStyle.Render(fmt.Sprintf(" %-12s", name))+hint)
		}
	}
	if len(m.acMatches) > maxShow {
		lines = append(lines, acHintStyle.Render(fmt.Sprintf("  ... and %d more", len(m.acMatches)-maxShow)))
	}

	content := strings.Join(lines, "\n")
	popup := acPopupStyle.Render(content)

	// Left-pad to align with input prompt.
	return lipgloss.NewStyle().PaddingLeft(1).Render(popup)
}
