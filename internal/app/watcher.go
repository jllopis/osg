package app

import (
	"context"
	"time"

	"osg/internal/config"
	"osg/internal/logging"
)

// RunWatcher watches the vault and content/static/templates roots, and
// rebuilds (and runs update-content when the vault is configured) on
// debounced filesystem events. The function blocks until ctx is cancelled.
//
// It reuses the same startWatch/runWatchLoop code path as `osg serve --watch`
// but without a reload hub: this is the right primitive for the dashboard
// supervisor, which only cares about driving rebuilds on disk changes —
// browser live-reload remains the responsibility of `osg serve`.
func RunWatcher(ctx context.Context, opts CLIOptions) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	if opts.VaultPath != "" {
		cfg.VaultPath = opts.VaultPath
	}
	if opts.OsgContentDir != "" {
		cfg.ContentDir = opts.OsgContentDir
	}
	if opts.PublicDir != "" {
		cfg.PublicDir = opts.PublicDir
	}
	if opts.IncludeDrafts != nil {
		cfg.IncludeDrafts = *opts.IncludeDrafts
	}

	debounce := cfg.ServeDebounce
	if opts.ServeDebounce != nil {
		debounce = *opts.ServeDebounce
	}
	if debounce <= 0 {
		debounce = 300
	}

	// Skip AI summaries on rebuild — same rationale as `osg serve`.
	opts.SkipAI = true

	logger := logging.NewWithWriter(cfg.Logging, opts.Verbose, opts.LogWriter)
	logger.Info("watcher starting",
		"vault", cfg.VaultPath,
		"content", cfg.ContentDir,
		"debounce_ms", debounce,
	)

	events, errs, err := startWatch(ctx, cfg, opts.ConfigPath, logger)
	if err != nil {
		return err
	}
	if events == nil {
		// startWatch returns nil channels when no roots are configured.
		// Block until ctx is cancelled so the supervisor reports running
		// state rather than immediate idle.
		<-ctx.Done()
		return nil
	}

	// Run an initial build so the public/ directory reflects current
	// content as soon as the watcher starts. Mirrors `osg serve --watch`.
	if err := runInitialBuild(ctx, opts, logger, cfg.VaultPath != ""); err != nil {
		logger.Warn("initial build failed", "error", err)
	}

	runWatchLoop(ctx, events, errs, opts, logger, nil, time.Duration(debounce)*time.Millisecond)
	return nil
}
