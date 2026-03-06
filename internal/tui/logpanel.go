package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
)

// LogTab identifies which tab is active in the log panel.
type LogTab int

const (
	LogTabServe LogTab = iota
	LogTabAPI
	LogTabAll
)

// LogPanel is a self-contained bottom panel that shows serve/API/all logs.
type LogPanel struct {
	visible  bool
	tab      LogTab
	viewport viewport.Model
	ready    bool
	width    int
	height   int // full panel height including border/title
}

// NewLogPanel creates a log panel. It starts hidden.
func NewLogPanel() LogPanel {
	return LogPanel{
		tab: LogTabAll,
	}
}

// Visible returns whether the panel is shown.
func (lp LogPanel) Visible() bool { return lp.visible }

// Toggle flips visibility.
func (lp *LogPanel) Toggle() {
	lp.visible = !lp.visible
}

// SetTab changes the active tab.
func (lp *LogPanel) SetTab(t LogTab) {
	lp.tab = t
}

// Tab returns the active tab.
func (lp LogPanel) Tab() LogTab { return lp.tab }

// NextTab cycles to the next tab.
func (lp *LogPanel) NextTab() {
	lp.tab = (lp.tab + 1) % 3
}

// PrevTab cycles to the previous tab.
func (lp *LogPanel) PrevTab() {
	lp.tab = (lp.tab + 2) % 3 // +2 mod 3 == -1
}

// Resize recalculates the internal viewport dimensions.
// panelHeight is the total height allocated to this panel (including title/border).
// width is the full terminal width.
func (lp *LogPanel) Resize(width, panelHeight int) {
	lp.width = width
	lp.height = panelHeight

	// Viewport height = panel height minus 1 for the title/tab bar.
	vpHeight := panelHeight - 1
	if vpHeight < 1 {
		vpHeight = 1
	}
	vpWidth := width - 2 // border/padding
	if vpWidth < 10 {
		vpWidth = 10
	}

	if !lp.ready {
		lp.viewport = viewport.New(vpWidth, vpHeight)
		lp.ready = true
	} else {
		lp.viewport.Width = vpWidth
		lp.viewport.Height = vpHeight
	}
}

// SetContent updates the viewport content from the given messages.
func (lp *LogPanel) SetContent(msgs []Message) {
	if !lp.ready {
		return
	}
	lp.viewport.SetContent(renderLogMessages(msgs, lp.viewport.Width))
	lp.viewport.GotoBottom()
}

// ScrollUp scrolls the log viewport up.
func (lp *LogPanel) ScrollUp(n int) { lp.viewport.ScrollUp(n) }

// ScrollDown scrolls the log viewport down.
func (lp *LogPanel) ScrollDown(n int) { lp.viewport.ScrollDown(n) }

// View renders the entire log panel (title bar + viewport).
func (lp LogPanel) View() string {
	if !lp.visible || !lp.ready {
		return ""
	}

	titleBar := lp.renderTitleBar()
	vpContent := lp.viewport.View()

	return lipgloss.JoinVertical(lipgloss.Left, titleBar, vpContent)
}

// PanelHeight computes how tall the log panel should be (roughly 1/3 of terminal height).
func PanelHeight(termHeight int) int {
	h := termHeight / 3
	if h < 4 {
		h = 4
	}
	if h > 20 {
		h = 20
	}
	return h
}

// ---- Internal rendering ----

func (lp LogPanel) renderTitleBar() string {
	tabs := []struct {
		label string
		tab   LogTab
	}{
		{"Serve", LogTabServe},
		{"API", LogTabAPI},
		{"All", LogTabAll},
	}

	var parts []string
	for _, t := range tabs {
		if t.tab == lp.tab {
			parts = append(parts, logTabActiveStyle.Render(" "+t.label+" "))
		} else {
			parts = append(parts, logTabInactiveStyle.Render(" "+t.label+" "))
		}
	}

	tabStr := strings.Join(parts, " ")
	title := logPanelTitleStyle.Render(" Logs ")

	// Scroll indicator.
	scrollInfo := ""
	if lp.ready {
		pct := lp.viewport.ScrollPercent()
		if pct < 1.0 {
			scrollInfo = logPanelScrollStyle.Render(fmt.Sprintf(" %d%% ", int(pct*100)))
		}
	}

	// Fill remaining width with a separator line.
	used := lipgloss.Width(title) + lipgloss.Width(tabStr) + lipgloss.Width(scrollInfo) + 4
	gap := lp.width - used
	if gap < 0 {
		gap = 0
	}
	separator := logPanelSepStyle.Render(strings.Repeat("─", gap))

	return lipgloss.NewStyle().Width(lp.width).Render(
		title + " " + tabStr + " " + separator + scrollInfo,
	)
}

// renderLogMessages formats a slice of messages for the log panel viewport.
func renderLogMessages(msgs []Message, width int) string {
	if len(msgs) == 0 {
		return logPanelEmptyStyle.Render("  No log entries")
	}
	var lines []string
	for _, msg := range msgs {
		ts := timestampStyle.Render(msg.Time.Format("15:04:05"))
		label := labelStyle(msg.Label).Render(fmt.Sprintf("%-5s", msg.Label))
		text := msg.Text

		// Truncate long lines.
		maxText := width - 16 // timestamp + label + spacing
		if maxText < 20 {
			maxText = 20
		}
		if len(text) > maxText {
			text = text[:maxText-1] + "~"
		}

		lines = append(lines, fmt.Sprintf(" %s %s %s", ts, label, text))
	}
	return strings.Join(lines, "\n")
}

// MessagesForTab returns the appropriate message slice for the given tab.
func MessagesForTab(tab LogTab, serveMessages, apiMessages, allMessages []Message) []Message {
	switch tab {
	case LogTabServe:
		return serveMessages
	case LogTabAPI:
		return apiMessages
	default:
		return allMessages
	}
}
