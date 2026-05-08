package app

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"

	"osg/internal/config"
	"osg/internal/logging"
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

	supervisor := ui.NewSupervisor(buildServiceMetas(opts, cfg))

	srv, err := ui.NewServer(ui.ServerOptions{
		Addr:       addr,
		Version:    Version,
		Cfg:        cfg,
		ConfigPath: opts.ConfigPath,
		Logger:     logger,
		Supervisor: supervisor,
	})
	if err != nil {
		return err
	}
	return srv.Run(ctx)
}

// buildServiceMetas returns the runner closures the supervisor will invoke
// when the user clicks Start. The runners reuse RunServe / RunAPI directly
// so behavior matches the standalone commands.
func buildServiceMetas(parent CLIOptions, cfg config.Config) []ui.ServiceMeta {
	serveAddr := parent.ServeAddr
	if serveAddr == "" {
		serveAddr = ":1313"
	}
	apiAddr := cfg.Interactions.Listen
	if apiAddr == "" {
		apiAddr = ":8090"
	}

	return []ui.ServiceMeta{
		{
			Name:        "serve",
			Description: "Preview server for the public/ directory",
			Addr:        serveAddr,
			Runner: func(ctx context.Context, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.ServeAddr = serveAddr
				o.Progress = nil
				return RunServe(ctx, o)
			},
		},
		{
			Name:        "api",
			Description: "Standalone interactions API (comments, likes, views)",
			Addr:        apiAddr,
			Runner: func(ctx context.Context, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				return RunAPI(ctx, o, APIOptions{Listen: apiAddr})
			},
		},
		{
			Name:        "watcher",
			Description: "Watches the vault and rebuilds on changes (no preview server)",
			Addr:        "(filesystem)",
			Runner: func(ctx context.Context, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				return RunWatcher(ctx, o)
			},
		},
		{
			Name:        "scheduler",
			Description: "Rebuilds when a future publish_at date becomes due",
			Addr:        "(timer)",
			Runner: func(ctx context.Context, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				return RunScheduler(ctx, o)
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
