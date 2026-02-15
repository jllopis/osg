package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// renderSidebar renders the collapsible right-side panel.
func (m Model) renderSidebar(height int) string {
	width := sidebarWidth - 2 // account for border
	if width < 10 {
		width = 10
	}

	style := sidebarStyle.
		Width(width).
		Height(height)

	var sections []string

	// -- Project section --
	sections = append(sections, sectionTitleStyle.Render("  Project"))
	sections = append(sections, kvLine("  config", defaultConfig(m.options.ConfigPath), width))
	sections = append(sections, kvLine("  vault", defaultValue(m.options.VaultPath), width))
	sections = append(sections, kvLine("  public", defaultValue(m.options.PublicDir), width))
	if m.options.SiteTitle != "" {
		sections = append(sections, kvLine("  title", m.options.SiteTitle, width))
	}
	sections = append(sections, "")

	// -- Workflow section --
	sections = append(sections, sectionTitleStyle.Render("  Workflow"))
	for _, step := range m.steps {
		sections = append(sections, "  "+formatStepLine(step, m.spinner.View(), time.Now()))
	}
	sections = append(sections, "")

	// -- Serve section --
	sections = append(sections, sectionTitleStyle.Render("  Serve"))
	sections = append(sections, fmt.Sprintf("  %s  %s", serveBadge(m.serveRunning), defaultAddr(m.options.ServeAddr)))
	sections = append(sections, "")

	// -- Plugins section --
	sections = append(sections, sectionTitleStyle.Render("  Plugins"))
	if len(m.options.Plugins) == 0 && len(m.enabledPlugins) == 0 {
		sections = append(sections, sidebarLabelStyle.Render("  none"))
	} else {
		installed := map[string]bool{}
		for _, p := range m.options.Plugins {
			installed[p] = true
			icon := lipgloss.NewStyle().Foreground(nordBg3).Render("  off")
			if m.enabledPlugins[p] {
				icon = lipgloss.NewStyle().Foreground(nordGreen).Render("  on ")
			}
			sections = append(sections, fmt.Sprintf("  %s %s", icon, p))
		}
		for name := range m.enabledPlugins {
			if installed[name] {
				continue
			}
			sections = append(sections, fmt.Sprintf("  %s %s",
				lipgloss.NewStyle().Foreground(nordRed).Render("  !!"),
				name))
		}
	}

	// -- Build stats --
	if m.lastBuild != nil {
		sections = append(sections, "")
		sections = append(sections, sectionTitleStyle.Render("  Last Build"))
		sections = append(sections, kvLine("  pages", fmt.Sprintf("%d", m.lastBuild.Total), width))
		sections = append(sections, kvLine("  rendered", fmt.Sprintf("%d", m.lastBuild.Rendered), width))
		sections = append(sections, kvLine("  cached", fmt.Sprintf("%d", m.lastBuild.Cached), width))
		if m.lastBuild.Errors > 0 {
			sections = append(sections, kvLine("  errors",
				lipgloss.NewStyle().Foreground(nordRed).Render(fmt.Sprintf("%d", m.lastBuild.Errors)), width))
		}
	}

	if m.lastDoctor != nil {
		sections = append(sections, "")
		sections = append(sections, sectionTitleStyle.Render("  Doctor"))
		sections = append(sections, kvLine("  warnings", fmt.Sprintf("%d", m.lastDoctor.Warnings), width))
		sections = append(sections, kvLine("  errors", fmt.Sprintf("%d", m.lastDoctor.Errors), width))
	}

	return style.Render(strings.Join(sections, "\n"))
}

// formatStepLine renders a workflow step with icon and optional duration.
func formatStepLine(step Step, spin string, now time.Time) string {
	var icon string
	var style lipgloss.Style

	switch step.Status {
	case StepDone:
		icon = "+"
		style = stepDoneStyle
	case StepRunning:
		if spin != "" {
			icon = spin
		} else {
			icon = ">"
		}
		style = stepRunStyle
	default:
		icon = "o"
		style = stepPendingStyle
	}

	suffix := ""
	if step.Status == StepRunning && !step.Start.IsZero() {
		suffix = " " + formatDuration(now.Sub(step.Start))
	} else if step.Last > 0 {
		suffix = " " + formatDuration(step.Last)
	}

	return fmt.Sprintf("%s %s%s", style.Render(icon), step.Name, suffix)
}

// kvLine renders a key: value pair, truncated to fit width.
func kvLine(key string, value string, width int) string {
	k := sidebarLabelStyle.Render(key + ":")
	v := sidebarValueStyle.Render(" " + value)
	line := k + v
	// Lipgloss Width accounts for ANSI, but simple truncation is fine for sidebar.
	if lipgloss.Width(line) > width {
		// Truncate value portion.
		maxVal := width - lipgloss.Width(k) - 2
		if maxVal < 3 {
			maxVal = 3
		}
		if len(value) > maxVal {
			value = value[:maxVal-1] + "~"
		}
		v = sidebarValueStyle.Render(" " + value)
		line = k + v
	}
	return line
}
