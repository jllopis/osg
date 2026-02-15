package tui

import "github.com/charmbracelet/lipgloss"

// Nord palette — Polar Night (backgrounds)
var (
	nordBg0 = lipgloss.Color("#2e3440")
	nordBg1 = lipgloss.Color("#3b4252")
	nordBg2 = lipgloss.Color("#434c5e")
	nordBg3 = lipgloss.Color("#4c566a")
)

// Nord palette — Snow Storm (text)
var (
	nordFg0 = lipgloss.Color("#d8dee9")
	nordFg1 = lipgloss.Color("#e5e9f0")
	nordFg2 = lipgloss.Color("#eceff4")
)

// Nord palette — Frost (accents)
var (
	nordFrost0 = lipgloss.Color("#8fbcbb")
	nordFrost1 = lipgloss.Color("#88c0d0")
	nordFrost2 = lipgloss.Color("#81a1c1")
	nordFrost3 = lipgloss.Color("#5e81ac")
)

// Nord palette — Aurora (status)
var (
	nordRed    = lipgloss.Color("#bf616a")
	nordOrange = lipgloss.Color("#d08770")
	nordYellow = lipgloss.Color("#ebcb8b")
	nordGreen  = lipgloss.Color("#a3be8c")
	nordPurple = lipgloss.Color("#b48ead")
)

// Semantic aliases.
var (
	colorPrimary = nordFrost1
	colorAccent  = nordFrost2
	colorMuted   = nordBg3
	colorSuccess = nordGreen
	colorWarning = nordYellow
	colorError   = nordRed
	colorInfo    = nordFrost0
	colorSpecial = nordPurple
)

// ---- Reusable styles ----

// Header bar: dark background, bright text.
var headerStyle = lipgloss.NewStyle().
	Background(nordBg1).
	Foreground(nordFg2).
	Bold(true).
	Padding(0, 1)

// Sidebar panel: subtle border.
var sidebarStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(nordBg2).
	Padding(0, 1)

// Sidebar section title.
var sectionTitleStyle = lipgloss.NewStyle().
	Foreground(nordFrost1).
	Bold(true)

// Sidebar key-value label (left).
var sidebarLabelStyle = lipgloss.NewStyle().
	Foreground(nordBg3)

// Sidebar key-value value (right).
var sidebarValueStyle = lipgloss.NewStyle().
	Foreground(nordFg0)

// Input prompt.
var promptStyle = lipgloss.NewStyle().
	Foreground(nordFrost2).
	Bold(true)

// Hint bar (bottom).
var hintBarStyle = lipgloss.NewStyle().
	Background(nordBg1).
	Foreground(nordBg3).
	Padding(0, 1)

// Hint keys.
var hintKeyStyle = lipgloss.NewStyle().
	Foreground(nordFg0).
	Bold(true)

// Hint descriptions.
var hintDescStyle = lipgloss.NewStyle().
	Foreground(nordBg3)

// Log labels.
var (
	labelSYS      = lipgloss.NewStyle().Foreground(nordFrost1)
	labelINFO     = lipgloss.NewStyle().Foreground(nordFrost0)
	labelWARN     = lipgloss.NewStyle().Foreground(nordYellow)
	labelERROR    = lipgloss.NewStyle().Foreground(nordRed)
	labelDEBUG    = lipgloss.NewStyle().Foreground(nordBg3)
	labelPROGRESS = lipgloss.NewStyle().Foreground(nordOrange)
)

// Timestamp in log lines.
var timestampStyle = lipgloss.NewStyle().Foreground(nordBg3)

// Workflow step styles.
var (
	stepDoneStyle    = lipgloss.NewStyle().Foreground(nordGreen)
	stepRunStyle     = lipgloss.NewStyle().Foreground(nordOrange)
	stepPendingStyle = lipgloss.NewStyle().Foreground(nordBg3)
)

// Badge: small colored inline tag.
func badge(text string, fg lipgloss.Color, bg lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(fg).
		Background(bg).
		Padding(0, 1).
		Bold(true).
		Render(text)
}

// serveBadge returns a RUNNING/STOPPED badge.
func serveBadge(running bool) string {
	if running {
		return badge("RUNNING", nordBg0, nordGreen)
	}
	return badge("STOPPED", nordFg0, nordBg2)
}

// Autocomplete popup styles.
var (
	acPopupStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(nordBg2).Padding(0, 1)
	acSelectedStyle = lipgloss.NewStyle().Foreground(nordFg2).Background(nordBg2).Bold(true)
	acNormalStyle   = lipgloss.NewStyle().Foreground(nordFg0)
	acHintStyle     = lipgloss.NewStyle().Foreground(nordBg3)
)

// labelStyle returns the style for a given log label.
func labelStyle(label string) lipgloss.Style {
	switch label {
	case "SYS":
		return labelSYS
	case "INFO":
		return labelINFO
	case "WARN", "WARNING":
		return labelWARN
	case "ERROR":
		return labelERROR
	case "DEBUG":
		return labelDEBUG
	case "PROGRESS":
		return labelPROGRESS
	default:
		return labelINFO
	}
}
