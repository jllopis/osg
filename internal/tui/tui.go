package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const version = "v0.1.0"

const maxMessages = 200
const defaultPrefixTimeout = 600 * time.Millisecond

var (
	colorPrimary = lipgloss.Color("#8ECDF2")
	colorAccent  = lipgloss.Color("#F6A000")
	colorMuted   = lipgloss.Color("#7A7A7A")
	colorSuccess = lipgloss.Color("#7CFC90")
	colorDanger  = lipgloss.Color("#FF6B6B")
)

type Actions struct {
	Init          func(context.Context) error
	Update        func(context.Context) error
	Build         func(context.Context) error
	Serve         func(context.Context) error
	Doctor        func(context.Context) error
	ThemeInit     func(context.Context, string) error
	PluginEnable  func(context.Context, string) error
	PluginDisable func(context.Context, string) error
	PluginToggle  func(context.Context, string) error
	PluginInstall func(context.Context, string, string) error
	PluginList    func(context.Context) error
	PluginInit    func(context.Context, string, string) error
	Version       func() string
}

type Options struct {
	ConfigPath     string
	VaultPath      string
	ContentDir     string
	PublicDir      string
	ServeAddr      string
	LogPath        string
	PrefixKey      string
	PrefixMs       int
	Plugins        []string
	EnabledPlugins []string
	HasContent     bool
}

type LogSink struct {
	mu     sync.Mutex
	buf    bytes.Buffer
	ch     chan string
	writer Writer
}

type Writer interface {
	Write([]byte) (int, error)
}

func NewLogSink(writer Writer) *LogSink {
	return &LogSink{
		ch:     make(chan string, 200),
		writer: writer,
	}
}

func (s *LogSink) Channel() <-chan string {
	return s.ch
}

func (s *LogSink) Write(p []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	n, _ := s.buf.Write(p)
	for {
		data := s.buf.Bytes()
		idx := bytes.IndexByte(data, '\n')
		if idx == -1 {
			break
		}
		line := strings.TrimRight(string(data[:idx]), "\r")
		_ = s.buf.Next(idx + 1)
		if strings.TrimSpace(line) == "" {
			continue
		}
		if s.writer != nil {
			_, _ = s.writer.Write([]byte(line + "\n"))
		}
		select {
		case s.ch <- line:
		default:
		}
	}

	return n, nil
}

type StepStatus int

const (
	StepDone StepStatus = iota
	StepRunning
	StepPending
	StepDisabled
)

type Step struct {
	Name   string
	Key    string
	Status StepStatus
	Start  time.Time
	Last   time.Duration
}

type Message struct {
	Label  string
	Text   string
	Time   time.Time
	Fields map[string]any
}

type BuildSummary struct {
	Total    int
	Rendered int
	Skipped  int
	Cached   int
	Errors   int
}

type DoctorSummary struct {
	Warnings int
	Errors   int
}

type taskKind int

const (
	taskInit taskKind = iota
	taskUpdate
	taskBuild
	taskServe
)

type taskFinishedMsg struct {
	kind taskKind
	err  error
}

type pluginActionFinishedMsg struct {
	action  string
	name    string
	enabled bool
	err     error
}

type simpleActionFinishedMsg struct {
	label string
	err   error
}

type logLineMsg struct {
	line string
}

type progressTickMsg time.Time

type prefixTimeoutMsg struct {
	token int
}

type Model struct {
	width          int
	height         int
	showHeader     bool
	showRight      bool
	serveRunning   bool
	serveCancel    context.CancelFunc
	lastAction     string
	steps          []Step
	messages       []Message
	input          textinput.Model
	spinner        spinner.Model
	statusSpin     spinner.Model
	progress       progress.Model
	actions        Actions
	options        Options
	logCh          <-chan string
	history        *History
	prefixKey      string
	prefixArmed    bool
	prefixToken    int
	prefixDelay    time.Duration
	enabledPlugins map[string]bool
	guideEnabled   bool
	wizardEnabled  bool
	hasInit        bool
	lastBuild      *BuildSummary
	lastDoctor     *DoctorSummary
}

func New(actions Actions, options Options, sink *LogSink, history *History) Model {
	now := time.Now()
	var logCh <-chan string
	if sink != nil {
		logCh = sink.Channel()
	}

	input := textinput.New()
	input.Prompt = "> "
	input.Placeholder = "init | update | build | serve | stop | doctor | theme ... | plugin ... | help"
	input.Focus()

	spin := spinner.New()
	spin.Spinner = spinner.Line
	spin.Style = lipgloss.NewStyle().Foreground(colorAccent)

	statusSpin := spinner.New()
	statusSpin.Spinner = spinner.MiniDot
	statusSpin.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	bar := progress.New(progress.WithDefaultGradient())
	prefixKey := normalizePrefix(options.PrefixKey)
	prefixText := prefixLabel(prefixKey)
	prefixDelay := normalizePrefixDelay(options.PrefixMs)

	return Model{
		showHeader: true,
		showRight:  true,
		steps: []Step{
			{Name: "Init", Key: "I", Status: StepPending},
			{Name: "Update content", Key: "A", Status: StepPending},
			{Name: "Build", Key: "B", Status: StepPending},
			{Name: "Serve preview", Key: "S", Status: StepPending},
		},
		messages: []Message{
			{Label: "SYS", Text: "OSG Builder ready", Time: now},
			{Label: "INFO", Text: fmt.Sprintf("Use %s + I/A/B/S/D/L/V. Prompt for theme/plugin commands.", prefixText), Time: now},
		},
		input:          input,
		spinner:        spin,
		statusSpin:     statusSpin,
		progress:       bar,
		actions:        actions,
		options:        options,
		logCh:          logCh,
		history:        history,
		prefixKey:      prefixKey,
		prefixDelay:    prefixDelay,
		enabledPlugins: normalizePluginSet(options.EnabledPlugins),
		guideEnabled:   true,
		wizardEnabled:  true,
		hasInit:        options.HasContent,
	}
}

