package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.recalcLayout()
		return m, nil

	case tea.KeyMsg:
		// Global keys (non-input).
		switch {
		case key.Matches(msg, keys.Quit):
			m.appendHistory("EXIT", "User quit")
			return m, tea.Quit

		case key.Matches(msg, keys.ToggleSidebar):
			m.sidebarVisible = !m.sidebarVisible
			m.recalcLayout()
			return m, nil

		case key.Matches(msg, keys.ClearOutput):
			m.messages = m.messages[:0]
			m.appendMessage("SYS", "Output cleared")
			return m, nil
		}

		// Autocomplete navigation takes priority over Submit when popup is visible.
		if m.acVisible {
			switch msg.String() {
			case "up":
				if m.acSelected > 0 {
					m.acSelected--
				}
				return m, nil
			case "down":
				if m.acSelected < len(m.acMatches)-1 {
					m.acSelected++
				}
				return m, nil
			case "esc":
				m.acVisible = false
				return m, nil
			case "tab", "enter":
				if len(m.acMatches) > 0 {
					selected := m.acMatches[m.acSelected]
					m.input.SetValue(selected.Name + " ")
					m.input.CursorEnd()
					m.acVisible = false
					if msg.String() == "enter" {
						return m.handleSubmit()
					}
				}
				return m, nil
			}
		}

		// Submit (Enter) — only reached when autocomplete is NOT visible.
		if key.Matches(msg, keys.Submit) {
			return m.handleSubmit()
		}

		// If input is focused, check autocomplete trigger.
		val := m.input.Value()
		if msg.String() == "/" && val == "" {
			// Trigger autocomplete.
			m.acVisible = true
			m.acMatches = matchCommands("/")
			m.acSelected = 0
		}

		// Scrolling viewport (when not in autocomplete).
		if !m.acVisible {
			switch {
			case key.Matches(msg, keys.ScrollUp):
				m.viewport.LineUp(1)
				return m, nil
			case key.Matches(msg, keys.ScrollDown):
				m.viewport.LineDown(1)
				return m, nil
			case key.Matches(msg, keys.PageUp):
				m.viewport.HalfViewUp()
				return m, nil
			case key.Matches(msg, keys.PageDown):
				m.viewport.HalfViewDown()
				return m, nil
			}
		}

		// Update autocomplete matches as user types.
		var inputCmd tea.Cmd
		m.input, inputCmd = m.input.Update(msg)
		cmds = append(cmds, inputCmd)
		m.updateAutocomplete()
		return m, tea.Batch(cmds...)

	case logLineMsg:
		m.appendParsedLog(msg.line)
		if m.logCh != nil {
			cmds = append(cmds, listenLogCmd(m.logCh))
		}
		return m, tea.Batch(cmds...)

	case taskFinishedMsg:
		return m.finishTask(msg)

	case pluginActionFinishedMsg:
		return m.finishPluginAction(msg)

	case simpleActionFinishedMsg:
		return m.finishSimpleAction(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}

	// Fallback: forward to input.
	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	cmds = append(cmds, inputCmd)
	return m, tea.Batch(cmds...)
}

// ---- Layout calculation ----

const sidebarWidth = 30
const minMainWidth = 30

func (m *Model) recalcLayout() {
	// Header = 1 line, input = 1 line, hint bar = 1 line, borders = 0
	// Viewport height = total - 3
	vpHeight := m.height - 3
	if vpHeight < 1 {
		vpHeight = 1
	}
	vpWidth := m.mainWidth()

	if !m.viewportReady {
		m.viewport = viewport.New(vpWidth, vpHeight)
		m.viewport.SetContent(m.renderOutput())
		m.viewport.GotoBottom()
		m.viewportReady = true
	} else {
		m.viewport.Width = vpWidth
		m.viewport.Height = vpHeight
		m.viewport.SetContent(m.renderOutput())
		m.viewport.GotoBottom()
	}

	// Input width matches main panel.
	iw := vpWidth - 4
	if iw < 10 {
		iw = 10
	}
	m.input.Width = iw
}

func (m Model) mainWidth() int {
	total := m.width
	if m.sidebarVisible {
		total -= sidebarWidth
	}
	if total < minMainWidth {
		return minMainWidth
	}
	return total
}

// ---- Autocomplete ----

func (m *Model) updateAutocomplete() {
	val := m.input.Value()
	if strings.HasPrefix(val, "/") {
		m.acMatches = matchCommands(val)
		m.acVisible = len(m.acMatches) > 0
		if m.acSelected >= len(m.acMatches) {
			m.acSelected = max(0, len(m.acMatches)-1)
		}
	} else {
		m.acVisible = false
	}
}

// ---- Submit / command handling ----

