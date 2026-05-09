package app

import (
	"context"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"strings"

	"osg/internal/config"
	"osg/internal/logging"
	"osg/internal/operations"
	"osg/internal/ui"
)

// RunUI starts the local web dashboard (`osg ui`). The dashboard is
// loopback-only: any non-loopback bind address is rejected.
func RunUI(ctx context.Context, opts CLIOptions) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	addr := opts.UIAddr
	if addr == "" {
		addr = cfg.UI.Addr
	}
	if addr == "" {
		addr = ":1314"
	}
	addr, err = normalizeLoopbackAddr(addr)
	if err != nil {
		return err
	}

	logger := logging.NewWithWriter(cfg.Logging, opts.Verbose, opts.LogWriter)
	logger.Info("starting ui dashboard", "addr", addr)

	store, err := operations.NewStore(OperationsDBPath(opts.ConfigPath))
	if err != nil {
		logger.Warn("operations audit store unavailable", "error", err)
	}
	if store != nil {
		defer func() { _ = store.Close() }()
	}

	runner := operations.New(buildOperationDefinitions(opts, cfg, store), store)

	srv, err := ui.NewServer(ui.ServerOptions{
		Addr:       addr,
		Version:    Version,
		Cfg:        cfg,
		ConfigPath: opts.ConfigPath,
		Logger:     logger,
		Runner:     runner,
	})
	if err != nil {
		return err
	}
	return srv.Run(ctx)
}

// OperationsDBPath returns the unified audit DB path. Lives in the same
// .osg/ directory as the rest of OSG state.
func OperationsDBPath(configPath string) string {
	dir := filepath.Dir(configPath)
	if dir == "" || dir == "." {
		return ".osg/operations.db"
	}
	return filepath.Join(dir, ".osg", "operations.db")
}

// buildOperationDefinitions enumerates every operation the dashboard can
// drive: the four long-running services (serve, api, watcher, scheduler)
// and the build task that powers /assets' "Rebuild now" button. Each
// definition wraps the corresponding app.RunX with the dashboard's
// CLIOptions baseline.
func buildOperationDefinitions(parent CLIOptions, cfg config.Config, store *operations.Store) []operations.Definition {
	serveAddr := parent.ServeAddr
	if serveAddr == "" {
		serveAddr = ":1313"
	}
	apiAddr := cfg.Interactions.Listen
	if apiAddr == "" {
		apiAddr = ":8090"
	}

	return []operations.Definition{
		{
			Name:        "serve",
			Kind:        operations.KindService,
			Description: "Preview server for the public/ directory",
			Addr:        serveAddr,
			Run: func(ctx context.Context, _ map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.ServeAddr = serveAddr
				o.Progress = nil
				return RunServe(ctx, o)
			},
		},
		{
			Name:        "api",
			Kind:        operations.KindService,
			Description: "Standalone interactions API (comments, likes, views)",
			Addr:        apiAddr,
			Run: func(ctx context.Context, _ map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				return RunAPI(ctx, o, APIOptions{Listen: apiAddr})
			},
		},
		{
			Name:        "watcher",
			Kind:        operations.KindService,
			Description: "Watches the vault and rebuilds on changes (no preview server)",
			Addr:        "(filesystem)",
			Run: func(ctx context.Context, _ map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				return RunWatcher(ctx, o)
			},
		},
		{
			Name:        "scheduler",
			Kind:        operations.KindService,
			Description: "Rebuilds when a future publish_at date becomes due",
			Addr:        "(timer)",
			Run: func(ctx context.Context, _ map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				return RunScheduler(ctx, o, store)
			},
		},
		{
			Name:        "build",
			Kind:        operations.KindTask,
			Description: "Render the site to public/ (one-shot)",
			Run: func(ctx context.Context, _ map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				o.SkipAI = true
				return RunBuild(ctx, o)
			},
		},
	}
}

// normalizeLoopbackAddr ensures the dashboard binds only to loopback. Empty
// host (e.g. ":1314") is rewritten to 127.0.0.1 to give a friendly default
// without exposing the dashboard to the network. Any explicit non-loopback
// host is rejected.
func normalizeLoopbackAddr(addr string) (string, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", fmt.Errorf("invalid ui addr %q: %w", addr, err)
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return net.JoinHostPort("127.0.0.1", port), nil
	}
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return addr, nil
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return addr, nil
	}
	return "", fmt.Errorf("ui addr %q must be loopback (127.0.0.1, ::1, or localhost)", addr)
}