func Run(ctx context.Context, actions Actions, options Options, sink *LogSink, history *History) error {
	model := New(actions, options, sink, history)
	if ctx == nil {
		ctx = context.Background()
	}
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := program.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	if m.logCh != nil {
		return tea.Batch(listenLogCmd(m.logCh), spinnerTickCmd(), progressTickCmd())
	}
	return tea.Batch(spinnerTickCmd(), progressTickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" {
			m.appendHistory("EXIT", "User quit")
			return m, tea.Quit
		}

		if m.prefixArmed {
			m.prefixArmed = false
			if key == "esc" {
				return m, nil
			}
			if model, cmd, ok := m.handlePrefixKey(key); ok {
				return model, cmd
			}
			if key == "enter" {
				return m.handleCommand()
			}
			m.insertPrefixKey()
			// Let this key fall through to input handling.
		}

		if m.isPrefixKey(key) {
			m.prefixArmed = true
			m.prefixToken++
			cmds = append(cmds, prefixTimeoutCmd(m.prefixToken, m.prefixDelay))
			return m, tea.Batch(cmds...)
		}

		if key == "enter" {
			return m.handleCommand()
		}
	case logLineMsg:
		m.appendParsedLog(msg.line)
		if m.logCh != nil {
			cmds = append(cmds, listenLogCmd(m.logCh))
		}
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
		var statusCmd tea.Cmd
		m.statusSpin, statusCmd = m.statusSpin.Update(msg)
		cmds = append(cmds, statusCmd)
	case progressTickMsg:
		return m.updateProgress(msg)
	case prefixTimeoutMsg:
		if msg.token == m.prefixToken && m.prefixArmed {
			m.prefixArmed = false
			m.insertPrefixKey()
		}
	case progress.FrameMsg:
		var cmd tea.Cmd
		var model tea.Model
		model, cmd = m.progress.Update(msg)
		if updated, ok := model.(progress.Model); ok {
			m.progress = updated
		}
		cmds = append(cmds, cmd)
	}

	var inputCmd tea.Cmd
	m.input, inputCmd = m.input.Update(msg)
	cmds = append(cmds, inputCmd)
	m.adjustInputWidth()

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if m.width == 0 {
		return "OSG Builder loading..."
	}

	header := ""
	if m.showHeader {
		header = renderHeader(m.width)
	}

	progressView := ""
	if m.hasRunningNonServe() {
		m.progress.Width = 18
		progressView = m.progress.View()
	}
	activity := ""
	if m.hasRunning() {
		activity = m.statusSpin.View()
	}
	status := renderStatusBar(m.width, m.lastAction, m.serveRunning, progressView, activity)
	bodyHeight := m.height - lipgloss.Height(header) - lipgloss.Height(status)
	if bodyHeight < 0 {
		bodyHeight = 0
	}
	panelHeight := bodyHeight - 3
	if panelHeight < 0 {
		panelHeight = 0
	}

	leftWidth, centerWidth, rightWidth := m.layoutWidths()

	spin := ""
	if m.hasRunning() {
		spin = m.spinner.View()
	}

	left := renderLeftPanel(m.steps, leftWidth, panelHeight, spin, time.Now(), prefixLabel(m.prefixKey), m.serveRunning, m.guideEnabled)
	center := renderCenterPanel(m, centerWidth, panelHeight, m.input.View())
	var body string
	if m.showRight {
		right := renderRightPanel(rightWidth, panelHeight, m.options, m.serveRunning, m.enabledPlugins)
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
		if slack := m.width - lipgloss.Width(body); slack > 0 {
			right = renderRightPanel(rightWidth+slack, panelHeight, m.options, m.serveRunning, m.enabledPlugins)
			body = lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
		}
	} else {
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, center)
	}

	return lipgloss.JoinVertical(lipgloss.Top, header, body, status)
}

