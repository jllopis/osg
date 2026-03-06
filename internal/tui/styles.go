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
	nordFg2 = lipgloss.Color("#eceff4")
)

// Nord palette — Frost (accents)
var (
	nordFrost0 = lipgloss.Color("#8fbcbb")
	nordFrost1 = lipgloss.Color("#88c0d0")
	nordFrost2 = lipgloss.Color("#81a1c1")
)

// Nord palette — Aurora (status)
var (
	nordRed    = lipgloss.Color("#bf616a")
	nordOrange = lipgloss.Color("#d08770")
	nordYellow = lipgloss.Color("#ebcb8b")
	nordGreen  = lipgloss.Color("#a3be8c")
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
	Foreground(nordFrost2)

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
	Foreground(nordFrost2)

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

// serveBadge returns a RUNNING/STOPPED badge for the dev server.
func serveBadge(running bool) string {
	if running {
		return badge("RUNNING", nordBg0, nordGreen)
	}
	return badge("STOPPED", nordFg0, nordBg2)
}

// apiBadge returns a RUNNING/STOPPED badge for the API server.
func apiBadge(running bool) string {
	if running {
		return badge("RUNNING", nordBg0, nordFrost1)
	}
	return badge("STOPPED", nordFg0, nordBg2)
}

// serveModeName returns a human-readable label for the serve mode.
func serveModeName(mode string) string {
	switch mode {
	case "api":
		return "static+api"
	default:
		return "static"
	}
}

// Autocomplete popup styles.
var (
	acPopupStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(nordBg2).Padding(0, 1)
	acSelectedStyle = lipgloss.NewStyle().Foreground(nordFg2).Background(nordBg2).Bold(true)
	acNormalStyle   = lipgloss.NewStyle().Foreground(nordFg0)
	acHintStyle     = lipgloss.NewStyle().Foreground(nordBg3)
)

// Log panel styles.
var (
	logTabActiveStyle   = lipgloss.NewStyle().Foreground(nordFg2).Background(nordBg2).Bold(true)
	logTabInactiveStyle = lipgloss.NewStyle().Foreground(nordBg3)
	logPanelTitleStyle  = lipgloss.NewStyle().Foreground(nordFrost1).Bold(true)
	logPanelSepStyle    = lipgloss.NewStyle().Foreground(nordBg2)
	logPanelScrollStyle = lipgloss.NewStyle().Foreground(nordBg3)
	logPanelEmptyStyle  = lipgloss.NewStyle().Foreground(nordBg3).Italic(true)
)

// Config screen styles.
var (
	cfgHeaderStyle = lipgloss.NewStyle().
			Background(nordBg1).
			Foreground(nordFg2).
			Bold(true).
			Padding(0, 1)

	cfgDirtyStyle = lipgloss.NewStyle().
			Foreground(nordOrange).
			Bold(true)

	cfgSectionListStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(nordBg2).
				Padding(0, 1)

	cfgSectionActiveStyle = lipgloss.NewStyle().
				Foreground(nordFg2).
				Background(nordBg2).
				Bold(true)

	cfgSectionNormalStyle = lipgloss.NewStyle().
				Foreground(nordFg0)

	cfgFieldPanelStyle = lipgloss.NewStyle().
				Padding(0, 1)

	cfgFieldLabelStyle = lipgloss.NewStyle().
				Foreground(nordFrost2).
				Bold(true)

	cfgFieldValueStyle = lipgloss.NewStyle().
				Foreground(nordFg0)

	cfgFieldDescStyle = lipgloss.NewStyle().
				Foreground(nordBg3).
				Italic(true)

	cfgFieldSelectedStyle = lipgloss.NewStyle().
				Foreground(nordFg2).
				Background(nordBg2)

	cfgFieldEditingStyle = lipgloss.NewStyle().
				Foreground(nordFg2).
				Background(nordFrost2)

	cfgSensitiveStyle = lipgloss.NewStyle().
				Foreground(nordBg3)

	cfgHintBarStyle = lipgloss.NewStyle().
			Background(nordBg1).
			Foreground(nordBg3).
			Padding(0, 1)

	cfgConfirmStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(nordOrange).
			Padding(1, 2).
			Foreground(nordFg2).
			Background(nordBg1)

	cfgListItemStyle = lipgloss.NewStyle().
				Foreground(nordFg0)

	cfgListSelectedStyle = lipgloss.NewStyle().
				Foreground(nordFg2).
				Background(nordBg2)

	cfgStructHeaderStyle = lipgloss.NewStyle().
				Foreground(nordFrost1).
				Bold(true)
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
