package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"osg/internal/config"
)

type viewData struct {
	Title    string
	Active   string
	Version  string
	Services []Service
	Now      time.Time
	State
}

func (s *Server) buildView(active, title string, r *http.Request) viewData {
	st := Collect(r.Context(), s.opts.Cfg, s.opts.Logger)
	v := viewData{
		Title:   title,
		Active:  active,
		Version: s.opts.Version,
		Now:     time.Now(),
		State:   st,
	}
	if s.opts.Supervisor != nil {
		v.Services = s.opts.Supervisor.Snapshot()
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
// persists the change to config.yaml via config.UpdatePluginsEnabled,
// which round-trips the YAML preserving comments. The in-memory cfg is
// also updated so subsequent renders show the new state without needing
// a Load() round trip.
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
	http.Redirect(w, r, "/plugins", http.StatusSeeOther)
}

func (s *Server) handleServices(w http.ResponseWriter, r *http.Request) {
	s.render(w, "services", s.buildView("services", "Services", r))
}

func (s *Server) handleServiceStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if s.opts.Supervisor == nil {
		http.Error(w, "supervisor not configured", http.StatusServiceUnavailable)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}
	if err := s.opts.Supervisor.Start(name); err != nil {
		if s.opts.Logger != nil {
			s.opts.Logger.Warn("service start failed", "name", name, "error", err)
		}
		// Surface the error inline by re-rendering the services page
		// after the supervisor has already recorded LastError on the
		// Service entry. Rendering returns 200 so the user sees the
		// page with the error pill rather than a bare error string.
	}
	http.Redirect(w, r, "/services", http.StatusSeeOther)
}

// servicesJSON is the polled response shape used by the /services page to
// keep state pills and uptime live without a full reload. Logs are served
// over SSE (see handleServiceLogs) and not duplicated here.
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
	if s.opts.Supervisor == nil {
		http.Error(w, "supervisor not configured", http.StatusServiceUnavailable)
		return
	}
	now := time.Now()
	snap := s.opts.Supervisor.Snapshot()
	out := servicesJSON{Now: now.Format(time.RFC3339)}
	for _, svc := range snap {
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
	if s.opts.Supervisor == nil {
		http.Error(w, "supervisor not configured", http.StatusServiceUnavailable)
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
	w.Header().Set("X-Accel-Buffering", "no") // disable proxy buffering
	w.WriteHeader(http.StatusOK)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	ch := s.opts.Supervisor.Logs(ctx, name)
	if ch == nil {
		_, _ = w.Write([]byte("event: error\ndata: unknown service\n\n"))
		flusher.Flush()
		return
	}

	// Heartbeat keeps the connection open through proxies and lets the
	// client detect a closed server quickly.
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
	// SSE requires each line of the data field to be prefixed with "data: ".
	// We collapse newlines into single-line events; OSG logs are JSON or
	// plaintext one per line so this is safe.
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
	if s.opts.Supervisor == nil {
		http.Error(w, "supervisor not configured", http.StatusServiceUnavailable)
		return
	}
	name := r.FormValue("name")
	if name == "" {
		http.Error(w, "missing name", http.StatusBadRequest)
		return
	}
	if err := s.opts.Supervisor.Stop(name); err != nil {
		if s.opts.Logger != nil {
			s.opts.Logger.Warn("service stop failed", "name", name, "error", err)
		}
	}
	http.Redirect(w, r, "/services", http.StatusSeeOther)
}
