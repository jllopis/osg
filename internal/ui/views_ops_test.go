package ui

import (
	"strings"
	"testing"
	"time"

	"osg/internal/operations"
)

func TestIconForOperation(t *testing.T) {
	tests := []struct {
		name string
		kind string
		want string
	}{
		{"build", operations.KindTask, "build"},
		{"deploy", operations.KindTask, "deploy"},
		{"check", operations.KindTask, "check"},
		{"audit", operations.KindTask, "audit"},
		{"new", operations.KindTask, "new"},
		{"theme:init", operations.KindTask, "theme"},
		{"theme-init", operations.KindTask, "theme"},
		{"plugin:install", operations.KindTask, "plugin"},
		{"plugin-install", operations.KindTask, "plugin"},
		{"import:wordpress", operations.KindTask, "import"},
		{"import-wordpress", operations.KindTask, "import"},
		{"import:hugo", operations.KindTask, "import"},
		{"import-hugo", operations.KindTask, "import"},
		{"update-content", operations.KindTask, "update"},
		{"serve", operations.KindService, "serve"},
		{"api", operations.KindService, "api"},
		{"watcher", operations.KindService, "watcher"},
		{"scheduler", operations.KindService, "scheduler"},
		{"scheduler:trigger", operations.KindTask, "scheduler"},
		// Unknown name, service kind falls back to "serve".
		{"mystery-service", operations.KindService, "serve"},
		// Unknown name, task kind falls back to "play".
		{"mystery-task", operations.KindTask, "play"},
		// Unknown name, empty kind falls back to "play".
		{"mystery", "", "play"},
	}
	for _, tt := range tests {
		t.Run(tt.name+"/"+tt.kind, func(t *testing.T) {
			if got := iconForOperation(tt.name, tt.kind); got != tt.want {
				t.Errorf("iconForOperation(%q, %q) = %q, want %q", tt.name, tt.kind, got, tt.want)
			}
		})
	}
}

func TestPillClassForState(t *testing.T) {
	tests := []struct {
		state string
		want  string
	}{
		{string(operations.StateRunning), "is-running"},
		{string(operations.StateStarting), "is-running"},
		{string(operations.StateStopping), "is-warn"},
		{string(operations.StateError), "is-error"},
		{string(operations.StateCancelled), "is-warn"},
		{string(operations.StateIdle), "is-idle"},
		{"something-else", "is-idle"},
		{"", "is-idle"},
	}
	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			if got := pillClassForState(tt.state); got != tt.want {
				t.Errorf("pillClassForState(%q) = %q, want %q", tt.state, got, tt.want)
			}
		})
	}
}

