package app

import (
	"context"
	"fmt"
	"log/slog"

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

	// Build first if requested
	if deployOpts.Build {
		fmt.Println("Building site...")
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
		fmt.Printf("Dry run: would deploy to %s using %s provider\n", cfg.PublicDir, provider)
		p, err := deploy.Get(provider, providerCfg)
		if err != nil {
			return err
		}
		if err := p.Validate(); err != nil {
			return err
		}
		fmt.Printf("Provider configuration validated successfully.\n")
		fmt.Printf("Public directory: %s\n", cfg.PublicDir)
		return nil
	}

	fmt.Printf("Deploying to %s...\n", provider)
	if err := deploy.Run(ctx, provider, providerCfg, cfg.PublicDir); err != nil {
		return err
	}

	webhook.Dispatch(ctx, cfg, "deploy.success", map[string]any{
		"provider": provider,
	}, slog.Default())

	return nil
}