func (m Model) startTask(kind taskKind) (tea.Model, tea.Cmd) {
	if kind == taskServe {
		return m, nil
	}

	stepIndex := stepIndexForTask(kind)
	if stepIndex >= 0 && stepIndex < len(m.steps) {
		if m.steps[stepIndex].Status == StepRunning {
			m.appendMessage("INFO", fmt.Sprintf("%s already running", m.steps[stepIndex].Name))
			return m, nil
		}
		m.steps[stepIndex].Status = StepRunning
		m.steps[stepIndex].Start = time.Now()
		m.steps[stepIndex].Last = 0
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
	return m, cmd
}

func (m Model) toggleServe() (tea.Model, tea.Cmd) {
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
	return m, cmd
}

func (m Model) finishTask(msg taskFinishedMsg) (tea.Model, tea.Cmd) {
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

func (m Model) finishPluginAction(msg pluginActionFinishedMsg) (tea.Model, tea.Cmd) {
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

func (m Model) finishSimpleAction(msg simpleActionFinishedMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.appendMessage("ERROR", fmt.Sprintf("%s failed: %v", msg.label, msg.err))
		m.lastAction = msg.label + " failed"
		return m, nil
	}
	m.appendMessage("INFO", fmt.Sprintf("%s completed", msg.label))
	m.lastAction = msg.label + " completed"
	return m, nil
}

func (m Model) handlePrefixKey(key string) (tea.Model, tea.Cmd, bool) {
	switch strings.ToLower(key) {
	case "i":
		model, cmd := m.startTask(taskInit)
		return model, cmd, true
	case "a":
		model, cmd := m.startTask(taskUpdate)
		return model, cmd, true
	case "b":
		model, cmd := m.startTask(taskBuild)
		return model, cmd, true
	case "s":
		model, cmd := m.toggleServe()
		return model, cmd, true
	case "d":
		model, cmd := m.runSimpleAction("Doctor", m.actions.Doctor)
		return model, cmd, true
	case "l":
		if m.actions.PluginList != nil {
			model, cmd := m.runSimpleAction("Plugin list", m.actions.PluginList)
			return model, cmd, true
		}
		model, cmd := m.renderPluginList()
		return model, cmd, true
	case "v":
		model, cmd := m.handleVersionCommand()
		return model, cmd, true
	case "n":
		model, cmd := m.handleNextCommand()
		return model, cmd, true
	case "w":
		m.wizardEnabled = !m.wizardEnabled
		m.appendMessage("INFO", fmt.Sprintf("Wizard %s", onOff(m.wizardEnabled)))
		return m, nil, true
	case "g":
		m.guideEnabled = !m.guideEnabled
		state := "enabled"
		if !m.guideEnabled {
			state = "disabled"
		}
		m.appendMessage("INFO", fmt.Sprintf("Guide %s", state))
		return m, nil, true
	case "h":
		m.showHeader = !m.showHeader
		return m, nil, true
	case "p":
		m.showRight = !m.showRight
		return m, nil, true
	case "q":
		m.appendHistory("EXIT", "User quit")
		return m, tea.Quit, true
	default:
		return m, nil, false
	}
}

func (m Model) handleCommand() (tea.Model, tea.Cmd) {
	raw := strings.TrimSpace(m.input.Value())
	m.input.SetValue("")
	m.input.Focus()
	if raw == "" {
		return m, nil
	}

	normalized := strings.TrimPrefix(raw, "/")
	fields := strings.Fields(normalized)
	if len(fields) == 0 {
		return m, nil
	}
	cmd := strings.ToLower(fields[0])
	m.appendHistory("CMD", strings.ToLower(normalized))

	switch cmd {
	case "init", "i":
		return m.startTask(taskInit)
	case "update", "update-content", "a":
		return m.startTask(taskUpdate)
	case "build", "b":
		return m.startTask(taskBuild)
	case "serve", "s":
		return m.toggleServe()
	case "stop":
		if !m.serveRunning {
			m.appendMessage("INFO", "Serve is not running")
			return m, nil
		}
		return m.toggleServe()
	case "help", "h":
		m.appendMessage("INFO", "Commands: init, update, build, serve, stop, next, wizard [on|off|toggle], doctor, guide [on|off|toggle], theme init <name>, plugin enable/disable/toggle/list/install/init, version, help, quit")
		return m, nil
	case "doctor":
		return m.runSimpleAction("Doctor", m.actions.Doctor)
	case "guide":
		return m.handleGuideCommand(fields)
	case "wizard":
		return m.handleWizardCommand(fields)
	case "next":
		return m.handleNextCommand()
	case "theme":
		return m.handleThemeCommand(fields)
	case "plugin":
		return m.handlePluginCommand(fields)
	case "version":
		return m.handleVersionCommand()
	case "quit", "exit":
		m.appendHistory("EXIT", "User quit via command")
		return m, tea.Quit
	default:
		m.appendMessage("ERROR", fmt.Sprintf("Unknown command: %s", raw))
		return m, nil
	}
}

func (m Model) handlePluginCommand(fields []string) (tea.Model, tea.Cmd) {
	if len(fields) < 2 {
		m.appendMessage("ERROR", "Usage: plugin <enable|disable|toggle|list|install|init> [args]")
		return m, nil
	}

	sub := strings.ToLower(fields[1])
	switch sub {
	case "list":
		if m.actions.PluginList != nil {
			return m.runSimpleAction("Plugin list", m.actions.PluginList)
		}
		return m.renderPluginList()
	case "enable", "disable", "toggle":
		if len(fields) < 3 {
			m.appendMessage("ERROR", fmt.Sprintf("Usage: plugin %s <name>", sub))
			return m, nil
		}
		name := normalizePluginName(fields[2])
		if name == "" {
			m.appendMessage("ERROR", "Plugin name is empty")
			return m, nil
		}
		return m.runPluginAction(sub, name)
	case "install":
		if len(fields) < 3 {
			m.appendMessage("ERROR", "Usage: plugin install <path> [name]")
			return m, nil
		}
		if m.actions.PluginInstall == nil {
			m.appendMessage("ERROR", "Plugin install not available")
			return m, nil
		}
		path := fields[2]
		name := ""
		if len(fields) >= 4 {
			name = fields[3]
		}
		return m.runPluginInstall(path, name)
	case "init":
		if len(fields) < 3 {
			m.appendMessage("ERROR", "Usage: plugin init <name> [dir]")
			return m, nil
		}
		if m.actions.PluginInit == nil {
			m.appendMessage("ERROR", "Plugin init not available")
			return m, nil
		}
		name := fields[2]
		dir := ""
		if len(fields) >= 4 {
			dir = fields[3]
		}
		return m.runPluginInit(name, dir)
	default:
		m.appendMessage("ERROR", "Usage: plugin <enable|disable|toggle|list|install|init> [args]")
		return m, nil
	}
}

func (m Model) runPluginAction(action string, name string) (tea.Model, tea.Cmd) {
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

func (m Model) runPluginInstall(path string, name string) (tea.Model, tea.Cmd) {
	m.appendMessage("PROGRESS", fmt.Sprintf("Installing plugin from %s...", path))
	m.lastAction = "Plugin install"
	return m, runPluginInstallCmd(context.Background(), path, name, m.actions.PluginInstall)
}

func (m Model) runPluginInit(name string, dir string) (tea.Model, tea.Cmd) {
	m.appendMessage("PROGRESS", fmt.Sprintf("Scaffolding plugin %s...", name))
	m.lastAction = "Plugin init"
	return m, runPluginInitCmd(context.Background(), name, dir, m.actions.PluginInit)
}

func (m Model) renderPluginList() (tea.Model, tea.Cmd) {
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

func (m Model) runSimpleAction(label string, action func(context.Context) error) (tea.Model, tea.Cmd) {
	if action == nil {
		m.appendMessage("ERROR", fmt.Sprintf("%s not available", label))
		return m, nil
	}
	m.appendMessage("PROGRESS", fmt.Sprintf("%s started", label))
	m.lastAction = label + " started"
	return m, runSimpleActionCmd(context.Background(), label, action)
}

func (m Model) handleThemeCommand(fields []string) (tea.Model, tea.Cmd) {
	if len(fields) < 2 || strings.ToLower(fields[1]) != "init" {
		m.appendMessage("ERROR", "Usage: theme init <name>")
		return m, nil
	}
	if len(fields) < 3 {
		m.appendMessage("ERROR", "Usage: theme init <name>")
		return m, nil
	}
	if m.actions.ThemeInit == nil {
		m.appendMessage("ERROR", "Theme init not available")
		return m, nil
	}
	name := fields[2]
	m.appendMessage("PROGRESS", fmt.Sprintf("Scaffolding theme %s...", name))
	m.lastAction = "Theme init"
	return m, runThemeInitCmd(context.Background(), name, m.actions.ThemeInit)
}

func (m Model) handleVersionCommand() (tea.Model, tea.Cmd) {
	if m.actions.Version == nil {
		m.appendMessage("ERROR", "Version not available")
		return m, nil
	}
	m.appendMessage("INFO", m.actions.Version())
	return m, nil
}

func (m Model) handleGuideCommand(fields []string) (tea.Model, tea.Cmd) {
	if len(fields) == 1 {
		m.guideEnabled = !m.guideEnabled
	} else {
		switch strings.ToLower(fields[1]) {
		case "on", "enable", "enabled":
			m.guideEnabled = true
		case "off", "disable", "disabled":
			m.guideEnabled = false
		case "toggle":
			m.guideEnabled = !m.guideEnabled
		default:
			m.appendMessage("ERROR", "Usage: guide [on|off|toggle]")
			return m, nil
		}
	}
	state := "enabled"
	if !m.guideEnabled {
		state = "disabled"
	}
	m.appendMessage("INFO", fmt.Sprintf("Guide %s", state))
	return m, nil
}

func (m Model) handleWizardCommand(fields []string) (tea.Model, tea.Cmd) {
	if len(fields) == 1 {
		m.wizardEnabled = !m.wizardEnabled
	} else {
		switch strings.ToLower(fields[1]) {
		case "on", "enable", "enabled":
			m.wizardEnabled = true
		case "off", "disable", "disabled":
			m.wizardEnabled = false
		case "toggle":
			m.wizardEnabled = !m.wizardEnabled
		default:
			m.appendMessage("ERROR", "Usage: wizard [on|off|toggle]")
			return m, nil
		}
	}
	m.appendMessage("INFO", fmt.Sprintf("Wizard %s", onOff(m.wizardEnabled)))
	return m, nil
}

func (m Model) handleNextCommand() (tea.Model, tea.Cmd) {
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

func (m Model) updateProgress(_ progressTickMsg) (tea.Model, tea.Cmd) {
	if !m.hasRunningNonServe() {
		return m, progressTickCmd()
	}

	elapsed := m.runningElapsedNonServe()
	percent := (elapsed.Seconds())
	percent = percent - float64(int(percent/10))*10
	percent = percent / 10

	cmd := m.progress.SetPercent(percent)
	return m, tea.Batch(cmd, progressTickCmd())
}

func (m *Model) setStepStatus(kind taskKind, status StepStatus) {
	index := stepIndexForTask(kind)
	if index >= 0 && index < len(m.steps) {
		m.steps[index].Status = status
	}
}

func (m *Model) setStepStart(kind taskKind, start time.Time) {
	index := stepIndexForTask(kind)
	if index >= 0 && index < len(m.steps) {
		m.steps[index].Start = start
	}
}

func (m *Model) setStepLast(kind taskKind, duration time.Duration) {
	index := stepIndexForTask(kind)
	if index >= 0 && index < len(m.steps) {
		m.steps[index].Last = duration
		m.steps[index].Start = time.Time{}
	}
}

func (m *Model) stepStart(kind taskKind) time.Time {
	index := stepIndexForTask(kind)
	if index >= 0 && index < len(m.steps) {
		return m.steps[index].Start
	}
	return time.Time{}
}

func (m *Model) appendMessage(label string, text string) {
	m.messages = append(m.messages, Message{
		Label: label,
		Text:  text,
		Time:  time.Now(),
	})
	m.trimMessages()
}

func (m *Model) appendHistory(label string, text string) {
	if m.history == nil {
		return
	}
	m.history.Append(label, text)
}

func (m *Model) appendParsedLog(line string) {
	m.captureSummaries(line)
	msg := parseLogLine(line)
	m.messages = append(m.messages, msg)
	m.trimMessages()
}

func (m *Model) trimMessages() {
	if len(m.messages) <= maxMessages {
		return
	}
	m.messages = m.messages[len(m.messages)-maxMessages:]
}

func listenLogCmd(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		line, ok := <-ch
		if !ok {
			return nil
		}
		return logLineMsg{line: line}
	}
}

func parseLogLine(line string) Message {
	now := time.Now()
	entry := map[string]any{}
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		return Message{Label: "LOG", Text: line, Time: now}
	}

	label := strings.ToUpper(fmt.Sprint(entry["level"]))
	if label == "" {
		label = "LOG"
	}

	text := fmt.Sprint(entry["msg"])
	if text == "<nil>" {
		text = line
	}

	timestamp := now
	if rawTime, ok := entry["time"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339Nano, rawTime); err == nil {
			timestamp = parsed
		}
	}

	fields := map[string]any{}
	for key, value := range entry {
		if key == "time" || key == "level" || key == "msg" {
			continue
		}
		fields[key] = value
	}

	return Message{Label: label, Text: text, Time: timestamp, Fields: fields}
}

func (m *Model) captureSummaries(line string) {
	entry := map[string]any{}
	if err := json.Unmarshal([]byte(line), &entry); err != nil {
		return
	}
	msg, _ := entry["msg"].(string)
	switch msg {
	case "build summary":
		m.lastBuild = &BuildSummary{
			Total:    asInt(entry["total"]),
			Rendered: asInt(entry["rendered"]),
			Skipped:  asInt(entry["skipped"]),
			Cached:   asInt(entry["cached"]),
			Errors:   asInt(entry["errors"]),
		}
	case "doctor summary":
		m.lastDoctor = &DoctorSummary{
			Warnings: asInt(entry["warnings"]),
			Errors:   asInt(entry["errors"]),
		}
	}
}

func spinnerTickCmd() tea.Cmd {
	return spinner.Tick
}

func progressTickCmd() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg {
		return progressTickMsg(t)
	})
}

