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

// CloudflareProvider deploys to Cloudflare Pages or Workers using Wrangler.
// Requires wrangler CLI installed and CLOUDFLARE_API_TOKEN set.
type CloudflareProvider struct {
	// Project is the Cloudflare Pages project name.
	Project string
	// Branch is the git branch for Pages (default: main).
	Branch string
	// AccountID is the Cloudflare account ID (optional if wrangler is configured).
	AccountID string
	// UseWorkers deploys as a Worker instead of Pages.
	UseWorkers bool
	// WorkerName is the Worker name (required if UseWorkers is true).
	WorkerName string
	// ExtraFlags passes additional flags to wrangler.
	ExtraFlags string
}

func newCloudflare(cfg map[string]any) Provider {
	return &CloudflareProvider{
		Project:    getString(cfg, "project", ""),
		Branch:     getString(cfg, "branch", "main"),
		AccountID:  getString(cfg, "account_id", ""),
		UseWorkers: getBool(cfg, "workers", false),
		WorkerName: getString(cfg, "worker_name", ""),
		ExtraFlags: getString(cfg, "extra_flags", ""),
	}
}

func (p *CloudflareProvider) Name() string { return "cloudflare" }

func (p *CloudflareProvider) Validate() error {
	// Check for API token
	if os.Getenv("CLOUDFLARE_API_TOKEN") == "" {
		return fmt.Errorf("cloudflare: CLOUDFLARE_API_TOKEN not set")
	}

	if p.UseWorkers {
		if p.WorkerName == "" {
			return fmt.Errorf("cloudflare: worker_name is required when workers=true")
		}
	} else {
		if p.Project == "" {
			return fmt.Errorf("cloudflare: project is required (Cloudflare Pages project name)")
		}
	}

	// Check wrangler is installed
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

	env := os.Environ()
	if p.AccountID != "" {
		env = append(env, "CLOUDFLARE_ACCOUNT_ID="+p.AccountID)
	}

	if p.UseWorkers {
		return p.deployWorkers(ctx, absPublic, env)
	}
	return p.deployPages(ctx, absPublic, env)
}

func (p *CloudflareProvider) deployPages(ctx context.Context, publicDir string, env []string) error {
	args := []string{"pages", "deploy", publicDir, "--project-name", p.Project, "--branch", p.Branch}

	if p.ExtraFlags != "" {
		args = append(args, strings.Fields(p.ExtraFlags)...)
	}

	fmt.Printf("Deploying to Cloudflare Pages: %s (branch: %s) ...\n", p.Project, p.Branch)
	cmd := exec.CommandContext(ctx, "wrangler", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cloudflare pages: %w", err)
	}

	fmt.Printf("Deployed to Cloudflare Pages: https://%s.pages.dev\n", p.Project)
	return nil
}

func (p *CloudflareProvider) deployWorkers(ctx context.Context, publicDir string, env []string) error {
	// Workers require a wrangler.toml or we pass all config via flags
	// For static sites, we deploy as a Worker with static assets
	args := []string{
		"deploy",
		"--name", p.WorkerName,
		"--compatibility-date", "2024-01-01",
	}

	if p.AccountID != "" {
		args = append(args, "--account-id", p.AccountID)
	}

	// For Workers serving static files, we need to create a minimal worker script
	// that serves from the public directory. This is more complex than Pages.
	// For simplicity, we'll use wrangler's static asset serving feature.

	// Create a minimal worker script
	workerScript := fmt.Sprintf(`
export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    const path = url.pathname;
    
    // Try to serve static file
    const key = path === '/' ? 'index.html' : path.slice(1);
    const object = await env.BUCKET.get(key);
    
    if (object) {
      const headers = new Headers();
      object.writeHttpMetadata(headers);
      headers.set('etag', object.httpEtag);
      return new Response(object.body, { headers });
    }
    
    // Try with .html extension for SPA-style routing
    const htmlKey = key + '.html';
    const htmlObject = await env.BUCKET.get(htmlKey);
    if (htmlObject) {
      const headers = new Headers();
      htmlObject.writeHttpMetadata(headers);
      return new Response(htmlObject.body, { headers });
    }
    
    return new Response('Not Found', { status: 404 });
  }
};
`)

	// Write temporary worker script
	tmpDir, err := os.MkdirTemp("", "osg-cloudflare-worker-*")
	if err != nil {
		return fmt.Errorf("cloudflare workers: creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	workerFile := filepath.Join(tmpDir, "worker.js")
	if err := os.WriteFile(workerFile, []byte(workerScript), 0644); err != nil {
		return fmt.Errorf("cloudflare workers: writing worker script: %w", err)
	}

	args = append(args, workerFile)

	if p.ExtraFlags != "" {
		args = append(args, strings.Fields(p.ExtraFlags)...)
	}

	fmt.Printf("Deploying to Cloudflare Workers: %s ...\n", p.WorkerName)
	fmt.Println("Note: For static sites, Cloudflare Pages is recommended over Workers.")
	cmd := exec.CommandContext(ctx, "wrangler", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("cloudflare workers: %w", err)
	}

	fmt.Printf("Deployed to Cloudflare Workers: %s\n", p.WorkerName)
	fmt.Println("Check the wrangler output above for the full deployment URL.")
	return nil
}
