package app

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"osg/internal/api"
	"osg/internal/config"
	"osg/internal/logging"
)

// APIOptions holds settings for the standalone API server.
type APIOptions struct {
	Listen string
}

// RunAPI starts the standalone interactions API server.
func RunAPI(ctx context.Context, opts CLIOptions, apiOpts APIOptions) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
	}

	logger := logging.NewWithWriter(cfg.Logging, opts.Verbose, opts.LogWriter)

	listen := apiOpts.Listen
	if listen == "" {
		listen = cfg.Interactions.Listen
	}

	store, err := api.NewStore(cfg.Interactions.DBPath, cfg.Interactions.ViewDedupHours)
	if err != nil {
		return err
	}
	defer store.Close()

	srv := api.NewServer(store, cfg.Interactions, logger)

	server := &http.Server{
		Addr:    listen,
		Handler: srv.Handler(),
	}

	logger.Info("interactions API listening", "addr", listen, "db", cfg.Interactions.DBPath)

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

// StartAPIHandler creates an API handler that can be mounted on an existing
// ServeMux (used by `osg serve --api`).
func StartAPIHandler(cfg config.InteractionsConfig, logger *slog.Logger) (*api.Server, *api.Store, error) {
	store, err := api.NewStore(cfg.DBPath, cfg.ViewDedupHours)
	if err != nil {
		return nil, nil, err
	}
	srv := api.NewServer(store, cfg, logger)
	return srv, store, nil
}
