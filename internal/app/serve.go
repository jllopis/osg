package app

import (
	"context"
	"errors"
	"net/http"
	"time"

	"osg/internal/config"
	"osg/internal/logging"
)

func RunServe(ctx context.Context, opts CLIOptions) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	// During serve, skip AI summary generation to avoid costly API calls
	// on every rebuild.  Pages without summaries get the "auto" fallback.
	opts.SkipAI = true

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

	watch := cfg.ServeWatch
	if opts.ServeWatch != nil {
		watch = *opts.ServeWatch
	}
	liveReload := cfg.ServeReload
	if opts.ServeReload != nil {
		liveReload = *opts.ServeReload
	}
	debounceMs := cfg.ServeDebounce
	if opts.ServeDebounce != nil {
		debounceMs = *opts.ServeDebounce
	}
	if debounceMs <= 0 {
		debounceMs = 300
	}

	addr := opts.ServeAddr
	if addr == "" {
		addr = ":1313"
	}

	logger := logging.NewWithWriter(cfg.Logging, opts.Verbose, opts.LogWriter)
	logger.Info("serving public", "dir", cfg.PublicDir, "addr", addr)

	var hub *reloadHub
	if watch && liveReload {
		hub = newReloadHub()
	}

	handler := http.FileServer(http.Dir(cfg.PublicDir))
	if hub != nil {
		handler = newLiveReloadHandler(cfg.PublicDir, handler, hub)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	if watch {
		logger.Info("watch enabled", "debounce_ms", debounceMs, "live_reload", liveReload)
		if err := runInitialBuild(ctx, opts, logger, cfg.VaultPath != ""); err != nil {
			logger.Warn("initial build failed", "error", err)
		} else if hub != nil {
			hub.Broadcast()
		}
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	if watch {
		watchCtx, cancel := context.WithCancel(ctx)
		defer cancel()
		events, errs, err := startWatch(watchCtx, cfg, opts.ConfigPath, logger)
		if err != nil {
			logger.Warn("watcher failed", "error", err)
		} else if events != nil {
			go runWatchLoop(watchCtx, events, errs, opts, logger, hub, time.Duration(debounceMs)*time.Millisecond)
		}
	}

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}

	return nil
}