func TestPillClassForStatus(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{operations.StatusRunning, "is-running"},
		{operations.StatusOK, "is-ok"},
		{operations.StatusError, "is-error"},
		{operations.StatusCancelled, "is-warn"},
		{"unknown", "is-idle"},
		{"", "is-idle"},
	}
	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			if got := pillClassForStatus(tt.status); got != tt.want {
				t.Errorf("pillClassForStatus(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestRelTime(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"sub-second", 500 * time.Millisecond, "just now"},
		{"exactly zero", 0, "just now"},
		{"seconds", 5 * time.Second, "5s ago"},
		{"just under a minute", 59 * time.Second, "59s ago"},
		{"minutes", 2 * time.Minute, "2m ago"},
		{"just under an hour", 59 * time.Minute, "59m ago"},
		{"hours", 3 * time.Hour, "3h ago"},
		{"just under a day", 23 * time.Hour, "23h ago"},
		{"days", 50 * time.Hour, "2d ago"},
		// Negative input is normalised to its absolute value.
		{"negative seconds", -5 * time.Second, "5s ago"},
		{"negative sub-second", -100 * time.Millisecond, "just now"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := relTime(tt.d); got != tt.want {
				t.Errorf("relTime(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestRelDuration(t *testing.T) {
	tests := []struct {
		name string
		d    time.Duration
		want string
	}{
		{"sub-millisecond", 500 * time.Microsecond, "<1ms"},
		{"zero", 0, "<1ms"},
		{"milliseconds", 45 * time.Millisecond, "45ms"},
		{"seconds", 2300 * time.Millisecond, "2.3s"},
		{"minutes and seconds", 72 * time.Second, "1m 12s"},
		{"hours and minutes", 90 * time.Minute, "1h 30m"},
		// Negative input is normalised to its absolute value.
		{"negative ms", -45 * time.Millisecond, "45ms"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := relDuration(tt.d); got != tt.want {
				t.Errorf("relDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestEncodeParamsJSON(t *testing.T) {
	if got := encodeParamsJSON(nil); got != "" {
		t.Errorf("encodeParamsJSON(nil) = %q, want empty", got)
	}
	if got := encodeParamsJSON(map[string]any{}); got != "" {
		t.Errorf("encodeParamsJSON(empty) = %q, want empty", got)
	}
	got := encodeParamsJSON(map[string]any{"force": true, "draft": false})
	if got == "" {
		t.Fatalf("encodeParamsJSON(non-empty) returned empty")
	}
	if !strings.Contains(got, "\"force\"") || !strings.Contains(got, "\"draft\"") {
		t.Errorf("encodeParamsJSON output missing keys: %q", got)
	}
	// MarshalIndent uses two-space indentation.
	if !strings.Contains(got, "\n  ") {
		t.Errorf("encodeParamsJSON output not indented: %q", got)
	}
}

func TestHistoryFilterFromQuery(t *testing.T) {
	tests := []struct {
		name       string
		query      map[string][]string
		wantName   string
		wantKind   string
		wantStatus string
	}{
		{
			name:       "all empty",
			query:      map[string][]string{},
			wantName:   "",
			wantKind:   "",
			wantStatus: "",
		},
		{
			name: "valid kind and status with surrounding spaces",
			query: map[string][]string{
				"name":   {"  build  "},
				"kind":   {"  task  "},
				"status": {"  ok  "},
			},
			wantName:   "build",
			wantKind:   operations.KindTask,
			wantStatus: operations.StatusOK,
		},
		{
			name: "valid service kind and running status",
			query: map[string][]string{
				"kind":   {operations.KindService},
				"status": {operations.StatusRunning},
			},
			wantKind:   operations.KindService,
			wantStatus: operations.StatusRunning,
		},
		{
			name: "invalid kind and status dropped from filter",
			query: map[string][]string{
				"kind":   {"bogus"},
				"status": {"nope"},
			},
			// The view rehydrates the raw (trimmed) values, but the typed
			// Filter drops unrecognised kind/status.
			wantKind:   "",
			wantStatus: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, view := historyFilterFromQuery(tt.query)
			if f.Name != tt.wantName {
				t.Errorf("Filter.Name = %q, want %q", f.Name, tt.wantName)
			}
			if f.Kind != tt.wantKind {
				t.Errorf("Filter.Kind = %q, want %q", f.Kind, tt.wantKind)
			}
			if f.Status != tt.wantStatus {
				t.Errorf("Filter.Status = %q, want %q", f.Status, tt.wantStatus)
			}
			if f.Limit != 100 {
				t.Errorf("Filter.Limit = %d, want 100", f.Limit)
			}
			// The view always carries the trimmed raw values.
			if tt.name == "valid kind and status with surrounding spaces" {
				if view.Name != "build" || view.Kind != "task" || view.Status != "ok" {
					t.Errorf("view = %+v, want trimmed values", view)
				}
			}
			if tt.name == "invalid kind and status dropped from filter" {
				if view.Kind != "bogus" || view.Status != "nope" {
					t.Errorf("view should retain raw values, got %+v", view)
				}
			}
		})
	}
}

func TestFirstValue(t *testing.T) {
	m := map[string][]string{
		"a":     {"first", "second"},
		"empty": {},
	}
	if got := firstValue(m, "a"); got != "first" {
		t.Errorf("firstValue(a) = %q, want first", got)
	}
	if got := firstValue(m, "empty"); got != "" {
		t.Errorf("firstValue(empty) = %q, want empty", got)
	}
	if got := firstValue(m, "missing"); got != "" {
		t.Errorf("firstValue(missing) = %q, want empty", got)
	}
}

func TestFindOperationView(t *testing.T) {
	views := []OperationView{
		{Name: "build"},
		{Name: "deploy"},
	}
	if got := findOperationView(views, "deploy"); got.Name != "deploy" {
		t.Errorf("findOperationView(deploy).Name = %q, want deploy", got.Name)
	}
	// Not found returns the zero value.
	if got := findOperationView(views, "nope"); got.Name != "" {
		t.Errorf("findOperationView(nope) = %+v, want zero value", got)
	}
	if got := findOperationView(nil, "build"); got.Name != "" {
		t.Errorf("findOperationView(nil) = %+v, want zero value", got)
	}
}

func TestFlowNodesFromViews(t *testing.T) {
	views := []OperationView{
		{Name: "build"},
		{Name: "deploy"},
	}
	nodes := flowNodesFromViews(views)
	if len(nodes) != len(actionFlow) {
		t.Fatalf("flowNodesFromViews len = %d, want %d", len(nodes), len(actionFlow))
	}
	for i, node := range nodes {
		wantLast := i == len(actionFlow)-1
		if node.IsLast != wantLast {
			t.Errorf("node[%d].IsLast = %v, want %v", i, node.IsLast, wantLast)
		}
		// Op.Name should reflect a lookup against actionFlow ordering;
		// when present in views it carries the name, otherwise zero.
		if node.Op.Name != "" && node.Op.Name != actionFlow[i] {
			t.Errorf("node[%d].Op.Name = %q, want %q or empty", i, node.Op.Name, actionFlow[i])
		}
	}
	// The "build" and "deploy" entries should have been matched.
	matched := map[string]bool{}
	for _, node := range nodes {
		if node.Op.Name != "" {
			matched[node.Op.Name] = true
		}
	}
	if !matched["build"] || !matched["deploy"] {
		t.Errorf("expected build and deploy matched, got %v", matched)
	}
}

func TestOperationViewFromSnapshot(t *testing.T) {
	now := time.Now()

	t.Run("idle with successful last run", func(t *testing.T) {
		snap := operations.Snapshot{
			Definition: operations.Definition{
				Name:        "build",
				Kind:        operations.KindTask,
				Description: "Build the site",
			},
			State: operations.StateIdle,
			LastRun: &operations.HistoryRun{
				Status:    operations.StatusOK,
				StartedAt: now.Add(-2 * time.Minute),
				EndedAt:   now.Add(-2*time.Minute + 3*time.Second),
			},
		}
		v := operationViewFromSnapshot(snap, now)
		if v.Running {
			t.Error("expected Running=false for idle snapshot")
		}
		if !v.HasLastRun {
			t.Error("expected HasLastRun=true")
		}
		if v.PillClass != "is-idle" {
			t.Errorf("PillClass = %q, want is-idle", v.PillClass)
		}
		if v.LastDurationLabel == "" {
			t.Error("expected LastDurationLabel set when EndedAt non-zero")
		}
		if v.IconName != "build" {
			t.Errorf("IconName = %q, want build", v.IconName)
		}
	})

	t.Run("running snapshot", func(t *testing.T) {
		snap := operations.Snapshot{
			Definition: operations.Definition{Name: "serve", Kind: operations.KindService},
			State:      operations.StateRunning,
			Active: &operations.Run{
				StartedAt: now.Add(-10 * time.Second),
			},
		}
		v := operationViewFromSnapshot(snap, now)
		if !v.Running {
			t.Error("expected Running=true when Active != nil")
		}
		if v.UptimeLabel == "" {
			t.Error("expected UptimeLabel set for running snapshot")
		}
		if v.PillClass != "is-running" {
			t.Errorf("PillClass = %q, want is-running", v.PillClass)
		}
	})

	t.Run("error on finished run overrides pill", func(t *testing.T) {
		snap := operations.Snapshot{
			Definition: operations.Definition{Name: "deploy", Kind: operations.KindTask},
			State:      operations.StateIdle,
			LastRun: &operations.HistoryRun{
				Status:    operations.StatusError,
				StartedAt: now.Add(-time.Minute),
				EndedAt:   now.Add(-time.Minute + time.Second),
				Error:     "boom",
			},
		}
		v := operationViewFromSnapshot(snap, now)
		if v.Running {
			t.Error("expected Running=false")
		}
		if v.PillClass != "is-error" {
			t.Errorf("PillClass = %q, want is-error (error overrides)", v.PillClass)
		}
		if v.StateLabel != "error" {
			t.Errorf("StateLabel = %q, want error", v.StateLabel)
		}
		if v.State != string(operations.StateError) {
			t.Errorf("State = %q, want error", v.State)
		}
	})
}

func TestOperationsViewFromRunner(t *testing.T) {
	// Nil runner yields an empty (non-nil) slice.
	if got := operationsViewFromRunner(nil); len(got) != 0 {
		t.Errorf("operationsViewFromRunner(nil) = %v, want empty", got)
	}

	runner := newTestRunner(t)

	// Trigger the long-running serve service and observe a Running view.
	if _, err := runner.Trigger("serve", nil); err != nil {
		t.Fatalf("Trigger(serve): %v", err)
	}

	var serveView OperationView
	var found bool
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		views := operationsViewFromRunner(runner)
		serveView = findOperationView(views, "serve")
		if serveView.Name == "serve" && serveView.Running {
			found = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !found {
		t.Fatalf("serve view never became Running: %+v", serveView)
	}
	if serveView.PillClass != "is-running" {
		t.Errorf("serve PillClass = %q, want is-running", serveView.PillClass)
	}

	// Stopping serve cancels it; the run finishes as Cancelled and is no
	// longer Running.
	if err := runner.Stop("serve"); err != nil {
		t.Fatalf("Stop(serve): %v", err)
	}
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		views := operationsViewFromRunner(runner)
		serveView = findOperationView(views, "serve")
		if serveView.Name == "serve" && !serveView.Running {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if serveView.Running {
		t.Errorf("serve still Running after Stop: %+v", serveView)
	}

	// The full set of definitions is represented.
	views := operationsViewFromRunner(runner)
	names := map[string]bool{}
	for _, v := range views {
		names[v.Name] = true
	}
	for _, want := range []string{"build", "deploy", "check", "serve"} {
		if !names[want] {
			t.Errorf("missing operation view %q (got %v)", want, names)
		}
	}
}

func TestHistoryRowsFromRuns(t *testing.T) {
	now := time.Now()
	runs := []operations.HistoryRun{
		{
			ID:        1,
			Name:      "build",
			Kind:      operations.KindTask,
			StartedAt: now.Add(-time.Minute),
			EndedAt:   now.Add(-time.Minute + 5*time.Second),
			Status:    operations.StatusOK,
		},
		{
			ID:        2,
			Name:      "serve",
			Kind:      operations.KindService,
			StartedAt: now,
			// EndedAt zero: still running, no duration label.
			Status: operations.StatusRunning,
			Error:  "",
		},
	}
	rows := historyRowsFromRuns(runs)
	if len(rows) != 2 {
		t.Fatalf("historyRowsFromRuns len = %d, want 2", len(rows))
	}

	finished := rows[0]
	if finished.DurationLabel == "" {
		t.Error("expected DurationLabel set when EndedAt non-zero")
	}
	if finished.PillClass != "is-ok" {
		t.Errorf("finished.PillClass = %q, want is-ok", finished.PillClass)
	}
	if finished.IconName != "build" {
		t.Errorf("finished.IconName = %q, want build", finished.IconName)
	}

	running := rows[1]
	if running.DurationLabel != "" {
		t.Errorf("running.DurationLabel = %q, want empty (EndedAt zero)", running.DurationLabel)
	}
	if running.PillClass != "is-running" {
		t.Errorf("running.PillClass = %q, want is-running", running.PillClass)
	}
	if running.IconName != "serve" {
		t.Errorf("running.IconName = %q, want serve", running.IconName)
	}
}

func TestSchedulerRunsFromHistory(t *testing.T) {
	due := time.Date(2026, 5, 29, 12, 0, 0, 0, time.UTC)
	started := time.Date(2026, 5, 29, 12, 0, 3, 0, time.UTC)

	rows := []operations.HistoryRun{
		{
			// due_at present and parseable -> DueAt comes from params.
			Name:      "scheduler:trigger",
			Status:    operations.StatusOK,
			StartedAt: started,
			Params:    map[string]any{"due_at": due.Format(time.RFC3339Nano)},
		},
		{
			// No due_at -> falls back to StartedAt.
			Name:      "scheduler:trigger",
			Status:    operations.StatusError,
			Error:     "failed",
			StartedAt: started,
			Params:    map[string]any{},
		},
		{
			// Unparseable due_at -> falls back to StartedAt.
			Name:      "scheduler:trigger",
			Status:    operations.StatusOK,
			StartedAt: started,
			Params:    map[string]any{"due_at": "not-a-time"},
		},
	}
	out := schedulerRunsFromHistory(rows)
	if len(out) != 3 {
		t.Fatalf("schedulerRunsFromHistory len = %d, want 3", len(out))
	}

	if !out[0].DueAt.Equal(due) {
		t.Errorf("out[0].DueAt = %v, want %v (from params)", out[0].DueAt, due)
	}
	if !out[0].RanAt.Equal(started) {
		t.Errorf("out[0].RanAt = %v, want %v", out[0].RanAt, started)
	}

	if !out[1].DueAt.Equal(started) {
		t.Errorf("out[1].DueAt = %v, want fallback %v", out[1].DueAt, started)
	}
	if out[1].Error != "failed" {
		t.Errorf("out[1].Error = %q, want failed", out[1].Error)
	}

	if !out[2].DueAt.Equal(started) {
		t.Errorf("out[2].DueAt = %v, want fallback %v (unparseable due_at)", out[2].DueAt, started)
	}
}
