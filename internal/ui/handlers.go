package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"osg/internal/config"
	"osg/internal/operations"
)

type viewData struct {
	Title         string
	Active        string
	Version       string
	Services      []Service
	Now           time.Time
	Assets        []AssetEntry
	AssetSummary  AssetSummary
	SchedulerRuns []SchedulerRun
	Rebuild       RebuildSnapshot
	State
}

func (s *Server) buildView(active, title string, r *http.Request) viewData {
	st := Collect(r.Context(), s.opts.Cfg, s.opts.Logger)
	v := viewData{
		Title:    title,
		Active:   active,
		Version:  s.opts.Version,
		Now:      time.Now(),
		State:    st,
		Services: servicesFromRunner(s.opts.Runner),
	}
	return v
}

func (s *Server) render(w http.ResponseWriter, name string, data viewData) {
	tpl, ok := s.templates[name]
	if !ok {
		http.Error(w, "template not found: "+name, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "layout", data); err != nil {
		if s.opts.Logger != nil {
			s.opts.Logger.Error("render failed", "template", name, "error", err)
		}
	}
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	s.render(w, "dashboard", s.buildView("dashboard", "Dashboard", r))
}

func (s *Server) handleVault(w http.ResponseWriter, r *http.Request) {
	s.render(w, "vault", s.buildView("vault", "Vault", r))
}

func (s *Server) handlePlugins(w http.ResponseWriter, r *http.Request) {
	s.render(w, "plugins", s.buildView("plugins", "Plugins", r))
}

// handlePluginToggle flips the enabled state of a single plugin and
// persists the change to config.yaml via config.UpdatePluginsEnabled.
// Running services that load plugins are restarted so the new list takes
// effect immediately.
func (s *Server) handlePluginToggle(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "missing plugin name", http.StatusBadRequest)
		return
	}

	enabled := append([]string(nil), s.opts.Cfg.PluginsEnabled...)
	found := false
	for i, n := range enabled {
		if n == name {
			enabled = append(enabled[:i], enabled[i+1:]...)
			found = true
			break
		}
	}
	if !found {
		enabled = append(enabled, name)
	}

	cfgPath := s.opts.ConfigPath
	if cfgPath == "" {
		cfgPath = "config.yaml"
	}
	if err := config.UpdatePluginsEnabled(cfgPath, enabled); err != nil {
		if s.opts.Logger != nil {
			s.opts.Logger.Warn("plugin toggle persist failed", "plugin", name, "error", err)
		}
		http.Error(w, "persist failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	s.opts.Cfg.PluginsEnabled = enabled

	if s.opts.Logger != nil {
		action := "enabled"
		if found {
			action = "disabled"
		}
		s.opts.Logger.Info("plugin toggled", "plugin", name, "action", action)
	}

	if s.opts.Runner != nil {
		for _, svcName := range []string{"serve", "watcher", "scheduler"} {
			if err := s.opts.Runner.Restart(svcName); err != nil && s.opts.Logger != nil {
				s.opts.Logger.Warn("plugin toggle restart failed", "service", svcName, "error", err)
			}
		}
	}

	http.Redirect(w, r, "/plugins", http.StatusSeeOther)
}

func (s *Server) handleScheduler(w http.ResponseWriter, r *http.Request) {
	v := s.buildView("scheduler", "Scheduler", r)
	if s.opts.Runner != nil {
		rows, err := s.opts.Runner.History(operations.Filter{Name: "scheduler:trigger", Limit: 50})
		if err != nil && s.opts.Logger != nil {
			s.opts.Logger.Warn("scheduler history fetch failed", "error", err)
		}
		v.SchedulerRuns = schedulerRunsFromHistory(rows)
	}
	s.render(w, "scheduler", v)
}

func (s *Server) handleAssets(w http.ResponseWriter, r *http.Request) {
	v := s.buildView("assets", "Assets", r)
	v.Assets, v.AssetSummary = collectAssets(s.opts.Cfg, s.opts.Logger)
	v.Rebuild = rebuildSnapshotFromRunner(s.opts.Runner)
	s.render(w, "assets", v)
}

// handleRebuild triggers a "build" task via the runner and redirects to
// /assets so the user immediately sees the running state.
func (s *Server) handleRebuild(w http.ResponseWriter, r *http.Request) {
	if s.opts.Runner == nil {
		http.Error(w, "runner not configured", http.StatusServiceUnavailable)
		return
	}
	if _, err := s.opts.Runner.Trigger("build", nil); err != nil {
		if s.opts.Logger != nil {
			s.opts.Logger.Warn("rebuild trigger failed", "error", err)
		}
	}
	http.Redirect(w, r, "/assets", http.StatusSeeOther)
}

// handleRebuildJSON returns the current rebuild state for client-side
// polling. Sourced from the runner's snapshot of the "build" operation.
func (s *Server) handleRebuildJSON(w http.ResponseWriter, r *http.Request) {
	snap := rebuildSnapshotFromRunner(s.opts.Runner)
	out := map[string]any{
		"available": snap.Available,
		"running":   snap.Running,
	}
	if !snap.LastRan.IsZero() {
		out["last_ran"] = snap.LastRan.Format(time.RFC3339)
		out["duration_ms"] = snap.Duration.Milliseconds()
		out["last_error"] = snap.LastError
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleServices(w http.ResponseWriter, r *http.Request) {
	s.render(w, "services", s.buildView("services", "Services", r))
}

func (s *Server) handleServiceStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.opts.Runner == nil {
		http.Error(w, "runner not configured", http.StatusServiceUnavailable)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}
	if _, err := s.opts.Runner.Trigger(name, nil); err != nil {
		if s.opts.Logger != nil {
			s.opts.Logger.Warn("service start failed", "name", name, "error", err)
		}
	}
	http.Redirect(w, r, "/services", http.StatusSeeOther)
}

// servicesJSON is the polled response shape used by the /services page to
// keep state pills and uptime live without a full reload.
type servicesJSON struct {
	Services []serviceJSON `json:"services"`
	Now      string        `json:"now"`
}

type serviceJSON struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	StartedAt string `json:"started_at,omitempty"`
	Uptime    string `json:"uptime,omitempty"`
	LastError string `json:"last_error,omitempty"`
}

func (s *Server) handleServicesJSON(w http.ResponseWriter, r *http.Request) {
	if s.opts.Runner == nil {
		http.Error(w, "runner not configured", http.StatusServiceUnavailable)
		return
	}
	now := time.Now()
	out := servicesJSON{Now: now.Format(time.RFC3339)}
	for _, svc := range servicesFromRunner(s.opts.Runner) {
		entry := serviceJSON{
			Name:      svc.Name,
			State:     string(svc.State),
			LastError: svc.LastError,
		}
		if !svc.StartedAt.IsZero() && (svc.State == StateRunning || svc.State == StateStarting) {
			entry.StartedAt = svc.StartedAt.Format(time.RFC3339)
			entry.Uptime = uptimeStr(svc.StartedAt, now)
		}
		out.Services = append(out.Services, entry)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) handleServiceLogs(w http.ResponseWriter, r *http.Request) {
	if s.opts.Runner == nil {
		http.Error(w, "runner not configured", http.StatusServiceUnavailable)
		return
	}
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "missing service name", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	ch := s.opts.Runner.Logs(ctx, name)
	if ch == nil {
		_, _ = w.Write([]byte("event: error\ndata: not running\n\n"))
		flusher.Flush()
		return
	}

	tick := time.NewTicker(15 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-ch:
			if !ok {
				return
			}
			if err := writeSSE(w, "log", line); err != nil {
				return
			}
			flusher.Flush()
		case <-tick.C:
			if _, err := w.Write([]byte(": heartbeat\n\n")); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeSSE(w http.ResponseWriter, event, data string) error {
	data = strings.ReplaceAll(data, "\r", "")
	data = strings.ReplaceAll(data, "\n", " ")
	_, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, data)
	return err
}

func (s *Server) handleServiceStop(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.opts.Runner == nil {
		http.Error(w, "runner not configured", http.StatusServiceUnavailable)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}
	if err := s.opts.Runner.Stop(name); err != nil {
		if s.opts.Logger != nil {
			s.opts.Logger.Warn("service stop failed", "name", name, "error", err)
		}
	}
	http.Redirect(w, r, "/services", http.StatusSeeOther)
}

// schedulerRunsFromHistory turns operations.HistoryRun rows into the
// template-friendly SchedulerRun shape, extracting the `due_at` value
// from the run's params (falling back to ran/started time when absent).
func schedulerRunsFromHistory(rows []operations.HistoryRun) []SchedulerRun {
	out := make([]SchedulerRun, 0, len(rows))
	for _, r := range rows {
		run := SchedulerRun{
			RanAt:  r.StartedAt,
			Status: r.Status,
			Error:  r.Error,
		}
		if v, ok := r.Params["due_at"].(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
				run.DueAt = t
			} else if t, err := time.Parse(time.RFC3339, v); err == nil {
				run.DueAt = t
			}
		}
		if run.DueAt.IsZero() {
			run.DueAt = r.StartedAt
		}
		out = append(out, run)
	}
	return out
}
