package ui

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"log/slog"
	"net/http"
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
}

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
		Handler:           s.mux,
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
	s.mux.HandleFunc("POST /operations/{name}/stop", s.handleOperationStop)
	s.mux.HandleFunc("GET /operations/{name}/logs", s.handleOperationLogs)
	s.mux.HandleFunc("GET /operations.json", s.handleOperationsJSON)
}

// loadTemplates parses each page template against the layout and the
// shared partial set, keyed by page name. Page templates can therefore
// invoke any `{{template "<partial>.html" .}}` without per-page parsing.
//
// The "fragments" key in the result map is special: it is parsed
// without the layout so individual partials (e.g. drawer.html) can be
// rendered directly into HTMX targets.
func loadTemplates() (map[string]*template.Template, error) {
	pages := []string{"dashboard", "actions", "history", "vault", "plugins", "assets", "scheduler", "services"}
	partials := []string{
		"templates/partials/quick-button.html",
		"templates/partials/operation-card.html",
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
