package tui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const maxMessages = 500

// ---- Public types preserved from old tui.go ----

// Actions holds the closures that execute each OSG operation.
type Actions struct {
	Init          func(context.Context) error
	Update        func(context.Context) error
	Build         func(context.Context) error
	Serve         func(context.Context) error // static-only dev server
	ServeWithAPI  func(context.Context) error // dev server + embedded API
	RunAPI        func(context.Context) error // standalone API server
	Doctor        func(context.Context) error
	ThemeInit     func(context.Context, string, string) error
	ThemeList     func(context.Context) error
	PluginEnable  func(context.Context, string) error
	PluginDisable func(context.Context, string) error
	PluginToggle  func(context.Context, string) error
	PluginInstall func(context.Context, string, string) error
	PluginList    func(context.Context) error
	PluginInit    func(context.Context, string, string, string) error
	PluginSearch  func(context.Context, string) error
	PluginUpdate  func(context.Context, string) error
	NewPost       func(context.Context, string) error
	Version       func() string
}

// Options carries the project configuration into the TUI.
type Options struct {
	ConfigPath     string
	VaultPath      string
	ContentDir     string
	PublicDir      string
	ServeAddr      string
	APIAddr        string // address for standalone API server
	LogPath        string
	SiteTitle      string
	PrefixKey      string // kept for compat, unused in new TUI
	PrefixMs       int    // kept for compat, unused in new TUI
	LogModifier    string // "alt" (default) or "shift" — modifier key for log panel navigation
	Plugins        []string
	EnabledPlugins []string
	HasContent     bool
}

// ---- Domain types ----

// StepStatus represents the state of a workflow step.
type StepStatus int

const (
	StepPending StepStatus = iota
	StepRunning
	StepDone
	StepDisabled
)

// Step represents a single workflow step in the sidebar.
type Step struct {
	Name   string
	Status StepStatus
	Start  time.Time
	Last   time.Duration
}

// Message represents a single log/output line.
type Message struct {
	Label  string
	Text   string
	Time   time.Time
	Fields map[string]any
}

// BuildSummary stores stats from the last build.
type BuildSummary struct {
	Total    int
	Rendered int
	Skipped  int
	Cached   int
	Errors   int
}

// DoctorSummary stores stats from the last doctor run.
type DoctorSummary struct {
	Warnings int
	Errors   int
}

// taskKind identifies which task is running.
type taskKind int

const (
	taskInit taskKind = iota
	taskUpdate
	taskBuild
	taskServe
	taskAPI
)

// ---- Tea messages ----

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
	source string
	line   string
}

// ---- Model ----

// Model is the Bubble Tea model for the OSG TUI.
type Model struct {
	width  int
	height int

	// Panels
	sidebarVisible bool
	viewport       viewport.Model
	viewportReady  bool

	// Serve
	serveRunning bool
	serveCancel  context.CancelFunc
	serveMode    string // "" | "static" | "api" (serve+api embedded)

	// Standalone API
	apiRunning bool
	apiCancel  context.CancelFunc

	// State
	lastAction     string
	steps          []Step
	messages       []Message
	hasInit        bool
	lastBuild      *BuildSummary
	lastDoctor     *DoctorSummary
	enabledPlugins map[string]bool

	// Per-source message buffers for the log panel.
	serveMessages []Message
	apiMessages   []Message

	// Log panel
	logPanel LogPanel
	logFocus bool // true when log panel has keyboard focus

	// Config editor modal
	configScreen *ConfigScreen
	configActive bool // true when config modal is shown

	// Components
	input   textinput.Model
	spinner spinner.Model
	actions Actions
	options Options
	logCh   <-chan TaggedLine
	history *History

	// Autocomplete
	acVisible  bool
	acMatches  []slashCommand
	acSelected int
}

