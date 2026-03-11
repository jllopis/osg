package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"osg/internal/config"

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
		if m.configActive && m.configScreen != nil {
			m.configScreen.Resize(msg.Width, msg.Height)
		}
		return m, nil

	case tea.KeyMsg:
		// If config editor is active, delegate all keys there.
		if m.configActive && m.configScreen != nil {
			return m.handleConfigKey(msg)
		}

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

		case key.Matches(msg, keys.ToggleServe):
			return m.toggleServe("static")

		case key.Matches(msg, keys.ToggleAPI):
			return m.toggleAPI()

		case key.Matches(msg, keys.ToggleLogs):
			m.logPanel.Toggle()
			if m.logPanel.Visible() {
				m.logFocus = true
				m.syncLogPanel()
			} else {
				m.logFocus = false
			}
			m.recalcLayout()
			return m, nil

		case key.Matches(msg, keys.ToggleConfig):
			return m.openConfigEditor()
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

		// Log panel navigation (when visible).
		if m.logPanel.Visible() && !m.acVisible {
			switch msg.String() {
			case logModKey(m, "up"):
				m.logPanel.ScrollUp(1)
				return m, nil
			case logModKey(m, "down"):
				m.logPanel.ScrollDown(1)
				return m, nil
			case logModKey(m, "left"):
				m.logPanel.PrevTab()
				m.syncLogPanel()
				return m, nil
			case logModKey(m, "right"):
				m.logPanel.NextTab()
				m.syncLogPanel()
				return m, nil
			}
		}

		// Scrolling viewport (when not in autocomplete).
		if !m.acVisible {
			switch {
			case key.Matches(msg, keys.ScrollUp):
				m.viewport.ScrollUp(1)
				return m, nil
			case key.Matches(msg, keys.ScrollDown):
				m.viewport.ScrollDown(1)
				return m, nil
			case key.Matches(msg, keys.PageUp):
				m.viewport.HalfPageUp()
				return m, nil
			case key.Matches(msg, keys.PageDown):
				m.viewport.HalfPageDown()
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
		m.appendParsedLog(msg.source, msg.line)
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
	// Viewport height = total - 3 - logPanelHeight (if visible)
	logH := 0
	if m.logPanel.Visible() {
		logH = PanelHeight(m.height)
	}
	vpHeight := m.height - 3 - logH
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

	// Resize log panel to full width.
	if m.logPanel.Visible() {
		m.logPanel.Resize(m.width, logH)
		m.syncLogPanel()
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
		// Check for --api flag.
		serveMode := "static"
		for _, arg := range args {
			if arg == "--api" {
				serveMode = "api"
			}
		}
		return m.toggleServe(serveMode)
	case "api":
		return m.toggleAPI()
	case "stop":
		return m.handleStop(args)
	case "check":
		return m.runSimpleAction("Check", m.actions.Check)
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
	case "logs":
		m.logPanel.Toggle()
		if m.logPanel.Visible() {
			m.logFocus = true
			m.syncLogPanel()
			mod := logModLabel(m)
			m.appendMessage("INFO", fmt.Sprintf("Log panel opened (%s+↑↓ scroll, %s+←→ tabs)", mod, mod))
		} else {
			m.logFocus = false
			m.appendMessage("INFO", "Log panel closed")
		}
		m.recalcLayout()
		return m, nil
	case "config":
		return m.openConfigEditor()
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
		return m.toggleServe("static")
	}
	if kind == taskAPI {
		return m.toggleAPI()
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

func (m Model) toggleServe(mode string) (Model, tea.Cmd) {
	if m.serveRunning {
		if m.serveCancel != nil {
			m.serveCancel()
		}
		m.appendMessage("PROGRESS", "Stopping preview server...")
		m.appendHistory("ACTION", "Serve stop requested")
		m.lastAction = "Serve stop requested"
		return m, nil
	}

	var action func(context.Context) error
	switch mode {
	case "api":
		action = m.actions.ServeWithAPI
		if action == nil {
			m.appendMessage("ERROR", "Serve with API action not available")
			return m, nil
		}
	default: // "static"
		action = m.actions.Serve
		if action == nil {
			m.appendMessage("ERROR", "Serve action not available")
			return m, nil
		}
	}

	serveCtx, cancel := context.WithCancel(context.Background())
	m.serveCancel = cancel
	m.serveRunning = true
	m.serveMode = mode
	m.setStepStatus(taskServe, StepRunning)
	m.setStepStart(taskServe, time.Now())

	modeLabel := "static"
	if mode == "api" {
		modeLabel = "static+api"
	}
	m.appendMessage("PROGRESS", fmt.Sprintf("Starting preview server on %s (mode: %s)...", defaultAddr(m.options.ServeAddr), modeLabel))
	m.appendHistory("ACTION", fmt.Sprintf("Serve started on %s (mode: %s)", defaultAddr(m.options.ServeAddr), modeLabel))
	m.lastAction = "Serve started"

	cmd := runTaskCmd(serveCtx, action, taskServe)
	return m, tea.Batch(cmd, m.spinner.Tick)
}

func (m Model) toggleAPI() (Model, tea.Cmd) {
	if m.apiRunning {
		if m.apiCancel != nil {
			m.apiCancel()
		}
		m.appendMessage("PROGRESS", "Stopping standalone API...")
		m.appendHistory("ACTION", "API stop requested")
		m.lastAction = "API stop requested"
		return m, nil
	}

	action := m.actions.RunAPI
	if action == nil {
		m.appendMessage("ERROR", "API action not available")
		return m, nil
	}

	apiCtx, cancel := context.WithCancel(context.Background())
	m.apiCancel = cancel
	m.apiRunning = true
	m.setStepStatus(taskAPI, StepRunning)
	m.setStepStart(taskAPI, time.Now())
	m.appendMessage("PROGRESS", fmt.Sprintf("Starting standalone API on %s...", defaultAPIAddr(m.options.APIAddr)))
	m.appendHistory("ACTION", fmt.Sprintf("API started on %s", defaultAPIAddr(m.options.APIAddr)))
	m.lastAction = "API started"

	cmd := runTaskCmd(apiCtx, action, taskAPI)
	return m, tea.Batch(cmd, m.spinner.Tick)
}

func (m Model) finishTask(msg taskFinishedMsg) (Model, tea.Cmd) {
	label := taskLabel(msg.kind)

	if msg.kind == taskServe {
		m.serveRunning = false
		m.serveMode = ""
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

	if msg.kind == taskAPI {
		m.apiRunning = false
		m.setStepStatus(taskAPI, StepPending)
		m.setStepLast(taskAPI, time.Since(m.stepStart(taskAPI)))
		if msg.err != nil {
			m.appendMessage("ERROR", fmt.Sprintf("API stopped: %v", msg.err))
			m.appendHistory("ERROR", fmt.Sprintf("API stopped: %v", msg.err))
			m.lastAction = "API error"
		} else {
			m.appendMessage("INFO", "API stopped")
			m.appendHistory("ACTION", "API stopped")
			m.lastAction = "API stopped"
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

	switch msg.kind {
	case taskInit:
		m.hasInit = true
	case taskUpdate:
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

func (m Model) handleStop(args []string) (Model, tea.Cmd) {
	target := "serve" // default if no argument
	if len(args) > 0 {
		target = strings.ToLower(args[0])
	}
	switch target {
	case "serve":
		if !m.serveRunning {
			m.appendMessage("INFO", "Server is not running")
			return m, nil
		}
		return m.toggleServe("static") // mode is ignored when stopping
	case "api":
		if !m.apiRunning {
			m.appendMessage("INFO", "API is not running")
			return m, nil
		}
		return m.toggleAPI()
	default:
		m.appendMessage("ERROR", "Usage: /stop serve|api")
		return m, nil
	}
}

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
		return m.toggleServe("static")
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
	if len(args) < 1 {
		m.appendMessage("ERROR", "Usage: /theme init <name> [--parent <parent>] | /theme list")
		return m, nil
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "list":
		if m.actions.ThemeList == nil {
			m.appendMessage("ERROR", "Theme list not available")
			return m, nil
		}
		m.appendMessage("PROGRESS", "Listing themes...")
		m.lastAction = "Theme list"
		return m.runSimpleAction("Theme list", m.actions.ThemeList)
	case "init":
		if len(args) < 2 {
			m.appendMessage("ERROR", "Usage: /theme init <name> [--parent <parent>]")
			return m, nil
		}
		if m.actions.ThemeInit == nil {
			m.appendMessage("ERROR", "Theme init not available")
			return m, nil
		}
		name := args[1]
		parent := ""
		// Parse optional --parent flag.
		for i := 2; i < len(args)-1; i++ {
			if args[i] == "--parent" {
				parent = args[i+1]
			}
		}
		if parent != "" {
			m.appendMessage("PROGRESS", fmt.Sprintf("Scaffolding child theme %s (parent: %s)...", name, parent))
		} else {
			m.appendMessage("PROGRESS", fmt.Sprintf("Scaffolding theme %s...", name))
		}
		m.lastAction = "Theme init"
		return m, runThemeInitCmd(context.Background(), name, parent, m.actions.ThemeInit)
	default:
		m.appendMessage("ERROR", "Usage: /theme init <name> [--parent <parent>] | /theme list")
		return m, nil
	}
}

func (m Model) handlePluginCommand(args []string) (Model, tea.Cmd) {
	if len(args) < 1 {
		m.appendMessage("ERROR", "Usage: /plugin <enable|disable|toggle|list|install|init|search|update> [args]")
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
	case "search":
		if m.actions.PluginSearch == nil {
			m.appendMessage("ERROR", "Plugin search not available")
			return m, nil
		}
		query := ""
		if len(args) >= 2 {
			query = strings.Join(args[1:], " ")
		}
		m.appendMessage("PROGRESS", "Searching plugin index...")
		m.lastAction = "Plugin search"
		return m, runPluginSearchCmd(context.Background(), query, m.actions.PluginSearch)
	case "update":
		if m.actions.PluginUpdate == nil {
			m.appendMessage("ERROR", "Plugin update not available")
			return m, nil
		}
		name := ""
		if len(args) >= 2 {
			name = args[1]
		}
		m.appendMessage("PROGRESS", "Checking for plugin updates...")
		m.lastAction = "Plugin update"
		return m, runPluginUpdateCmd(context.Background(), name, m.actions.PluginUpdate)
	default:
		m.appendMessage("ERROR", "Usage: /plugin <enable|disable|toggle|list|install|init|search|update> [args]")
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

// ---- Config editor ----

// openConfigEditor opens the config editor modal.
func (m Model) openConfigEditor() (Model, tea.Cmd) {
	path := m.options.ConfigPath
	if strings.TrimSpace(path) == "" {
		path = "config.yaml"
	}
	cs, err := NewConfigScreen(path)
	if err != nil {
		m.appendMessage("ERROR", fmt.Sprintf("Cannot open config: %v", err))
		return m, nil
	}
	cs.Resize(m.width, m.height)
	m.configScreen = cs
	m.configActive = true
	m.appendHistory("ACTION", "Config editor opened")
	return m, nil
}

// closeConfigEditor closes the config editor modal.
func (m *Model) closeConfigEditor() {
	m.configActive = false
	m.configScreen = nil
}

// handleConfigKey processes key events when the config editor is active.
func (m Model) handleConfigKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	cs := m.configScreen
	keyStr := msg.String()

	// Ctrl+C always quits.
	if key.Matches(msg, keys.Quit) {
		m.appendHistory("EXIT", "User quit")
		return m, tea.Quit
	}

	// Confirmation dialog takes priority.
	if cs.ConfirmQuitVisible() {
		return m.handleConfigConfirmKey(keyStr)
	}

	// Editing mode: delegate to field editor.
	if cs.Editing() {
		return m.handleConfigEditKey(msg)
	}

	// Normal navigation.
	switch keyStr {
	case "ctrl+s":
		if err := cs.Save(); err != nil {
			m.appendMessage("ERROR", fmt.Sprintf("Config save failed: %v", err))
		} else {
			m.appendMessage("INFO", "Config saved")
			m.appendHistory("ACTION", "Config saved")
			m.reloadOptionsFromConfig(cs)
		}
		return m, nil

	case "esc":
		if cs.IsDirty() {
			cs.ShowConfirmQuit()
		} else {
			m.closeConfigEditor()
			m.appendMessage("INFO", "Config editor closed")
		}
		return m, nil

	case "tab":
		cs.SwitchPanel()
		return m, nil

	case "up":
		if cs.FocusPanel() == "sections" {
			cs.MoveSection(-1)
		} else {
			cs.MoveField(-1)
		}
		return m, nil

	case "down":
		if cs.FocusPanel() == "sections" {
			cs.MoveSection(1)
		} else {
			cs.MoveField(1)
		}
		return m, nil

	case "enter":
		if cs.FocusPanel() == "fields" {
			field, ok := cs.currentField()
			if ok && field.Type == config.FieldBool {
				cs.ToggleBool()
			} else if ok {
				cs.StartEdit()
			}
		} else {
			// In section panel, Enter switches to fields.
			cs.SwitchPanel()
		}
		return m, nil

	case " ":
		if cs.FocusPanel() == "fields" {
			field, ok := cs.currentField()
			if ok && field.Type == config.FieldBool {
				cs.ToggleBool()
			}
		}
		return m, nil

	case "a":
		if cs.FocusPanel() == "fields" {
			cs.AddListItem()
		}
		return m, nil

	case "d":
		if cs.FocusPanel() == "fields" {
			cs.DeleteListItem()
		}
		return m, nil
	}

	return m, nil
}

// handleConfigEditKey handles key events while a field is being edited.
func (m Model) handleConfigEditKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	cs := m.configScreen
	keyStr := msg.String()

	switch keyStr {
	case "enter":
		if err := cs.fieldEditor.Validate(); err != nil {
			// Stay in edit mode — error shown in view.
			return m, nil
		}
		cs.ConfirmEdit()
		return m, nil
	case "esc":
		cs.CancelEdit()
		return m, nil
	default:
		// Forward the actual tea.Msg to the text input for proper handling.
		cs.fieldEditor.textInput, _ = cs.fieldEditor.textInput.Update(msg)
		return m, nil
	}
}

// handleConfigConfirmKey handles key events in the unsaved-changes dialog.
func (m Model) handleConfigConfirmKey(keyStr string) (Model, tea.Cmd) {
	cs := m.configScreen
	switch keyStr {
	case "y":
		if err := cs.Save(); err != nil {
			m.appendMessage("ERROR", fmt.Sprintf("Config save failed: %v", err))
			cs.HideConfirmQuit()
		} else {
			m.appendMessage("INFO", "Config saved and editor closed")
			m.appendHistory("ACTION", "Config saved and editor closed")
			m.closeConfigEditor()
		}
		return m, nil
	case "n":
		m.closeConfigEditor()
		m.appendMessage("INFO", "Config editor closed (changes discarded)")
		return m, nil
	case "esc":
		cs.HideConfirmQuit()
		return m, nil
	}
	return m, nil
}

// reloadOptionsFromConfig updates m.options fields from the in-memory
// config node after a successful save, so the sidebar reflects changes
// immediately.
func (m *Model) reloadOptionsFromConfig(cs *ConfigScreen) {
	if v := cs.GetValue("site_title"); v != "" {
		m.options.SiteTitle = v
	}
	if v := cs.GetValue("vault_path"); v != "" {
		m.options.VaultPath = v
	}
	if v := cs.GetValue("public_dir"); v != "" {
		m.options.PublicDir = v
	}
	if v := cs.GetValue("content_dir"); v != "" {
		m.options.ContentDir = v
	}
	if v := cs.GetValue("serve_addr"); v != "" {
		m.options.ServeAddr = v
	}
	if v := cs.GetValue("interactions.listen"); v != "" {
		m.options.APIAddr = v
	}

	// Reload enabled plugins list.
	if plugins, ok := cs.GetSequence("plugins_enabled"); ok {
		m.enabledPlugins = normalizePluginSet(plugins)
	}

	// Reload log modifier.
	if v := cs.GetValue("tui_log_modifier"); v != "" {
		m.options.LogModifier = v
	}
}

// logModKey returns the Bubble Tea key string for the configured log modifier
// combined with a direction (e.g. "alt+up", "shift+down").
func logModKey(m Model, dir string) string {
	mod := m.options.LogModifier
	if mod == "" {
		mod = "shift"
	}
	return mod + "+" + dir
}

// logModLabel returns a human-readable short label for the configured
// log modifier ("Alt" or "Shift").
func logModLabel(m Model) string {
	switch m.options.LogModifier {
	case "alt":
		return "Alt"
	default:
		return "Shift"
	}
}
