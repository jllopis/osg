package ui

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"osg/internal/operations"
)

// OperationView is the template-friendly summary of an operations.Snapshot.
// All time/duration formatting is pre-computed so html/template stays
// simple. Built from a runner snapshot in operationsViewFromRunner.
type OperationView struct {
	Name              string
	Kind              string
	Description       string
	Addr              string
	IconName          string
	State             string // raw state for JS / data-state pill matching
	StateLabel        string // human label ("running", "idle", ...)
	PillClass         string // status-pill modifier (.ok / .err / ...)
	Running           bool
	UptimeLabel       string
	HasLastRun        bool
	LastStatusLabel   string
	LastRanRelative   string
	LastDurationLabel string
	Params            []ParamDef // form schema for parameterised operations
	Confirm           string     // when set, JS prompts before submitting
	ParamsCollapsed   bool       // wrap Params in a closed <details>
}

// HistoryRow is the table-row view of an operations.HistoryRun.
type HistoryRow struct {
	ID            int64
	Name          string
	Kind          string
	IconName      string
	StartedAt     time.Time
	StatusLabel   string
	PillClass     string
	DurationLabel string
	Error         string
}

// HistoryFilter mirrors the GET query string of /history so the form
// can rehydrate its current selections.
type HistoryFilter struct {
	Name   string
	Kind   string
	Status string
}

// DrawerView feeds partials/drawer.html. Carries pre-formatted strings
// (no template-side helpers needed) plus the raw flags the template
// uses to decide which sections to render.
type DrawerView struct {
	Name          string
	Kind          string
	IconName      string
	Subtitle      string
	StatusLabel   string
	PillClass     string
	StartedAt     string
	EndedAt       string
	DurationLabel string
	Error         string
	ParamsJSON    string
	LogTail       []string
	CanStream     bool
	CanRerun      bool
	CanStop       bool
}

// operationsViewFromRunner builds OperationView entries for every
// runner-defined operation, in stable display order.
func operationsViewFromRunner(r *operations.Runner) []OperationView {
	out := []OperationView{}
	if r == nil {
		return out
	}
	now := time.Now()
	for _, snap := range r.Snapshot() {
		out = append(out, operationViewFromSnapshot(snap, now))
	}
	return out
}

func operationViewFromSnapshot(snap operations.Snapshot, now time.Time) OperationView {
	v := OperationView{
		Name:        snap.Definition.Name,
		Kind:        snap.Definition.Kind,
		Description: snap.Definition.Description,
		Addr:        snap.Definition.Addr,
		IconName:    iconForOperation(snap.Definition.Name, snap.Definition.Kind),
		State:       string(snap.State),
		StateLabel:  string(snap.State),
		PillClass:   pillClassForState(string(snap.State)),
	}
	if snap.Active != nil {
		v.Running = true
		v.UptimeLabel = relDuration(now.Sub(snap.Active.StartedAt))
	}
	if snap.LastRun != nil {
		v.HasLastRun = true
		v.LastStatusLabel = snap.LastRun.Status
		v.LastRanRelative = relTime(now.Sub(snap.LastRun.StartedAt))
		if !snap.LastRun.EndedAt.IsZero() {
			v.LastDurationLabel = relDuration(snap.LastRun.EndedAt.Sub(snap.LastRun.StartedAt))
		}
		// Errors persisted on a finished run override pill class so a
		// failed last run appears "error" while idle.
		if !v.Running && snap.LastRun.Error != "" {
			v.PillClass = pillClassForState(string(operations.StateError))
			v.StateLabel = "error"
			v.State = string(operations.StateError)
		}
	}
	v.Params = paramsForOperation(v.Name)
	v.Confirm = confirmTextFor(v.Name)
	v.ParamsCollapsed = hasCollapsibleParams(v.Name)
	return v
}

// findOperationView is a helper used by handlers to extract a single
// view by operation name from a slice (e.g. for the quick-action
// banner). Returns the zero value when not found.
func findOperationView(views []OperationView, name string) OperationView {
	for _, v := range views {
		if v.Name == name {
			return v
		}
	}
	return OperationView{}
}

// historyRowsFromRuns formats a slice of HistoryRun for the /history
// table: pretty status labels, pill classes, durations.
func historyRowsFromRuns(runs []operations.HistoryRun) []HistoryRow {
	out := make([]HistoryRow, 0, len(runs))
	for _, r := range runs {
		row := HistoryRow{
			ID:          r.ID,
			Name:        r.Name,
			Kind:        r.Kind,
			IconName:    iconForOperation(r.Name, r.Kind),
			StartedAt:   r.StartedAt,
			StatusLabel: r.Status,
			PillClass:   pillClassForStatus(r.Status),
			Error:       r.Error,
		}
		if !r.EndedAt.IsZero() {
			row.DurationLabel = relDuration(r.EndedAt.Sub(r.StartedAt))
		}
		out = append(out, row)
	}
	return out
}