// New creates a new Model, wired to the given actions, options, sinks and history.
// Pass one or more LogSinks; their channels are merged into a single fan-in
// channel. Pass nil for sinks that are not needed.
func New(actions Actions, options Options, history *History, sinks ...*LogSink) Model {
	var logCh <-chan TaggedLine
	// Filter out nil sinks and merge.
	var live []*LogSink
	for _, s := range sinks {
		if s != nil {
			live = append(live, s)
		}
	}
	if len(live) == 1 {
		logCh = live[0].Channel()
	} else if len(live) > 1 {
		logCh = MergeChannels(live...)
	}

	input := textinput.New()
	input.Prompt = promptStyle.Render("> ")
	input.Placeholder = "type a command or / for autocomplete"
	input.Focus()
	input.CharLimit = 256

	spin := spinner.New()
	spin.Spinner = spinner.MiniDot
	spin.Style = lipgloss.NewStyle().Foreground(nordOrange)

	now := time.Now()

	m := Model{
		sidebarVisible: true,
		logPanel:       NewLogPanel(),
		steps: []Step{
			{Name: "Init", Status: StepPending},
			{Name: "Update content", Status: StepPending},
			{Name: "Build", Status: StepPending},
			{Name: "Serve", Status: StepPending},
			{Name: "API", Status: StepPending},
		},
		messages: []Message{
			{Label: "SYS", Text: "OSG ready", Time: now},
			{Label: "INFO", Text: "Type /help for commands, tab to toggle sidebar", Time: now},
		},
		input:          input,
		spinner:        spin,
		actions:        actions,
		options:        options,
		logCh:          logCh,
		history:        history,
		enabledPlugins: normalizePluginSet(options.EnabledPlugins),
		hasInit:        options.HasContent,
	}
	return m
}

// Run starts the Bubble Tea program. This is the entry point called by app/tui.go.
func Run(ctx context.Context, actions Actions, options Options, history *History, sinks ...*LogSink) error {
	model := New(actions, options, history, sinks...)
	if ctx == nil {
		ctx = context.Background()
	}
	program := tea.NewProgram(model, tea.WithAltScreen(), tea.WithContext(ctx))
	_, err := program.Run()
	return err
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick}
	if m.logCh != nil {
		cmds = append(cmds, listenLogCmd(m.logCh))
	}
	return tea.Batch(cmds...)
}

// ---- Tea commands ----

