package app

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"osg/internal/config"
	"osg/internal/deploy"
	"osg/internal/webhook"
)

// DeployOptions holds CLI flags for the deploy command.
type DeployOptions struct {
	// Provider overrides the configured deploy provider.
	Provider string
	// Build runs osg build before deploying.
	Build bool
	// DryRun shows what would be deployed without making changes.
	DryRun bool
}

// RunDeploy deploys the site to the configured destination.
func RunDeploy(ctx context.Context, opts CLIOptions, deployOpts DeployOptions) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("deploy: loading config: %w", err)
	}

	// Resolve the writer used for status lines and for the deploy
	// subprocess output (wrangler/rsync). When opts.LogWriter is set
	// (osg ui supplies the runner's MultiWriter), everything flows
	// through it so the flow drawer sees the same lines as the
	// terminal. Otherwise we keep the CLI behaviour: status lines on
	// stdout, subprocess output on stdout/stderr via runCommand.
	out := io.Writer(os.Stdout)
	if opts.LogWriter != nil {
		out = opts.LogWriter
		ctx = deploy.WithLogWriter(ctx, opts.LogWriter)
	}

	// Build first if requested
	if deployOpts.Build {
		_, _ = fmt.Fprintln(out, "Building site...")
		if err := RunBuild(ctx, opts); err != nil {
			return fmt.Errorf("deploy: build failed: %w", err)
		}
	}

	// Determine provider
	provider := deployOpts.Provider
	if provider == "" {
		provider = cfg.Deploy.Provider
	}
	if provider == "" {
		return fmt.Errorf("deploy: no provider configured (set deploy.provider in config.yaml or use --provider)")
	}

	// Get provider-specific config
	var providerCfg map[string]any
	switch provider {
	case "cloudflare":
		providerCfg = cfg.Deploy.Cloudflare
	case "rsync":
		providerCfg = cfg.Deploy.Rsync
	case "s3":
		providerCfg = cfg.Deploy.S3
	default:
		providerCfg = make(map[string]any)
	}

	if deployOpts.DryRun {
		_, _ = fmt.Fprintf(out, "Dry run: would deploy to %s using %s provider\n", cfg.PublicDir, provider)
		p, err := deploy.Get(provider, providerCfg)
		if err != nil {
			return err
		}
		if err := p.Validate(); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "Provider configuration validated successfully.\n")
		_, _ = fmt.Fprintf(out, "Public directory: %s\n", cfg.PublicDir)
		return nil
	}

	_, _ = fmt.Fprintf(out, "Deploying to %s...\n", provider)
	if err := deploy.Run(ctx, provider, providerCfg, cfg.PublicDir); err != nil {
		return err
	}

	webhook.Dispatch(ctx, cfg, "deploy.success", map[string]any{
		"provider": provider,
	}, slog.Default())

	return nil
}
