package ui

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"osg/internal/config"
	"osg/internal/operations"
)

// ServerOptions configures a dashboard server instance.
type ServerOptions struct {
	Addr       string
	Version    string
	Cfg        config.Config
	ConfigPath string
	Logger     *slog.Logger
	// Runner is the unified operations runner driving services and
	// tasks. The dashboard is read-only when Runner is nil.
	Runner *operations.Runner
	// StateTTL is how long a Collect snapshot (vault stats, pages,
	// plugin metadata) is reused across requests before being
	// recomputed. Zero means the default; negative disables caching.
	StateTTL time.Duration
}

// defaultStateTTL bounds how stale the dashboard state may be. Collect
// re-parses the whole vault and recompiles every WASM plugin, so serving
// each request from a short-lived snapshot keeps the dashboard responsive.
const defaultStateTTL = 5 * time.Second

// SchedulerRun is the template-friendly view of one scheduler trigger
// row. Built from operations.HistoryRun in handleScheduler.
type SchedulerRun struct {
	DueAt  time.Time
	RanAt  time.Time
	Status string
	Error  string
}

// Server is the OSG UI dashboard server.
type Server struct {
	opts      ServerOptions
	templates map[string]*template.Template
	mux       *http.ServeMux

	stateMu sync.Mutex
	state   State
	stateAt time.Time
}

// collectState returns the current State, reusing the last snapshot while
// it is younger than StateTTL. The mutex also single-flights concurrent
// requests so only one Collect runs at a time.
func (s *Server) collectState(ctx context.Context) State {
	ttl := s.opts.StateTTL
	if ttl == 0 {
		ttl = defaultStateTTL
	}
	s.stateMu.Lock()
	defer s.stateMu.Unlock()
	if ttl > 0 && !s.stateAt.IsZero() && time.Since(s.stateAt) < ttl {
		return s.state
	}
	// Detach from the request context: a client disconnect mid-Collect
	// must not poison the cached snapshot with partial data.
	st := Collect(context.WithoutCancel(ctx), s.opts.Cfg, s.opts.Logger)
	s.state = st
	s.stateAt = time.Now()
	return st
}

// invalidateState drops the cached snapshot so the next request recomputes
// it. Called after state-changing requests (plugin toggles, page edits,
// theme changes) so their redirect target renders fresh data.
func (s *Server) invalidateState() {
	s.stateMu.Lock()
	s.stateAt = time.Time{}
	s.stateMu.Unlock()
}

// NewServer prepares the dashboard server: parses templates and registers
// routes. Returns an error if templates fail to compile.
func NewServer(opts ServerOptions) (*Server, error) {
	if opts.Logger == nil {
		opts.Logger = slog.Default()
	}
	tpls, err := loadTemplates()
	if err != nil {
		return nil, err
	}
	s := &Server{
		opts:      opts,
		templates: tpls,
		mux:       http.NewServeMux(),
	}
	s.routes()
	return s, nil
}

// Run starts the HTTP server and blocks until ctx is cancelled or the
// server returns. Performs graceful shutdown with a 3s timeout. On
// shutdown the runner stops every active operation and the audit log
// rows for in-flight runs are marked as cancelled.
func (s *Server) Run(ctx context.Context) error {
	server := &http.Server{
		Addr:              s.opts.Addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- server.ListenAndServe() }()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		if s.opts.Runner != nil {
			s.opts.Runner.StopAll()
		}
		if err := <-errCh; err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case err := <-errCh:
		if s.opts.Runner != nil {
			s.opts.Runner.StopAll()
		}
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	return nil
}

// Handler returns the dashboard's HTTP handler wrapped in the security
// middleware. The dashboard runs every CLI command (build, deploy, imports,
// plugin installs), so it must not be reachable cross-origin.
func (s *Server) Handler() http.Handler {
	return s.securityMiddleware(s.mux)
}

// securityMiddleware protects the loopback dashboard against two browser-borne
// attacks that loopback binding alone does not stop:
//
//   - DNS rebinding: a remote page rebinds its hostname to 127.0.0.1. The TCP
//     connection is local, but the browser still sends the attacker's Host
//     header, so requiring a loopback Host rejects it.
//   - CSRF: a remote page auto-submits a form to http://127.0.0.1:<port>/…. The
//     browser tags such requests Sec-Fetch-Site: cross-site (and sends a
//     cross-origin Origin), both of which are rejected for state-changing
//     methods. Same-origin requests and non-browser clients (curl, no Origin)
//     are allowed.
func (s *Server) securityMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !isLoopbackHost(r.Host) {
			http.Error(w, "forbidden: dashboard is loopback-only", http.StatusForbidden)
			return
		}
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
			// Safe methods: no state change, no CSRF check.
		default:
			if site := r.Header.Get("Sec-Fetch-Site"); site != "" {
				if site != "same-origin" && site != "same-site" && site != "none" {
					http.Error(w, "forbidden: cross-site request blocked", http.StatusForbidden)
					return
				}
			} else if origin := r.Header.Get("Origin"); origin != "" {
				if u, err := url.Parse(origin); err != nil || !isLoopbackHost(u.Host) {
					http.Error(w, "forbidden: cross-origin request blocked", http.StatusForbidden)
					return
				}
			}
		}
		next.ServeHTTP(w, r)
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			// The request may have changed vault/plugin/theme state;
			// make the next page load recompute the snapshot.
			s.invalidateState()
		}
	})
}

