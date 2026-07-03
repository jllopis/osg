package deploy

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func init() {
	Register("s3", newS3)
}

// S3Provider deploys to AWS S3 or S3-compatible storage (Cloudflare R2, Backblaze B2).
// Uses the AWS CLI (aws s3 sync) which must be installed and configured.
type S3Provider struct {
	Bucket     string // S3 bucket name (e.g. my-bucket)
	Region     string // AWS region (e.g. eu-west-1)
	Endpoint   string // Custom endpoint for R2/B2 (optional)
	Path       string // Bucket path prefix (optional, e.g. blog/)
	Delete     bool   // Remove remote files not in local
	Profile    string // AWS profile to use (optional)
	ACL        string // Object ACL (default: public-read)
	ExtraFlags string // Additional aws s3 sync flags
}

func newS3(cfg map[string]any) Provider {
	return &S3Provider{
		Bucket:     getString(cfg, "bucket", ""),
		Region:     getString(cfg, "region", getEnvOr("AWS_DEFAULT_REGION", "")),
		Endpoint:   getString(cfg, "endpoint", ""),
		Path:       strings.Trim(getString(cfg, "path", ""), "/"),
		Delete:     getBool(cfg, "delete", false),
		Profile:    getString(cfg, "profile", getEnvOr("AWS_PROFILE", "")),
		ACL:        getString(cfg, "acl", "public-read"),
		ExtraFlags: getString(cfg, "extra_flags", ""),
	}
}

func (p *S3Provider) Name() string { return "s3" }

func (p *S3Provider) Validate() error {
	if p.Bucket == "" {
		return fmt.Errorf("s3: bucket is required")
	}
	// Check for AWS credentials - either via env vars or profile
	hasCreds := os.Getenv("AWS_ACCESS_KEY_ID") != "" ||
		os.Getenv("AWS_SECRET_ACCESS_KEY") != "" ||
		os.Getenv("AWS_PROFILE") != "" ||
		p.Profile != ""
	if !hasCreds {
		// Could also check for ~/.aws/credentials file, but keep it simple
		return fmt.Errorf("s3: no AWS credentials found (set AWS_ACCESS_KEY_ID/AWS_SECRET_ACCESS_KEY or AWS_PROFILE)")
	}
	return nil
}

func (p *S3Provider) Deploy(ctx context.Context, publicDir string) error {
	absPublic, err := filepath.Abs(publicDir)
	if err != nil {
		return fmt.Errorf("s3: resolving public dir: %w", err)
	}

	// Build s3:// URL
	s3URL := "s3://" + p.Bucket
	if p.Path != "" {
		s3URL += "/" + p.Path
	}
	s3URL += "/"

	args := []string{"s3", "sync", absPublic + "/", s3URL}

	// Let the AWS CLI guess Content-Type from file extensions (its default).
	// We previously passed --no-guess-mime-type with an s3cmd-style
	// --content-type-map, which the AWS CLI rejects as an unknown option.

	// ACL
	if p.ACL != "" {
		args = append(args, "--acl", p.ACL)
	}

	// Delete remote files not in local
	if p.Delete {
		args = append(args, "--delete")
	}

	// Custom endpoint (for R2, B2, MinIO)
	if p.Endpoint != "" {
		args = append(args, "--endpoint-url", p.Endpoint)
	}

	// Extra flags
	if p.ExtraFlags != "" {
		args = append(args, strings.Fields(p.ExtraFlags)...)
	}

	// Set up environment with region/profile
	env := os.Environ()
	if p.Region != "" {
		env = append(env, "AWS_DEFAULT_REGION="+p.Region)
	}
	if p.Profile != "" {
		env = append(env, "AWS_PROFILE="+p.Profile)
	}

	destination := p.Bucket
	if p.Endpoint != "" {
		if u, err := url.Parse(p.Endpoint); err == nil {
			destination = u.Host + "/" + p.Bucket
		}
	}

	logf(ctx, "Deploying to S3: %s ...\n", destination)
	cmd := exec.CommandContext(ctx, "aws", args...)
	wireCommandOutput(ctx, cmd)
	cmd.Env = env
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("s3: %w", err)
	}

	// Invalidate CloudFront if distribution ID is set
	if cfDist := os.Getenv("CLOUDFRONT_DISTRIBUTION_ID"); cfDist != "" {
		logf(ctx, "Invalidating CloudFront distribution %s ...\n", cfDist)
		cfCmd := exec.CommandContext(ctx, "aws", "cloudfront", "create-invalidation",
			"--distribution-id", cfDist,
			"--paths", "/*")
		wireCommandOutput(ctx, cfCmd)
		cfCmd.Env = env
		if err := cfCmd.Run(); err != nil {
			logf(ctx, "Warning: CloudFront invalidation failed: %v\n", err)
		}
	}

	logf(ctx, "Deployed to s3://%s\n", p.Bucket)
	return nil
}