func (m Model) handleSubmit() (Model, tea.Cmd) {
	raw := strings.TrimSpace(m.input.Value())
	m.input.SetValue("")
	m.input.Focus()
	m.acVisible = false
	if raw == "" {
		return m, nil
	}

	m.appendHistory("CMD", raw)
	cmd, args := resolveCommand(raw)
	if cmd == "" {
		m.appendMessage("ERROR", fmt.Sprintf("Unknown command: %s", raw))
		return m, nil
	}

	switch cmd {
	case "init":
		return m.startTask(taskInit)
	case "update":
		return m.startTask(taskUpdate)
	case "build":
		return m.startTask(taskBuild)
	case "serve":
		return m.toggleServe()
	case "stop":
		if !m.serveRunning {
			m.appendMessage("INFO", "Server is not running")
			return m, nil
		}
		return m.toggleServe()
	case "doctor":
		return m.runSimpleAction("Doctor", m.actions.Doctor)
	case "next":
		return m.handleNext()
	case "theme":
		return m.handleThemeCommand(args)
	case "plugin":
		return m.handlePluginCommand(args)
	case "new":
		return m.handleNewCommand(args)
	case "version":
		return m.handleVersionCommand()
	case "clear":
		m.messages = m.messages[:0]
		m.appendMessage("SYS", "Output cleared")
		return m, nil
	case "help":
		m.appendMessage("INFO", helpText())
		return m, nil
	case "quit", "exit":
		m.appendHistory("EXIT", "User quit via command")
		return m, tea.Quit
	default:
		m.appendMessage("ERROR", fmt.Sprintf("Unknown command: %s", raw))
		return m, nil
	}
}

// ---- Task lifecycle ----

func (m Model) startTask(kind taskKind) (Model, tea.Cmd) {
	if kind == taskServe {
		return m.toggleServe()
	}

	idx := stepIndexForTask(kind)
	if idx >= 0 && idx < len(m.steps) {
		if m.steps[idx].Status == StepRunning {
			m.appendMessage("INFO", fmt.Sprintf("%s already running", m.steps[idx].Name))
			return m, nil
		}
		m.steps[idx].Status = StepRunning
		m.steps[idx].Start = time.Now()
		m.steps[idx].Last = 0
	}

	label := taskLabel(kind)
	m.lastAction = label + " started"
	m.appendMessage("PROGRESS", fmt.Sprintf("%s started", label))
	m.appendHistory("ACTION", fmt.Sprintf("%s started", label))

	action := actionForTask(m.actions, kind)
	if action == nil {
		m.appendMessage("ERROR", fmt.Sprintf("%s not available", label))
		return m, nil
	}

	cmd := runTaskCmd(context.Background(), action, kind)
	return m, tea.Batch(cmd, m.spinner.Tick)
}

func (m Model) toggleServe() (Model, tea.Cmd) {
	if m.serveRunning {
		if m.serveCancel != nil {
			m.serveCancel()
		}
		m.appendMessage("PROGRESS", "Stopping preview server...")
		m.appendHistory("ACTION", "Serve stop requested")
		m.lastAction = "Serve stop requested"
		return m, nil
	}

	action := m.actions.Serve
	if action == nil {
		m.appendMessage("ERROR", "Serve action not available")
		return m, nil
	}

	serveCtx, cancel := context.WithCancel(context.Background())
	m.serveCancel = cancel
	m.serveRunning = true
	m.setStepStatus(taskServe, StepRunning)
	m.setStepStart(taskServe, time.Now())
	m.appendMessage("PROGRESS", fmt.Sprintf("Starting preview server on %s...", defaultAddr(m.options.ServeAddr)))
	m.appendHistory("ACTION", fmt.Sprintf("Serve started on %s", defaultAddr(m.options.ServeAddr)))
	m.lastAction = "Serve started"

	cmd := runTaskCmd(serveCtx, action, taskServe)
	return m, tea.Batch(cmd, m.spinner.Tick)
}