func prefixTimeoutCmd(token int, delay time.Duration) tea.Cmd {
	return tea.Tick(delay, func(time.Time) tea.Msg {
		return prefixTimeoutMsg{token: token}
	})
}

func runTaskCmd(ctx context.Context, action func(context.Context) error, kind taskKind) tea.Cmd {
	return func() tea.Msg {
		if action == nil {
			return taskFinishedMsg{kind: kind, err: fmt.Errorf("action not available")}
		}
		err := action(ctx)
		return taskFinishedMsg{kind: kind, err: err}
	}
}

func runPluginActionCmd(ctx context.Context, action string, name string, enabled bool, handler func(context.Context, string) error) tea.Cmd {
	return func() tea.Msg {
		if handler == nil {
			return pluginActionFinishedMsg{action: action, name: name, enabled: enabled, err: fmt.Errorf("action not available")}
		}
		err := handler(ctx, name)
		return pluginActionFinishedMsg{action: action, name: name, enabled: enabled, err: err}
	}
}

func runSimpleActionCmd(ctx context.Context, label string, action func(context.Context) error) tea.Cmd {
	return func() tea.Msg {
		if action == nil {
			return simpleActionFinishedMsg{label: label, err: fmt.Errorf("action not available")}
		}
		err := action(ctx)
		return simpleActionFinishedMsg{label: label, err: err}
	}
}