// isLoopbackHost reports whether host (an HTTP Host or Origin authority,
// optionally with a port) names the loopback interface or "localhost".
func isLoopbackHost(host string) bool {
	if host == "" {
		return false
	}
	h, _, err := net.SplitHostPort(host)
	if err != nil {
		h = host // no port present
	}
	h = strings.TrimSuffix(strings.TrimPrefix(h, "["), "]")
	if strings.EqualFold(h, "localhost") {
		return true
	}
	if ip := net.ParseIP(h); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

func (s *Server) routes() {
	assetsSub, _ := fs.Sub(assetsFS, "assets")
	s.mux.Handle("/assets/", http.StripPrefix("/assets/", http.FileServer(http.FS(assetsSub))))
	s.mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok"))
	})
	s.mux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/assets/favicon.svg", http.StatusFound)
	})
	s.mux.HandleFunc("/", s.handleDashboard)
	s.mux.HandleFunc("/actions", s.handleActions)
	s.mux.HandleFunc("/history", s.handleHistory)
	s.mux.HandleFunc("/vault", s.handleVault)
	s.mux.HandleFunc("/plugins", s.handlePlugins)
	s.mux.HandleFunc("/assets", s.handleAssets)
	s.mux.HandleFunc("/scheduler", s.handleScheduler)
	s.mux.HandleFunc("/import", s.handleImport)
	s.mux.HandleFunc("/themes", s.handleThemes)
	s.mux.HandleFunc("/audit", s.handleAudit)
	s.mux.HandleFunc("GET /operations/{name}/drawer", s.handleOperationDrawer)
	s.mux.HandleFunc("GET /history/{id}/drawer", s.handleHistoryDrawer)
	s.mux.HandleFunc("POST /rebuild", s.handleRebuild)
	s.mux.HandleFunc("GET /rebuild.json", s.handleRebuildJSON)
	s.mux.HandleFunc("/services", s.handleServices)
	s.mux.HandleFunc("POST /plugins/toggle", s.handlePluginToggle)
	s.mux.HandleFunc("/services/start", s.handleServiceStart)
	s.mux.HandleFunc("/services/stop", s.handleServiceStop)
	s.mux.HandleFunc("GET /services.json", s.handleServicesJSON)
	s.mux.HandleFunc("GET /services/{name}/logs", s.handleServiceLogs)
	// Generic operations endpoints used by /actions cards (Etapa 3) and
	// safe to call directly with curl during Etapa 2 testing.
	s.mux.HandleFunc("POST /operations/{name}/run", s.handleOperationRun)
	s.mux.HandleFunc("POST /operations/{name}/run-flow", s.handleOperationRunFlow)
	s.mux.HandleFunc("POST /operations/{name}/stop", s.handleOperationStop)
	s.mux.HandleFunc("GET /operations/{name}/logs", s.handleOperationLogs)
	s.mux.HandleFunc("GET /operations/{name}/card", s.handleOperationCard)
	s.mux.HandleFunc("GET /operations.json", s.handleOperationsJSON)
	s.mux.HandleFunc("POST /summary/invalidate", s.handleSummaryInvalidate)
	s.mux.HandleFunc("POST /summary/save", s.handleSummarySave)
	s.mux.HandleFunc("POST /summary/regenerate", s.handleSummaryRegenerate)
	s.mux.HandleFunc("POST /summary/suggest", s.handleSummarySuggest)
	s.mux.HandleFunc("GET /vault/page", s.handleVaultPage)
}

// loadTemplates parses each page template against the layout and the
// shared partial set, keyed by page name. Page templates can therefore
// invoke any `{{template "<partial>.html" .}}` without per-page parsing.
//
// The "fragments" key in the result map is special: it is parsed
// without the layout so individual partials (e.g. drawer.html) can be
// rendered directly into HTMX targets.
func loadTemplates() (map[string]*template.Template, error) {
	pages := []string{"dashboard", "actions", "history", "vault", "page", "plugins", "assets", "scheduler", "services", "import", "themes", "audit"}
	partials := []string{
		"templates/partials/quick-button.html",
		"templates/partials/operation-card.html",
		"templates/partials/op-field.html",
		"templates/partials/task-form.html",
		"templates/partials/flow-node.html",
		"templates/partials/drawer.html",
	}
	out := make(map[string]*template.Template, len(pages)+1)

	funcs := template.FuncMap{
		"humanSize": humanSize,
		"uptime":    uptimeStr,
	}

	for _, name := range pages {
		files := append([]string{
			"templates/layout.html",
			"templates/" + name + ".html",
		}, partials...)
		t := template.New(name).Funcs(funcs)
		t, err := t.ParseFS(templatesFS, files...)
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", name, err)
		}
		out[name] = t
	}

	// Fragment tree: only the partials, no layout. Used to render drawer
	// content into HTMX targets.
	fragments := template.New("fragments").Funcs(funcs)
	fragments, err := fragments.ParseFS(templatesFS, partials...)
	if err != nil {
		return nil, fmt.Errorf("parse fragments: %w", err)
	}
	out["fragments"] = fragments
	return out, nil
}

func uptimeStr(start, now time.Time) string {
	if start.IsZero() {
		return "—"
	}
	d := now.Sub(start)
	if d < time.Second {
		return "<1s"
	}
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
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

func humanSize(n int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case n >= gb:
		return fmt.Sprintf("%.1f GB", float64(n)/gb)
	case n >= mb:
		return fmt.Sprintf("%.1f MB", float64(n)/mb)
	case n >= kb:
		return fmt.Sprintf("%.1f KB", float64(n)/kb)
	default:
		return fmt.Sprintf("%d B", n)
	}
}
