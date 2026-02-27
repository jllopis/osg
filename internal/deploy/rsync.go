package deploy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func init() {
	Register("rsync", newRsync)
}

// RsyncProvider deploys via rsync over SSH.
type RsyncProvider struct {
	Host       string // user@host or host (SSH destination)
	Path       string // Remote path (e.g. /var/www/blog)
	Port       string // SSH port (default 22)
	KeyFile    string // SSH private key path
	Delete     bool   // --delete flag (remove remote files not in local)
	Exclude    string // --exclude patterns (comma-separated)
	ExtraFlags string // Additional rsync flags
}

func newRsync(cfg map[string]any) Provider {
	return &RsyncProvider{
		Host:       getString(cfg, "host", ""),
		Path:       getString(cfg, "path", ""),
		Port:       getString(cfg, "port", "22"),
		KeyFile:    getString(cfg, "key_file", ""),
		Delete:     getBool(cfg, "delete", true),
		Exclude:    getString(cfg, "exclude", ""),
		ExtraFlags: getString(cfg, "extra_flags", ""),
	}
}

func (p *RsyncProvider) Name() string { return "rsync" }

func (p *RsyncProvider) Validate() error {
	if p.Host == "" {
		return fmt.Errorf("rsync: host is required (e.g. user@example.com)")
	}
	if p.Path == "" {
		return fmt.Errorf("rsync: path is required (e.g. /var/www/blog)")
	}
	if p.KeyFile != "" {
		keyPath := expandPath(p.KeyFile)
		if _, err := os.Stat(keyPath); err != nil {
			return fmt.Errorf("rsync: cannot read key_file %s: %w", keyPath, err)
		}
	}
	return nil
}

func (p *RsyncProvider) Deploy(ctx context.Context, publicDir string) error {
	absPublic, err := filepath.Abs(publicDir)
	if err != nil {
		return fmt.Errorf("rsync: resolving public dir: %w", err)
	}

	args := []string{
		"-avz",              // archive, verbose, compress
		"--checksum",        // use checksums for file comparison
		"--no-perms",        // don't preserve permissions (web servers vary)
		"--no-owner",        // don't preserve owner
		"--no-group",        // don't preserve group
		"--chmod=D755,F644", // set sensible web permissions
	}

	if p.Delete {
		args = append(args, "--delete")
	}
	if p.Delete {
		args = append(args, "--delete-after")
	}

	if p.Exclude != "" {
		for _, pattern := range strings.Split(p.Exclude, ",") {
			args = append(args, "--exclude", strings.TrimSpace(pattern))
		}
	}

	// SSH options
	sshOpts := []string{"ssh"}
	if p.Port != "22" {
		sshOpts = append(sshOpts, "-p", p.Port)
	}
	if p.KeyFile != "" {
		sshOpts = append(sshOpts, "-i", expandPath(p.KeyFile))
	}
	sshOpts = append(sshOpts, "-o", "StrictHostKeyChecking=accept-new")
	args = append(args, "-e", strings.Join(sshOpts, " "))

	if p.ExtraFlags != "" {
		args = append(args, strings.Fields(p.ExtraFlags)...)
	}

	// Source and destination
	args = append(args, absPublic+"/", p.Host+":"+strings.TrimSuffix(p.Path, "/")+"/")

	fmt.Printf("Deploying via rsync to %s:%s ...\n", p.Host, p.Path)
	if err := runCommand(ctx, "rsync", args...); err != nil {
		return fmt.Errorf("rsync: %w", err)
	}
	fmt.Printf("Deployed to %s:%s\n", p.Host, p.Path)
	return nil
}