func listenLogCmd(ch <-chan TaggedLine) tea.Cmd {
	return func() tea.Msg {
		tl, ok := <-ch
		if !ok {
			return nil
		}
		return logLineMsg{source: tl.Source, line: tl.Line}
	}
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

func runThemeInitCmd(ctx context.Context, name string, parent string, action func(context.Context, string, string) error) tea.Cmd {
	return func() tea.Msg {
		if action == nil {
			return simpleActionFinishedMsg{label: "Theme init", err: fmt.Errorf("action not available")}
		}
		err := action(ctx, name, parent)
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

func runPluginInitCmd(ctx context.Context, name string, dir string, lang string, action func(context.Context, string, string, string) error) tea.Cmd {
	return func() tea.Msg {
		if action == nil {
			return simpleActionFinishedMsg{label: "Plugin init", err: fmt.Errorf("action not available")}
		}
		err := action(ctx, name, dir, lang)
		return simpleActionFinishedMsg{label: "Plugin init", err: err}
	}
}

func runPluginSearchCmd(ctx context.Context, query string, action func(context.Context, string) error) tea.Cmd {
	return func() tea.Msg {
		if action == nil {
			return simpleActionFinishedMsg{label: "Plugin search", err: fmt.Errorf("action not available")}
		}
		err := action(ctx, query)
		return simpleActionFinishedMsg{label: "Plugin search", err: err}
	}
}

func runPluginUpdateCmd(ctx context.Context, name string, action func(context.Context, string) error) tea.Cmd {
	return func() tea.Msg {
		if action == nil {
			return simpleActionFinishedMsg{label: "Plugin update", err: fmt.Errorf("action not available")}
		}
		err := action(ctx, name)
		return simpleActionFinishedMsg{label: "Plugin update", err: err}
	}
}

func runNewPostCmd(ctx context.Context, title string, action func(context.Context, string) error) tea.Cmd {
	return func() tea.Msg {
		if action == nil {
			return simpleActionFinishedMsg{label: "New post", err: fmt.Errorf("action not available")}
		}
		err := action(ctx, title)
		return simpleActionFinishedMsg{label: "New post", err: err}
	}
}

// ---- Helpers ----

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
	case taskAPI:
		return 4
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
	case taskAPI:
		return "API"
	default:
		return "Task"
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

// ---- Step helpers (pointer receiver for mutation) ----

func (m *Model) setStepStatus(kind taskKind, status StepStatus) {
	idx := stepIndexForTask(kind)
	if idx >= 0 && idx < len(m.steps) {
		m.steps[idx].Status = status
	}
}

func (m *Model) setStepStart(kind taskKind, t time.Time) {
	idx := stepIndexForTask(kind)
	if idx >= 0 && idx < len(m.steps) {
		m.steps[idx].Start = t
	}
}

func (m *Model) setStepLast(kind taskKind, d time.Duration) {
	idx := stepIndexForTask(kind)
	if idx >= 0 && idx < len(m.steps) {
		m.steps[idx].Last = d
		m.steps[idx].Start = time.Time{}
	}
}

func (m *Model) stepStart(kind taskKind) time.Time {
	idx := stepIndexForTask(kind)
	if idx >= 0 && idx < len(m.steps) {
		return m.steps[idx].Start
	}
	return time.Time{}
}

// ---- Message helpers ----

func (m *Model) appendMessage(label string, text string) {
	m.messages = append(m.messages, Message{
		Label: label,
		Text:  text,
		Time:  time.Now(),
	})
	m.trimMessages()
	m.syncViewport()
}

func (m *Model) appendHistory(label string, text string) {
	if m.history == nil {
		return
	}
	m.history.Append(label, text)
}

func (m *Model) appendParsedLog(source, line string) {
	m.captureSummaries(line)
	msg := parseLogLine(line)
	m.messages = append(m.messages, msg)
	// Route to per-source buffer for the log panel.
	switch source {
	case "serve":
		m.serveMessages = append(m.serveMessages, msg)
		if len(m.serveMessages) > maxMessages {
			m.serveMessages = m.serveMessages[len(m.serveMessages)-maxMessages:]
		}
	case "api":
		m.apiMessages = append(m.apiMessages, msg)
		if len(m.apiMessages) > maxMessages {
			m.apiMessages = m.apiMessages[len(m.apiMessages)-maxMessages:]
		}
	}
	m.trimMessages()
	m.syncViewport()
	m.syncLogPanel()
}

func (m *Model) trimMessages() {
	if len(m.messages) <= maxMessages {
		return
	}
	m.messages = m.messages[len(m.messages)-maxMessages:]
}

func (m *Model) syncViewport() {
	if !m.viewportReady {
		return
	}
	m.viewport.SetContent(m.renderOutput())
	m.viewport.GotoBottom()
}

func (m *Model) syncLogPanel() {
	if !m.logPanel.Visible() {
		return
	}
	msgs := MessagesForTab(m.logPanel.Tab(), m.serveMessages, m.apiMessages, m.messages)
	m.logPanel.SetContent(msgs)
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

// ---- Workflow navigation ----

func (m Model) hasRunning() bool {
	for _, step := range m.steps {
		if step.Status == StepRunning {
			return true
		}
	}
	return false
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

// ---- Generic utility helpers ----

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

func defaultAddr(value string) string {
	if strings.TrimSpace(value) == "" {
		return ":1313"
	}
	return value
}

func defaultValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return "-"
	}
	return value
}

func defaultConfig(value string) string {
	if strings.TrimSpace(value) == "" {
		return "config.yaml"
	}
	return value
}

func defaultAPIAddr(value string) string {
	if strings.TrimSpace(value) == "" {
		return ":8090"
	}
	return value
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

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	s := int(d.Seconds())
	if s < 60 {
		return fmt.Sprintf("%ds", s)
	}
	return fmt.Sprintf("%dm%02ds", s/60, s%60)
}
