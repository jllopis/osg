package app

import (
	"context"

	"osg/internal/build"
	"osg/internal/config"
)

func RunBuild(ctx context.Context, opts CLIOptions) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return err
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

	buildOpts := build.BuildOptions{
		SkipAI:           opts.SkipAI,
		ForceAISummaries: opts.ForceAISummaries,
	}

	return build.Run(ctx, cfg, buildOpts, opts.Verbose, opts.LogWriter)
}
