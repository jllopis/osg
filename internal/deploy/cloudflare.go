package deploy

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func init() {
	Register("cloudflare", newCloudflare)
}

// CloudflareProvider deploys to Cloudflare Workers (Static Assets) using Wrangler.
// Requires wrangler CLI installed and CLOUDFLARE_API_TOKEN set.
//
// Workers Static Assets is the current Cloudflare recommendation for static
// sites (Cloudflare Pages has been merged into Workers). The provider generates
// a temporary wrangler.toml pointing at the public directory and runs
// "wrangler deploy".
type CloudflareProvider struct {
	// WorkerName is the Worker name in Cloudflare (required).
	WorkerName string
	// AccountID is the Cloudflare account ID (optional if wrangler is configured).
	AccountID string
	// CompatibilityDate for the Worker runtime (default: 2024-09-01).
	CompatibilityDate string
	// NotFoundHandling controls 404 behavior: "none", "single-page-application",
	// or "404-page" (default: "404-page").
	NotFoundHandling string
	// ExtraFlags passes additional flags to wrangler.
	ExtraFlags string
}

func newCloudflare(cfg map[string]any) Provider {
	return &CloudflareProvider{
		WorkerName:        getString(cfg, "worker_name", getString(cfg, "project", "")),
		AccountID:         getString(cfg, "account_id", ""),
		CompatibilityDate: getString(cfg, "compatibility_date", "2024-09-01"),
		NotFoundHandling:  getString(cfg, "not_found_handling", "404-page"),
		ExtraFlags:        getString(cfg, "extra_flags", ""),
	}
}

func (p *CloudflareProvider) Name() string { return "cloudflare" }

func (p *CloudflareProvider) Validate() error {
	if os.Getenv("CLOUDFLARE_API_TOKEN") == "" {
		return fmt.Errorf("cloudflare: CLOUDFLARE_API_TOKEN not set")
	}
	if p.WorkerName == "" {
		return fmt.Errorf("cloudflare: worker_name (or project) is required")
	}
	if _, err := exec.LookPath("wrangler"); err != nil {
		return fmt.Errorf("cloudflare: wrangler CLI not found (install with: npm install -g wrangler)")
	}
	return nil
}

func (p *CloudflareProvider) Deploy(ctx context.Context, publicDir string) error {
	absPublic, err := filepath.Abs(publicDir)
	if err != nil {
		return fmt.Errorf("cloudflare: resolving public dir: %w", err)
	}

	// Verify public dir exists and has content.
	entries, err := os.ReadDir(absPublic)
	if err != nil {
		return fmt.Errorf("cloudflare: reading public dir: %w", err)
	}
	if len(entries) == 0 {
		return fmt.Errorf("cloudflare: public dir %q is empty (run 'osg build' first)", absPublic)
	}

	// Generate a temporary wrangler.toml that points [assets] at the public dir.
	tmpDir, err := os.MkdirTemp("", "osg-cf-deploy-*")
	if err != nil {
		return fmt.Errorf("cloudflare: creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	toml := fmt.Sprintf(`name = %q
compatibility_date = %q

[vars]
ENVIRONMENT = "production"

[assets]
directory = %q
not_found_handling = %q
`, p.WorkerName, p.CompatibilityDate, absPublic, p.NotFoundHandling)

	tomlPath := filepath.Join(tmpDir, "wrangler.toml")
	if err := os.WriteFile(tomlPath, []byte(toml), 0644); err != nil {
		return fmt.Errorf("cloudflare: writing wrangler.toml: %w", err)
	}

	// Build wrangler args.
	args := []string{"deploy", "--config", tomlPath}
	if p.ExtraFlags != "" {
		args = append(args, strings.Fields(p.ExtraFlags)...)
	}

	// Build environment.
	env := os.Environ()
	if p.AccountID != "" {
		env = append(env, "CLOUDFLARE_ACCOUNT_ID="+p.AccountID)
	}

	logf(ctx, "Deploying to Cloudflare Workers: %s ...\n", p.WorkerName)
	cmd := exec.CommandContext(ctx, "wrangler", args...)
	wireCommandOutput(ctx, cmd)
	cmd.Env = env
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cloudflare: wrangler deploy failed: %w", err)
	}

	logf(ctx, "Deployed to Cloudflare Workers: %s\n", p.WorkerName)
	return nil
}