func runThemeInitCmd(ctx context.Context, name string, action func(context.Context, string) error) tea.Cmd {
	return func() tea.Msg {
		if action == nil {
			return simpleActionFinishedMsg{label: "Theme init", err: fmt.Errorf("action not available")}
		}
		err := action(ctx, name)
		return simpleActionFinishedMsg{label: "Theme init", err: err}
	}
}

func runPluginInstallCmd(ctx context.Context, path string, name string, action func(context.Context, string, string) error) tea.Cmd {
	return func() tea.Msg {
		if action == nil {
			return simpleActionFinishedMsg{label: "Plugin install", err: fmt.Errorf("action not available")}
		}
		err := action(ctx, path, name)
		return simpleActionFinishedMsg{label: "Plugin install", err: err}
	}
}

func runPluginInitCmd(ctx context.Context, name string, dir string, action func(context.Context, string, string) error) tea.Cmd {
	return func() tea.Msg {
		if action == nil {
			return simpleActionFinishedMsg{label: "Plugin init", err: fmt.Errorf("action not available")}
		}
		err := action(ctx, name, dir)
		return simpleActionFinishedMsg{label: "Plugin init", err: err}
	}
}

func actionForTask(actions Actions, kind taskKind) func(context.Context) error {
	switch kind {
	case taskInit:
		return actions.Init
	case taskUpdate:
		return actions.Update
	case taskBuild:
		return actions.Build
	default:
		return nil
	}
}

func stepIndexForTask(kind taskKind) int {
	switch kind {
	case taskInit:
		return 0
	case taskUpdate:
		return 1
	case taskBuild:
		return 2
	case taskServe:
		return 3
	default:
		return -1
	}
}

func taskLabel(kind taskKind) string {
	switch kind {
	case taskInit:
		return "Init"
	case taskUpdate:
		return "Update content"
	case taskBuild:
		return "Build"
	case taskServe:
		return "Serve"
	default:
		return "Task"
	}
}

