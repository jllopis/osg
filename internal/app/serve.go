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

	if opts.PublicDir != "" {
		cfg.PublicDir = opts.PublicDir
	}

	addr := opts.ServeAddr
	if addr == "" {
		addr = ":1313"
	}

	logger := logging.NewWithWriter(cfg.Logging, opts.Verbose, opts.LogWriter)
	logger.Info("serving public", "dir", cfg.PublicDir, "addr", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: http.FileServer(http.Dir(cfg.PublicDir)),
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

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