func (m Model) finishTask(msg taskFinishedMsg) (Model, tea.Cmd) {
	label := taskLabel(msg.kind)

	if msg.kind == taskServe {
		m.serveRunning = false
		m.setStepStatus(taskServe, StepPending)
		m.setStepLast(taskServe, time.Since(m.stepStart(taskServe)))
		if msg.err != nil {
			m.appendMessage("ERROR", fmt.Sprintf("Serve stopped: %v", msg.err))
			m.appendHistory("ERROR", fmt.Sprintf("Serve stopped: %v", msg.err))
			m.lastAction = "Serve error"
		} else {
			m.appendMessage("INFO", "Serve stopped")
			m.appendHistory("ACTION", "Serve stopped")
			m.lastAction = "Serve stopped"
		}
		return m, nil
	}

	if msg.err != nil {
		m.setStepStatus(msg.kind, StepPending)
		m.setStepLast(msg.kind, time.Since(m.stepStart(msg.kind)))
		m.appendMessage("ERROR", fmt.Sprintf("%s failed: %v", label, msg.err))
		m.appendHistory("ERROR", fmt.Sprintf("%s failed: %v", label, msg.err))
		m.lastAction = label + " failed"
		return m, nil
	}

	m.setStepStatus(msg.kind, StepDone)
	m.setStepLast(msg.kind, time.Since(m.stepStart(msg.kind)))
	m.appendMessage("INFO", fmt.Sprintf("%s completed", label))
	m.appendHistory("ACTION", fmt.Sprintf("%s completed", label))
	m.lastAction = label + " completed"

	if msg.kind == taskInit {
		m.hasInit = true
	} else if msg.kind == taskUpdate {
		if pathExists(m.options.ContentDir) {
			m.hasInit = true
		}
	}
	return m, nil
}

func (m Model) finishPluginAction(msg pluginActionFinishedMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		m.appendMessage("ERROR", fmt.Sprintf("Plugin %s %s failed: %v", msg.name, msg.action, msg.err))
		m.lastAction = fmt.Sprintf("Plugin %s error", msg.action)
		return m, nil
	}
	if msg.enabled {
		m.enabledPlugins[msg.name] = true
		m.appendMessage("INFO", fmt.Sprintf("Plugin %s enabled", msg.name))
		m.lastAction = fmt.Sprintf("Plugin %s enabled", msg.name)
	} else {
		delete(m.enabledPlugins, msg.name)
		m.appendMessage("INFO", fmt.Sprintf("Plugin %s disabled", msg.name))
		m.lastAction = fmt.Sprintf("Plugin %s disabled", msg.name)
	}
	return m, nil
}

func (m Model) finishSimpleAction(msg simpleActionFinishedMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		m.appendMessage("ERROR", fmt.Sprintf("%s failed: %v", msg.label, msg.err))
		m.lastAction = msg.label + " failed"
		return m, nil
	}
	m.appendMessage("INFO", fmt.Sprintf("%s completed", msg.label))
	m.lastAction = msg.label + " completed"
	return m, nil
}

// ---- Sub-command handlers ----

func (m Model) handleNext() (Model, tea.Cmd) {
	next := m.nextActionKind()
	switch next {
	case taskInit:
		return m.startTask(taskInit)
	case taskUpdate:
		return m.startTask(taskUpdate)
	case taskBuild:
		return m.startTask(taskBuild)
	case taskServe:
		return m.toggleServe()
	default:
		m.appendMessage("INFO", "No next action")
		return m, nil
	}
}

func (m Model) handleVersionCommand() (Model, tea.Cmd) {
	if m.actions.Version == nil {
		m.appendMessage("ERROR", "Version not available")
		return m, nil
	}
	m.appendMessage("INFO", m.actions.Version())
	return m, nil
}

func (m Model) handleNewCommand(args []string) (Model, tea.Cmd) {
	if len(args) < 1 {
		m.appendMessage("ERROR", "Usage: /new <title>")
		return m, nil
	}
	if m.actions.NewPost == nil {
		m.appendMessage("ERROR", "New post not available")
		return m, nil
	}
	// Join all args to support multi-word titles: /new My Great Post
	title := strings.Join(args, " ")
	m.appendMessage("PROGRESS", fmt.Sprintf("Creating post: %s...", title))
	m.lastAction = "New post"
	return m, runNewPostCmd(context.Background(), title, m.actions.NewPost)
}

func (m Model) handleThemeCommand(args []string) (Model, tea.Cmd) {
	if len(args) < 1 || strings.ToLower(args[0]) != "init" {
		m.appendMessage("ERROR", "Usage: /theme init <name>")
		return m, nil
	}
	if len(args) < 2 {
		m.appendMessage("ERROR", "Usage: /theme init <name>")
		return m, nil
	}
	if m.actions.ThemeInit == nil {
		m.appendMessage("ERROR", "Theme init not available")
		return m, nil
	}
	name := args[1]
	m.appendMessage("PROGRESS", fmt.Sprintf("Scaffolding theme %s...", name))
	m.lastAction = "Theme init"
	return m, runThemeInitCmd(context.Background(), name, m.actions.ThemeInit)
}

