package app

import (
	"context"
	"errors"
	"net/http"

	"osg/internal/config"
	"osg/internal/logging"
)

func RunServe(_ context.Context, opts CLIOptions) error {
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

	logger := logging.New(cfg.Logging, opts.Verbose)
	logger.Info("serving public", "dir", cfg.PublicDir, "addr", addr)

	server := &http.Server{
		Addr:    addr,
		Handler: http.FileServer(http.Dir(cfg.PublicDir)),
	}

	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}
