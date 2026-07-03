package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// handleOperationRun is the generic Trigger endpoint. Form fields become
// the run's params: bool fields use "true" / "false", strings pass
// through. The /actions UI submits forms here in Etapa 3, but it is
// also safe to call with curl. On success returns 303 to the page that
// posted the form (Referer) or to / when absent.
func (s *Server) handleOperationRun(w http.ResponseWriter, r *http.Request) {
	if s.opts.Runner == nil {
		http.Error(w, "runner not configured", http.StatusServiceUnavailable)
		return
	}
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "missing operation name", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form: "+err.Error(), http.StatusBadRequest)
		return
	}
	params := paramsFromForm(r.PostForm)

	if _, err := s.opts.Runner.Trigger(name, params); err != nil {
		if s.opts.Logger != nil {
			s.opts.Logger.Warn("operation trigger failed", "name", name, "error", err)
		}
		// Surface the failure to API callers; HTML callers will see the
		// state error via the page they navigate back to.
		if wantsJSON(r) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusConflict)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}
	}
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "name": name})
		return
	}
	dest := r.Header.Get("Referer")
	if dest == "" {
		dest = "/"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// handleOperationRunFlow walks the canonical action flow starting at
// the named operation and triggers each downstream step in sequence in
// a background goroutine. Returns 303 immediately so the user can
// watch progress via /history; aborts on the first failure.
func (s *Server) handleOperationRunFlow(w http.ResponseWriter, r *http.Request) {
	if s.opts.Runner == nil {
		http.Error(w, "runner not configured", http.StatusServiceUnavailable)
		return
	}
	name := r.PathValue("name")
	chain := flowDownstream(name)
	if len(chain) == 0 {
		http.Error(w, "operation is not part of the action flow", http.StatusBadRequest)
		return
	}
	go func(seq []string) {
		failed, err := s.opts.Runner.RunFlow(seq)
		if err != nil && s.opts.Logger != nil {
			s.opts.Logger.Warn("run-flow aborted", "from", name, "failed_at", failed, "error", err)
		}
	}(chain)
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "chain": chain})
		return
	}
	dest := r.Header.Get("Referer")
	if dest == "" {
		dest = "/actions"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// handleOperationStop cancels the active run by name. 303 redirect to
// Referer for HTML callers, JSON when the client asked for it.
func (s *Server) handleOperationStop(w http.ResponseWriter, r *http.Request) {
	if s.opts.Runner == nil {
		http.Error(w, "runner not configured", http.StatusServiceUnavailable)
		return
	}
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "missing operation name", http.StatusBadRequest)
		return
	}
	if err := s.opts.Runner.Stop(name); err != nil {
		if s.opts.Logger != nil {
			s.opts.Logger.Warn("operation stop failed", "name", name, "error", err)
		}
	}
	if wantsJSON(r) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "name": name})
		return
	}
	dest := r.Header.Get("Referer")
	if dest == "" {
		dest = "/"
	}
	http.Redirect(w, r, dest, http.StatusSeeOther)
}

