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
	logger.Info("starting ui dashboard",
		"addr", addr,
		"version", Version,
		"theme", cfg.Theme,
		"summary_strategy", cfg.SummaryStrategy,
		"ai_provider", cfg.AI.Provider,
		"ai_model", cfg.AI.Model,
		"ai_base_url", cfg.AI.BaseURL,
		"ai_timeout_s", cfg.AI.Timeout,
	)

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
			Name:        "init",
			Kind:        operations.KindTask,
			Description: "Scaffold project structure (config, content, theme)",
			Run: func(ctx context.Context, _ map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				return RunInit(ctx, o)
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
		{
			Name:        "update-content",
			Kind:        operations.KindTask,
			Description: "Sync content from the configured Obsidian vault",
			Run: func(ctx context.Context, _ map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				return RunUpdateContent(ctx, o)
			},
		},
		{
			Name:        "deploy",
			Kind:        operations.KindTask,
			Description: "Deploy the built site to the configured remote",
			Run: func(ctx context.Context, params map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				return RunDeploy(ctx, o, deployOptionsFromParams(params, cfg))
			},
		},
		{
			Name:        "check",
			Kind:        operations.KindTask,
			Description: "Validate content (links, images, frontmatter)",
			Run: func(ctx context.Context, params map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				return RunCheck(ctx, o, checkOptionsFromParams(params))
			},
		},
		{
			Name:        "audit",
			Kind:        operations.KindTask,
			Description: "Audit the rendered site for quality issues",
			Run: func(ctx context.Context, params map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				jsonOutput, _ := params["json"].(bool)
				return RunAudit(ctx, o, jsonOutput)
			},
		},
		{
			Name:        "new",
			Kind:        operations.KindTask,
			Description: "Create a new draft post in the vault",
			Run: func(ctx context.Context, params map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				postOpts, err := newPostOptionsFromParams(params)
				if err != nil {
					return err
				}
				return RunNew(ctx, o, postOpts)
			},
		},
		{
			Name:        "theme-init",
			Kind:        operations.KindTask,
			Description: "Scaffold a new theme (optionally inheriting from another)",
			Run: func(ctx context.Context, params map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				name, _ := params["name"].(string)
				if name == "" {
					return fmt.Errorf("theme-init: name is required")
				}
				parentName, _ := params["parent"].(string)
				return RunThemeInit(ctx, o, name, parentName)
			},
		},
		{
			Name:        "plugin-install",
			Kind:        operations.KindTask,
			Description: "Install a WASM plugin from a path or GitHub repo",
			Run: func(ctx context.Context, params map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				path, _ := params["path"].(string)
				if path == "" {
					return fmt.Errorf("plugin-install: path is required")
				}
				return RunPluginInstall(ctx, o, path, "")
			},
		},
		{
			Name:        "import-wordpress",
			Kind:        operations.KindTask,
			Description: "Import posts from a WordPress WXR export",
			Run: func(ctx context.Context, params map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				file, _ := params["file"].(string)
				if file == "" {
					return fmt.Errorf("import-wordpress: file is required")
				}
				return RunImportWordpress(ctx, o, file, parent.DryRun)
			},
		},
		{
			Name:        "import-hugo",
			Kind:        operations.KindTask,
			Description: "Import posts from a Hugo content directory",
			Run: func(ctx context.Context, params map[string]any, w io.Writer) error {
				o := parent
				o.LogWriter = w
				o.Progress = nil
				dir, _ := params["dir"].(string)
				if dir == "" {
					return fmt.Errorf("import-hugo: dir is required")
				}
				return RunImportHugo(ctx, o, dir, parent.DryRun)
			},
		},
	}
}

// newPostOptionsFromParams builds NewPostOptions from a Run's params.
// Title is required. Tags are split on comma; whitespace trimmed.
func newPostOptionsFromParams(params map[string]any) (NewPostOptions, error) {
	title, _ := params["title"].(string)
	if title == "" {
		return NewPostOptions{}, fmt.Errorf("new: title is required")
	}
	publish, _ := params["publish"].(bool)
	notesDir, _ := params["notes-dir"].(string)
	tagsRaw, _ := params["tags"].(string)
	var tags []string
	for _, t := range strings.Split(tagsRaw, ",") {
		t = strings.TrimSpace(t)
		if t != "" {
			tags = append(tags, t)
		}
	}
	return NewPostOptions{
		Title:    title,
		Tags:     tags,
		Publish:  publish,
		NotesDir: notesDir,
		Editor:   false, // never auto-open from the dashboard
	}, nil
}

// deployOptionsFromParams builds DeployOptions from a Run's params map,
// falling back to the project's deploy config defaults when keys are
// missing. Used by the deploy task closure so /actions can supply
// provider/preview/build overrides without changing the runner API.
func deployOptionsFromParams(params map[string]any, cfg config.Config) DeployOptions {
	opts := DeployOptions{
		Provider: cfg.Deploy.Provider,
		Build:    true,
	}
	if v, ok := params["provider"].(string); ok && v != "" {
		opts.Provider = v
	}
	if v, ok := params["build"].(bool); ok {
		opts.Build = v
	}
	if v, ok := params["preview"].(bool); ok {
		opts.DryRun = v
	}
	return opts
}

// checkOptionsFromParams builds CheckOptions from a Run's params map.
// When no flags are set, RunCheck itself defaults to running every
// check, so passing zero-value here is a sensible default.
func checkOptionsFromParams(params map[string]any) CheckOptions {
	opts := CheckOptions{}
	if v, ok := params["links"].(bool); ok {
		opts.Links = v
	}
	if v, ok := params["images"].(bool); ok {
		opts.Images = v
	}
	if v, ok := params["frontmatter"].(bool); ok {
		opts.Frontmatter = v
	}
	if v, ok := params["json"].(bool); ok {
		opts.JSON = v
	}
	return opts
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