// drawerViewForOperation builds a drawer for the named operation.
// Prefers the active run when one is in flight; otherwise renders the
// most recent persisted run.
func drawerViewForOperation(r *operations.Runner, name string) (DrawerView, bool) {
	if r == nil {
		return DrawerView{}, false
	}
	for _, snap := range r.Snapshot() {
		if snap.Definition.Name != name {
			continue
		}
		v := DrawerView{
			Name:        snap.Definition.Name,
			Kind:        snap.Definition.Kind,
			IconName:    iconForOperation(snap.Definition.Name, snap.Definition.Kind),
			Subtitle:    snap.Definition.Description,
			StatusLabel: string(snap.State),
			PillClass:   pillClassForState(string(snap.State)),
		}
		switch {
		case snap.Active != nil:
			v.StartedAt = snap.Active.StartedAt.Local().Format("2006-01-02 15:04:05")
			v.LogTail = snap.LogTail
			v.CanStream = true
			v.CanStop = true
			v.ParamsJSON = encodeParamsJSON(snap.Active.Params)
			v.Error = snap.Active.LastError
		case snap.LastRun != nil:
			v.StartedAt = snap.LastRun.StartedAt.Local().Format("2006-01-02 15:04:05")
			v.EndedAt = snap.LastRun.EndedAt.Local().Format("2006-01-02 15:04:05")
			v.DurationLabel = relDuration(snap.LastRun.EndedAt.Sub(snap.LastRun.StartedAt))
			v.StatusLabel = snap.LastRun.Status
			v.PillClass = pillClassForStatus(snap.LastRun.Status)
			v.Error = snap.LastRun.Error
			v.ParamsJSON = encodeParamsJSON(snap.LastRun.Params)
			v.CanRerun = true
		default:
			v.CanRerun = true
		}
		return v, true
	}
	return DrawerView{}, false
}

// drawerViewForHistory renders a drawer for a specific past run
// looked up by id in the history table. Returns ok=false when the id
// doesn't exist.
func drawerViewForHistory(r *operations.Runner, id int64) (DrawerView, bool) {
	if r == nil {
		return DrawerView{}, false
	}
	rows, err := r.History(operations.Filter{Limit: 1000})
	if err != nil {
		return DrawerView{}, false
	}
	for _, row := range rows {
		if row.ID != id {
			continue
		}
		v := DrawerView{
			Name:        row.Name,
			Kind:        row.Kind,
			IconName:    iconForOperation(row.Name, row.Kind),
			Subtitle:    fmt.Sprintf("run #%d", row.ID),
			StatusLabel: row.Status,
			PillClass:   pillClassForStatus(row.Status),
			StartedAt:   row.StartedAt.Local().Format("2006-01-02 15:04:05"),
			Error:       row.Error,
			ParamsJSON:  encodeParamsJSON(row.Params),
			CanRerun:    row.Status != operations.StatusRunning,
		}
		if !row.EndedAt.IsZero() {
			v.EndedAt = row.EndedAt.Local().Format("2006-01-02 15:04:05")
			v.DurationLabel = relDuration(row.EndedAt.Sub(row.StartedAt))
		}
		return v, true
	}
	return DrawerView{}, false
}

// iconForOperation maps an operation name (or, for unknowns, its kind)
// to the symbol id in /assets/icons.svg. Keeps the template free of
// large if-else chains.
func iconForOperation(name, kind string) string {
	switch name {
	case "build":
		return "build"
	case "deploy":
		return "deploy"
	case "check":
		return "check"
	case "audit":
		return "audit"
	case "new":
		return "new"
	case "theme:init", "theme-init":
		return "theme"
	case "plugin:install", "plugin-install":
		return "plugin"
	case "import:wordpress", "import-wordpress", "import:hugo", "import-hugo":
		return "import"
	case "update-content":
		return "update"
	case "serve":
		return "serve"
	case "api":
		return "api"
	case "watcher":
		return "watcher"
	case "scheduler", "scheduler:trigger":
		return "scheduler"
	}
	if kind == operations.KindService {
		return "serve"
	}
	return "play"
}

// pillClassForState maps a runtime State to the .status-pill modifier.
func pillClassForState(state string) string {
	switch state {
	case string(operations.StateRunning):
		return "is-running"
	case string(operations.StateStarting):
		return "is-running"
	case string(operations.StateStopping):
		return "is-warn"
	case string(operations.StateError):
		return "is-error"
	case string(operations.StateCancelled):
		return "is-warn"
	}
	return "is-idle"
}

// pillClassForStatus maps a persisted Status to the same .status-pill
// modifier so /history rows use a consistent palette.
func pillClassForStatus(status string) string {
	switch status {
	case operations.StatusRunning:
		return "is-running"
	case operations.StatusOK:
		return "is-ok"
	case operations.StatusError:
		return "is-error"
	case operations.StatusCancelled:
		return "is-warn"
	}
	return "is-idle"
}

// encodeParamsJSON returns a stable, indented JSON representation of a
// run's params (or "" when empty).
func encodeParamsJSON(params map[string]any) string {
	if len(params) == 0 {
		return ""
	}
	b, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}

// relTime renders a duration as an English relative string ("2m ago",
// "3h ago", ...). Negative or sub-second values fall back to "just now".
func relTime(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Second:
		return "just now"
	case d < time.Minute:
		return fmt.Sprintf("%ds ago", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours())/24)
	}
}

// relDuration renders a duration as a compact duration label
// ("2.3s", "45ms", "1m 12s") for status lines.
func relDuration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	switch {
	case d < time.Millisecond:
		return "<1ms"
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	case d < time.Hour:
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", m, s)
	default:
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", h, m)
	}
}

// historyFilterFromQuery turns the raw URL query values into a typed
// filter, validating known statuses and kinds.
func historyFilterFromQuery(query map[string][]string) (operations.Filter, HistoryFilter) {
	view := HistoryFilter{
		Name:   strings.TrimSpace(firstValue(query, "name")),
		Kind:   strings.TrimSpace(firstValue(query, "kind")),
		Status: strings.TrimSpace(firstValue(query, "status")),
	}
	f := operations.Filter{Limit: 100}
	if view.Name != "" {
		f.Name = view.Name
	}
	switch view.Kind {
	case operations.KindService, operations.KindTask:
		f.Kind = view.Kind
	}
	switch view.Status {
	case operations.StatusOK, operations.StatusError,
		operations.StatusCancelled, operations.StatusRunning:
		f.Status = view.Status
	}
	return f, view
}

func firstValue(m map[string][]string, key string) string {
	if vs := m[key]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}