// handleOperationLogs streams SSE log events for the currently active
// run with the given name. Returns event=error when no run is in
// flight. Same shape as handleServiceLogs (which delegates internally
// in Etapa 3); kept separate for now to allow per-route customisation.
func (s *Server) handleOperationLogs(w http.ResponseWriter, r *http.Request) {
	if s.opts.Runner == nil {
		http.Error(w, "runner not configured", http.StatusServiceUnavailable)
		return
	}
	name := r.PathValue("name")
	if name == "" {
		http.Error(w, "missing operation name", http.StatusBadRequest)
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

// handleOperationCard renders a single card partial for the named
// operation. The poller calls this whenever it detects a state change,
// swapping the previous element so meta line and Run/Stop button stay
// in sync without a full page reload. The "style" query parameter
// selects the partial: flow-node, op-card, quick-button or task-form.
func (s *Server) handleOperationCard(w http.ResponseWriter, r *http.Request) {
	if s.opts.Runner == nil {
		http.Error(w, "runner not configured", http.StatusServiceUnavailable)
		return
	}
	name := r.PathValue("name")
	style := r.URL.Query().Get("style")
	tplName := ""
	switch style {
	case "op-card":
		// The card partial's file (and thus its template name) is
		// "operation-card.html", but the markup tags itself with
		// data-card-style="op-card"; map the style id to the real name.
		tplName = "operation-card.html"
	case "flow-node", "quick-button", "task-form":
		tplName = style + ".html"
	default:
		http.Error(w, "unknown card style", http.StatusBadRequest)
		return
	}
	views := operationsViewFromRunner(s.opts.Runner)
	view := findOperationView(views, name)
	if view.Name == "" {
		http.Error(w, "operation not found", http.StatusNotFound)
		return
	}
	s.renderFragment(w, tplName, view)
}

// handleOperationsJSON returns the runner's current snapshot for every
// definition. The /actions page polls this every couple of seconds to
// refresh state pills without reloading the page.
func (s *Server) handleOperationsJSON(w http.ResponseWriter, r *http.Request) {
	if s.opts.Runner == nil {
		http.Error(w, "runner not configured", http.StatusServiceUnavailable)
		return
	}
	now := time.Now()
	out := operationsJSON{Now: now.Format(time.RFC3339)}
	for _, snap := range s.opts.Runner.Snapshot() {
		entry := operationJSON{
			Name:        snap.Definition.Name,
			Kind:        snap.Definition.Kind,
			Description: snap.Definition.Description,
			Addr:        snap.Definition.Addr,
			State:       string(snap.State),
		}
		if snap.Active != nil {
			entry.StartedAt = snap.Active.StartedAt.Format(time.RFC3339)
			entry.Uptime = uptimeStr(snap.Active.StartedAt, now)
			entry.LastError = snap.Active.LastError
		}
		if snap.LastRun != nil {
			entry.LastRanAt = snap.LastRun.StartedAt.Format(time.RFC3339)
			entry.LastStatus = snap.LastRun.Status
			if !snap.LastRun.EndedAt.IsZero() {
				d := snap.LastRun.EndedAt.Sub(snap.LastRun.StartedAt)
				entry.LastDurationMs = d.Milliseconds()
			}
			if snap.LastRun.Error != "" && entry.LastError == "" {
				entry.LastError = snap.LastRun.Error
			}
		}
		out.Operations = append(out.Operations, entry)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(out)
}

type operationsJSON struct {
	Now        string          `json:"now"`
	Operations []operationJSON `json:"operations"`
}

type operationJSON struct {
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	Description    string `json:"description,omitempty"`
	Addr           string `json:"addr,omitempty"`
	State          string `json:"state"`
	StartedAt      string `json:"started_at,omitempty"`
	Uptime         string `json:"uptime,omitempty"`
	LastError      string `json:"last_error,omitempty"`
	LastRanAt      string `json:"last_ran_at,omitempty"`
	LastStatus     string `json:"last_status,omitempty"`
	LastDurationMs int64  `json:"last_duration_ms,omitempty"`
}

// paramsFromForm coerces form values into a typed map[string]any. Bool
// fields are detected by the literal "true"/"false" values; everything
// else stays as a string. Empty values are dropped so the runner sees
// only meaningful overrides.
func paramsFromForm(form map[string][]string) map[string]any {
	if len(form) == 0 {
		return nil
	}
	out := make(map[string]any, len(form))
	for k, vs := range form {
		if len(vs) == 0 {
			continue
		}
		v := vs[0]
		if v == "" {
			continue
		}
		switch v {
		case "true":
			out[k] = true
		case "false":
			out[k] = false
		default:
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				out[k] = n
			} else {
				out[k] = v
			}
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// wantsJSON returns true when the client's Accept header asks for JSON.
// Used to switch between HTML redirect (default) and JSON response on
// the trigger endpoints.
func wantsJSON(r *http.Request) bool {
	accept := r.Header.Get("Accept")
	for _, part := range splitAcceptHeader(accept) {
		if part == "application/json" {
			return true
		}
	}
	return false
}

func splitAcceptHeader(s string) []string {
	if s == "" {
		return nil
	}
	out := []string{}
	current := ""
	for _, ch := range s {
		switch ch {
		case ',', ';':
			// Drop anything after a `;q=...` quality marker too.
			if t := trimSpaceASCII(current); t != "" {
				out = append(out, t)
			}
			current = ""
		default:
			current += string(ch)
		}
	}
	if t := trimSpaceASCII(current); t != "" {
		out = append(out, t)
	}
	return out
}

func trimSpaceASCII(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
