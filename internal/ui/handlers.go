package ui

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
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