func renderHeader(width int) string {
	banner := []string{
		fmt.Sprintf("  ___   ___   ____  %s", version),
		" / _ \\ / _ \\ / ___|",
		"| | | | | | | |  _ ",
		"| |_| | |_| | |_| |",
		" \\___/ \\___/ \\____|",
	}

	bannerStyle := lipgloss.NewStyle().
		Foreground(colorPrimary).
		Padding(0, 1)

	lines := []string{}
	for _, line := range banner {
		lines = append(lines, bannerStyle.Width(width).Render(trimLine(line, width)))
	}

	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func renderLeftPanel(steps []Step, width int, height int, spin string, now time.Time, prefixText string, serveRunning bool, guideEnabled bool) string {
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		Padding(1, 1).
		Width(panelContentWidth(width)).
		Height(height)

	guideLabel := "Guide: on"
	if !guideEnabled {
		guideLabel = "Guide: off"
	}
	hint := lipgloss.NewStyle().Foreground(colorMuted).Render("Use prefix for quick actions. " + guideLabel)
	lines := []string{panelTitle("Workflow"), hint}
	for _, step := range steps {
		lines = append(lines, formatStep(step, spin, now))
	}
	lines = append(lines, "")
	lines = append(lines, panelTitle("Actions"))
	lines = append(lines, fmt.Sprintf("Prefix: %s", prefixText))
	lines = append(lines, fmt.Sprintf("%s + I Init", prefixText))
	lines = append(lines, fmt.Sprintf("%s + A Update", prefixText))
	lines = append(lines, fmt.Sprintf("%s + B Build", prefixText))
	serveLabel := "Serve"
	if serveRunning {
		serveLabel = "Stop serve"
	}
	lines = append(lines, fmt.Sprintf("%s + S %s", prefixText, serveLabel))
	lines = append(lines, fmt.Sprintf("%s + D Doctor", prefixText))
	lines = append(lines, fmt.Sprintf("%s + L Plugin list", prefixText))
	lines = append(lines, fmt.Sprintf("%s + V Version", prefixText))
	lines = append(lines, fmt.Sprintf("%s + G Toggle guide", prefixText))
	lines = append(lines, fmt.Sprintf("%s + W Toggle wizard", prefixText))
	lines = append(lines, fmt.Sprintf("%s + N Next step", prefixText))
	lines = append(lines, "")
	lines = append(lines, panelTitle("UI"))
	lines = append(lines, fmt.Sprintf("%s + H Toggle header", prefixText))
	lines = append(lines, fmt.Sprintf("%s + P Toggle panel", prefixText))
	lines = append(lines, fmt.Sprintf("%s + Q Quit", prefixText))

	return panelStyle.Render(strings.Join(lines, "\n"))
}

func renderCenterPanel(m Model, width int, height int, input string) string {
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		Padding(1, 1).
		Width(panelContentWidth(width)).
		Height(height)

	lines := []string{panelTitle("Status")}
	lines = append(lines, m.statusLines()...)
	lines = append(lines, "")
	lines = append(lines, panelTitle("Flow"))
	lines = append(lines, m.flowLines()...)
	lines = append(lines, "")
	lines = append(lines, panelTitle("Alerts"))
	lines = append(lines, m.alertLines(3)...)
	lines = append(lines, "")
	lines = append(lines, panelTitle("Recent output"))
	for _, msg := range m.recentMessages(8) {
		lines = append(lines, formatEvent(msg)...)
	}
	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Foreground(colorMuted).Render("Prompt (type help):"))
	lines = append(lines, input)

	return panelStyle.Render(strings.Join(lines, "\n"))
}

func renderRightPanel(width int, height int, options Options, serveRunning bool, enabledPlugins map[string]bool) string {
	panelStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		Padding(1, 1).
		Width(panelContentWidth(width)).
		Height(height)

	serverBadge := badge("STOPPED", colorMuted)
	if serveRunning {
		serverBadge = badge("RUNNING", colorSuccess)
	}

	lines := []string{
		panelTitle("Project"),
		fmt.Sprintf("Config: %s", defaultConfig(options.ConfigPath)),
		fmt.Sprintf("Vault: %s", defaultValue(options.VaultPath)),
		fmt.Sprintf("Content: %s", defaultValue(options.ContentDir)),
		fmt.Sprintf("Public: %s", defaultValue(options.PublicDir)),
		"",
		panelTitle("Preview Server"),
		fmt.Sprintf("Addr: %s", defaultAddr(options.ServeAddr)),
		fmt.Sprintf("Status: %s", serverBadge),
	}

	lines = append(lines, "")
	lines = append(lines, panelTitle("Plugins"))
	if len(options.Plugins) == 0 && len(enabledPlugins) == 0 {
		lines = append(lines, "none")
	} else {
		installed := map[string]bool{}
		for _, plugin := range options.Plugins {
			installed[plugin] = true
			state := "[off]"
			if enabledPlugins[plugin] {
				state = "[on]"
			}
			lines = append(lines, fmt.Sprintf("%s %s", state, plugin))
		}
		for plugin := range enabledPlugins {
			if installed[plugin] {
				continue
			}
			lines = append(lines, fmt.Sprintf("[missing] %s", plugin))
		}
	}

	if strings.TrimSpace(options.LogPath) != "" {
		lines = append(lines, "")
		lines = append(lines, panelTitle("Logs"))
		lines = append(lines, options.LogPath)
	}

	return panelStyle.Render(strings.Join(lines, "\n"))
}

func renderStatusBar(width int, lastAction string, serveRunning bool, progressView string, activity string) string {
	style := lipgloss.NewStyle().
		Background(lipgloss.Color("#333333")).
		Foreground(lipgloss.Color("#FAFAFA")).
		Padding(0, 1)

	status := "idle"
	if lastAction != "" {
		status = lastAction
	}
	serverBadge := badge("STOPPED", colorMuted)
	if serveRunning {
		serverBadge = badge("RUNNING", colorSuccess)
	}
	text := fmt.Sprintf("status: %s | server: %s", status, serverBadge)
	if progressView != "" {
		text = fmt.Sprintf("status: %s | %s | server: %s", status, progressView, serverBadge)
	}
	if activity != "" {
		text = fmt.Sprintf("%s | %s", text, activity)
	}
	return style.Width(width).Render(text)
}

func panelTitle(title string) string {
	return lipgloss.NewStyle().
		Foreground(colorPrimary).
		Bold(true).
		Render(title)
}

func badge(text string, color lipgloss.Color) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#111111")).
		Background(color).
		Padding(0, 1).
		Bold(true).
		Render(text)
}

