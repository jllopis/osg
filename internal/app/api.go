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

	// Optionally create comment store and auth providers.
	var commentStore *api.CommentStore
	var authProviders map[string]*api.AuthProvider
	if cfg.Interactions.Comments.Enabled && len(cfg.Interactions.Comments.Providers) > 0 {
		cs, err := api.NewCommentStore(cfg.Interactions.Comments.DBPath, cfg.Interactions.Comments.AuthSessionDays)
		if err != nil {
			return err
		}
		defer cs.Close()
		commentStore = cs

		baseCallback := cfg.Interactions.Comments.AuthCallbackURL
		authProviders = api.BuildAuthProviders(cfg.Interactions.Comments, baseCallback)

		logger.Info("comments enabled", "db", cfg.Interactions.Comments.DBPath, "providers", len(authProviders))
	}

	srv := api.NewServer(store, cfg.Interactions, logger, commentStore, authProviders)

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
func StartAPIHandler(cfg config.InteractionsConfig, logger *slog.Logger) (*api.Server, *api.Store, *api.CommentStore, error) {
	store, err := api.NewStore(cfg.DBPath, cfg.ViewDedupHours)
	if err != nil {
		return nil, nil, nil, err
	}

	var commentStore *api.CommentStore
	var authProviders map[string]*api.AuthProvider
	if cfg.Comments.Enabled && len(cfg.Comments.Providers) > 0 {
		cs, err := api.NewCommentStore(cfg.Comments.DBPath, cfg.Comments.AuthSessionDays)
		if err != nil {
			store.Close()
			return nil, nil, nil, err
		}
		commentStore = cs

		baseCallback := cfg.Comments.AuthCallbackURL
		authProviders = api.BuildAuthProviders(cfg.Comments, baseCallback)
	}

	srv := api.NewServer(store, cfg, logger, commentStore, authProviders)
	return srv, store, commentStore, nil
}