func (m Model) handlePluginCommand(args []string) (Model, tea.Cmd) {
	if len(args) < 1 {
		m.appendMessage("ERROR", "Usage: /plugin <enable|disable|toggle|list|install|init> [args]")
		return m, nil
	}
	sub := strings.ToLower(args[0])
	switch sub {
	case "list":
		if m.actions.PluginList != nil {
			return m.runSimpleAction("Plugin list", m.actions.PluginList)
		}
		return m.renderPluginList()
	case "enable", "disable", "toggle":
		if len(args) < 2 {
			m.appendMessage("ERROR", fmt.Sprintf("Usage: /plugin %s <name>", sub))
			return m, nil
		}
		name := normalizePluginName(args[1])
		if name == "" {
			m.appendMessage("ERROR", "Plugin name is empty")
			return m, nil
		}
		return m.runPluginAction(sub, name)
	case "install":
		if len(args) < 2 {
			m.appendMessage("ERROR", "Usage: /plugin install <path> [name]")
			return m, nil
		}
		if m.actions.PluginInstall == nil {
			m.appendMessage("ERROR", "Plugin install not available")
			return m, nil
		}
		path := args[1]
		name := ""
		if len(args) >= 3 {
			name = args[2]
		}
		m.appendMessage("PROGRESS", fmt.Sprintf("Installing plugin from %s...", path))
		m.lastAction = "Plugin install"
		return m, runPluginInstallCmd(context.Background(), path, name, m.actions.PluginInstall)
	case "init":
		if len(args) < 2 {
			m.appendMessage("ERROR", "Usage: /plugin init <name> [dir] [lang]")
			return m, nil
		}
		if m.actions.PluginInit == nil {
			m.appendMessage("ERROR", "Plugin init not available")
			return m, nil
		}
		name := args[1]
		dir := ""
		if len(args) >= 3 {
			dir = args[2]
		}
		lang := ""
		if len(args) >= 4 {
			lang = args[3]
		}
		m.appendMessage("PROGRESS", fmt.Sprintf("Scaffolding plugin %s...", name))
		m.lastAction = "Plugin init"
		return m, runPluginInitCmd(context.Background(), name, dir, lang, m.actions.PluginInit)
	default:
		m.appendMessage("ERROR", "Usage: /plugin <enable|disable|toggle|list|install|init> [args]")
		return m, nil
	}
}

func (m Model) runPluginAction(action string, name string) (Model, tea.Cmd) {
	switch action {
	case "enable":
		if m.actions.PluginEnable == nil {
			m.appendMessage("ERROR", "Plugin enable not available")
			return m, nil
		}
		m.appendMessage("PROGRESS", fmt.Sprintf("Enabling plugin %s...", name))
		m.lastAction = fmt.Sprintf("Plugin enable %s", name)
		return m, runPluginActionCmd(context.Background(), action, name, true, m.actions.PluginEnable)
	case "disable":
		if m.actions.PluginDisable == nil {
			m.appendMessage("ERROR", "Plugin disable not available")
			return m, nil
		}
		m.appendMessage("PROGRESS", fmt.Sprintf("Disabling plugin %s...", name))
		m.lastAction = fmt.Sprintf("Plugin disable %s", name)
		return m, runPluginActionCmd(context.Background(), action, name, false, m.actions.PluginDisable)
	case "toggle":
		if m.actions.PluginToggle == nil {
			m.appendMessage("ERROR", "Plugin toggle not available")
			return m, nil
		}
		enabled := m.enabledPlugins[name]
		next := !enabled
		verb := "Enabling"
		if enabled {
			verb = "Disabling"
		}
		m.appendMessage("PROGRESS", fmt.Sprintf("%s plugin %s...", verb, name))
		m.lastAction = fmt.Sprintf("Plugin toggle %s", name)
		return m, runPluginActionCmd(context.Background(), action, name, next, m.actions.PluginToggle)
	default:
		m.appendMessage("ERROR", "Unknown plugin action")
		return m, nil
	}
}

func (m Model) renderPluginList() (Model, tea.Cmd) {
	if len(m.options.Plugins) == 0 && len(m.enabledPlugins) == 0 {
		m.appendMessage("INFO", "No plugins installed")
		return m, nil
	}
	installed := map[string]bool{}
	for _, plugin := range m.options.Plugins {
		installed[plugin] = true
		state := "off"
		if m.enabledPlugins[plugin] {
			state = "on"
		}
		m.appendMessage("INFO", fmt.Sprintf("Plugin %s: %s", plugin, state))
	}
	for name := range m.enabledPlugins {
		if installed[name] {
			continue
		}
		m.appendMessage("WARN", fmt.Sprintf("Plugin %s: missing (enabled but not installed)", name))
	}
	return m, nil
}

func (m Model) runSimpleAction(label string, action func(context.Context) error) (Model, tea.Cmd) {
	if action == nil {
		m.appendMessage("ERROR", fmt.Sprintf("%s not available", label))
		return m, nil
	}
	m.appendMessage("PROGRESS", fmt.Sprintf("%s started", label))
	m.lastAction = label + " started"
	return m, runSimpleActionCmd(context.Background(), label, action)
}