func panelContentWidth(total int) int {
	width := total - 4 // 2 borders + 2 padding (1 each side)
	if width < 1 {
		return 1
	}
	return width
}

func formatStep(step Step, spin string, now time.Time) string {
	icon, style := stepIcon(step.Status, spin)
	key := ""
	if step.Key != "" {
		key = "(" + step.Key + ") "
	}

	suffix := ""
	if step.Status == StepRunning && !step.Start.IsZero() {
		suffix = " • " + formatDuration(now.Sub(step.Start))
	} else if step.Last > 0 {
		suffix = " • " + formatDuration(step.Last)
	}

	return fmt.Sprintf("%s %s%s%s", style.Render(icon), key, step.Name, suffix)
}

func stepIcon(status StepStatus, spin string) (string, lipgloss.Style) {
	switch status {
	case StepDone:
		return "[x]", lipgloss.NewStyle().Foreground(lipgloss.Color("#7CFC90"))
	case StepRunning:
		icon := "[>]"
		if spin != "" {
			icon = spin
		}
		return icon, lipgloss.NewStyle().Foreground(colorAccent)
	case StepDisabled:
		return "[-]", lipgloss.NewStyle().Foreground(colorMuted)
	default:
		return "[ ]", lipgloss.NewStyle().Foreground(colorMuted)
	}
}

func formatDuration(d time.Duration) string {
	seconds := int(d.Seconds())
	minutes := seconds / 60
	seconds = seconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func (m Model) statusLines() []string {
	lines := []string{}
	if m.guideEnabled {
		lines = append(lines, fmt.Sprintf("Next: %s", m.nextActionLabel()))
	}
	lines = append(lines, fmt.Sprintf("Wizard: %s", onOff(m.wizardEnabled)))

	lines = append(lines, fmt.Sprintf("Last: %s", fallback(m.lastAction, "idle")))

	serverBadge := badge("STOPPED", colorMuted)
	if m.serveRunning {
		serverBadge = badge("RUNNING", colorSuccess)
	}
	lines = append(lines, fmt.Sprintf("Serve: %s", serverBadge))

	if running := m.runningStepLabel(); running != "" {
		lines = append(lines, fmt.Sprintf("Running: %s", running))
	}

	if m.lastBuild != nil {
		lines = append(lines, fmt.Sprintf("Build: total %d • rendered %d • cached %d • errors %d", m.lastBuild.Total, m.lastBuild.Rendered, m.lastBuild.Cached, m.lastBuild.Errors))
	}
	if m.lastDoctor != nil {
		lines = append(lines, fmt.Sprintf("Doctor: warnings %d • errors %d", m.lastDoctor.Warnings, m.lastDoctor.Errors))
	}

	return lines
}

func (m Model) flowLines() []string {
	lines := []string{}
	now := time.Now()
	for _, step := range m.steps {
		lines = append(lines, formatStep(step, "", now))
	}
	return lines
}

func (m Model) alertLines(limit int) []string {
	lines := []string{}
	count := 0
	for i := len(m.messages) - 1; i >= 0 && count < limit; i-- {
		msg := m.messages[i]
		if msg.Label == "ERROR" || msg.Label == "WARN" || msg.Label == "WARNING" {
			lines = append(lines, formatEvent(msg)...)
			count++
		}
	}
	if count == 0 {
		return []string{lipgloss.NewStyle().Foreground(colorMuted).Render("No alerts")}
	}
	return lines
}

func (m Model) runningStepLabel() string {
	for _, step := range m.steps {
		if step.Status == StepRunning {
			return step.Name
		}
	}
	return ""
}

func (m Model) nextActionLabel() string {
	if !m.hasInit {
		return "Init"
	}
	for i, step := range m.steps {
		if step.Status == StepPending {
			if i == stepIndexForTask(taskServe) {
				if m.serveRunning {
					return "Stop serve"
				}
				return "Serve preview"
			}
			return step.Name
		}
	}
	if m.serveRunning {
		return "Stop serve"
	}
	return "Serve preview"
}

func (m Model) nextActionKind() taskKind {
	if !m.hasInit {
		return taskInit
	}
	for i, step := range m.steps {
		if step.Status == StepPending {
			if i == stepIndexForTask(taskServe) {
				return taskServe
			}
			return taskKind(i)
		}
	}
	return taskServe
}

func onOff(value bool) string {
	if value {
		return "on"
	}
	return "off"
}

func (m Model) recentMessages(limit int) []Message {
	if limit <= 0 || len(m.messages) == 0 {
		return nil
	}
	if len(m.messages) <= limit {
		return m.messages
	}
	return m.messages[len(m.messages)-limit:]
}

func fallback(value string, alt string) string {
	if strings.TrimSpace(value) == "" {
		return alt
	}
	return value
}

func formatMessage(msg Message) string {
	stamp := msg.Time.Format("15:04:05")
	label := fmt.Sprintf("[%s %s]", msg.Label, stamp)
	labelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#F5D76E"))
	switch msg.Label {
	case "SYS":
		labelStyle = lipgloss.NewStyle().Foreground(colorPrimary)
	case "PROGRESS":
		labelStyle = lipgloss.NewStyle().Foreground(colorAccent)
	case "ERROR":
		labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
	case "WARN", "WARNING":
		labelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFD166"))
	case "INFO":
		labelStyle = lipgloss.NewStyle().Foreground(colorPrimary)
	case "DEBUG":
		labelStyle = lipgloss.NewStyle().Foreground(colorMuted)
	}
	return fmt.Sprintf("%s %s", labelStyle.Render(label), msg.Text)
}

func formatEvent(msg Message) []string {
	label := msg.Label
	if label == "" {
		label = "LOG"
	}

	title := msg.Text
	detail := ""

	switch msg.Text {
	case "exported":
		dest := asString(msg.Fields["dest"])
		source := pathBase(asString(msg.Fields["source"]))
		if dest != "" {
			title = "Exported → " + dest
		}
		if source != "" {
			detail = "from " + source
		}
	case "update-content summary":
		title = fmt.Sprintf("Update content: exported %d, skipped %d, drafts %d, errors %d",
			asInt(msg.Fields["exported"]),
			asInt(msg.Fields["skipped"]),
			asInt(msg.Fields["drafts"]),
			asInt(msg.Fields["errors"]),
		)
	case "build incremental":
		mode := asString(msg.Fields["mode"])
		changed := asInt(msg.Fields["changed"])
		removed := asInt(msg.Fields["removed"])
		title = fmt.Sprintf("Build incremental: %s (changed %d, removed %d)", fallback(mode, "partial"), changed, removed)
	case "build summary":
		title = fmt.Sprintf("Build: rendered %d, cached %d, errors %d",
			asInt(msg.Fields["rendered"]),
			asInt(msg.Fields["cached"]),
			asInt(msg.Fields["errors"]),
		)
	case "initial build complete":
		title = "Initial build complete"
	case "watch enabled":
		title = fmt.Sprintf("Watch enabled (debounce %d ms, live reload %s)",
			asInt(msg.Fields["debounce_ms"]),
			asString(msg.Fields["live_reload"]),
		)
	}

	line := fmt.Sprintf("%s %s", badge(label, colorPrimary), title)
	if detail == "" {
		return []string{line}
	}
	return []string{line, lipgloss.NewStyle().Foreground(colorMuted).Render(detail)}
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case fmt.Stringer:
		return strings.TrimSpace(v.String())
	case bool:
		if v {
			return "true"
		}
		return "false"
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
	return ""
}

func pathBase(value string) string {
	if value == "" {
		return ""
	}
	return filepath.Base(value)
}

func pathExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func defaultValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func defaultAddr(value string) string {
	if strings.TrimSpace(value) == "" {
		return ":1313"
	}
	return value
}

func defaultConfig(value string) string {
	if strings.TrimSpace(value) == "" {
		return "config.yaml"
	}
	return value
}

func serveStatus(running bool) string {
	if running {
		return "running"
	}
	return "stopped"
}

func trimLine(line string, width int) string {
	if width <= 0 {
		return line
	}
	if len(line) <= width {
		return line
	}
	return line[:width]
}

func normalizePluginSet(input []string) map[string]bool {
	set := map[string]bool{}
	for _, name := range input {
		name = normalizePluginName(name)
		if name == "" {
			continue
		}
		set[name] = true
	}
	return set
}

func normalizePluginName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	if strings.HasSuffix(strings.ToLower(name), ".wasm") {
		name = strings.TrimSuffix(name, filepath.Ext(name))
	}
	return strings.TrimSpace(name)
}

func asInt(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case string:
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return 0
}

func normalizePrefix(prefix string) string {
	prefix = strings.TrimSpace(strings.ToLower(prefix))
	switch prefix {
	case "", "space", "spc":
		return " "
	default:
		return prefix
	}
}

func normalizePrefixDelay(ms int) time.Duration {
	if ms <= 0 {
		return defaultPrefixTimeout
	}
	return time.Duration(ms) * time.Millisecond
}

func prefixLabel(prefix string) string {
	if prefix == " " {
		return "SPACE"
	}
	return strings.ToUpper(prefix)
}

func (m Model) isPrefixKey(key string) bool {
	if m.prefixKey == " " {
		return key == " " || key == "space"
	}
	return key == m.prefixKey
}

func (m *Model) insertPrefixKey() {
	if m.prefixKey == " " {
		m.input.SetValue(m.input.Value() + " ")
		m.input.CursorEnd()
	}
}

func (m Model) hasRunning() bool {
	for _, step := range m.steps {
		if step.Status == StepRunning {
			return true
		}
	}
	return false
}

func (m Model) hasRunningNonServe() bool {
	for i, step := range m.steps {
		if step.Status == StepRunning && i != stepIndexForTask(taskServe) {
			return true
		}
	}
	return false
}

func (m Model) runningElapsedNonServe() time.Duration {
	for i, step := range m.steps {
		if step.Status == StepRunning && i != stepIndexForTask(taskServe) && !step.Start.IsZero() {
			return time.Since(step.Start)
		}
	}
	return 0
}

func (m *Model) adjustInputWidth() {
	_, centerWidth, _ := m.layoutWidths()
	width := centerWidth - 6
	if width < 10 {
		width = 10
	}
	m.input.Width = width
}

func (m Model) layoutWidths() (int, int, int) {
	leftWidth := 32
	rightWidth := 34
	minCenter := 20
	minSide := 20

	if !m.showRight {
		rightWidth = 0
	}

	available := m.width
	if available <= 0 {
		return leftWidth, minCenter, rightWidth
	}

	required := leftWidth + rightWidth + minCenter
	if available < required {
		deficit := required - available
		reduceRight := minInt(deficit, maxInt(0, rightWidth-minSide))
		rightWidth -= reduceRight
		deficit -= reduceRight

		reduceLeft := minInt(deficit, maxInt(0, leftWidth-minSide))
		leftWidth -= reduceLeft
		deficit -= reduceLeft

		if deficit > 0 {
			minCenter = maxInt(10, minCenter-deficit)
		}
	}

	centerWidth := available - leftWidth - rightWidth
	if centerWidth < minCenter {
		centerWidth = minCenter
	}
	if m.showRight {
		extra := available - (leftWidth + centerWidth + rightWidth)
		if extra > 0 {
			rightWidth += extra
		}
	}
	return leftWidth, centerWidth, rightWidth
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
