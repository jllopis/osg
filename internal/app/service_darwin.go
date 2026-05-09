//go:build darwin

package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

// darwinPlistPath returns the absolute path of the LaunchAgent plist:
// ~/Library/LaunchAgents/com.jllopis.<label>.plist.
func darwinPlistPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home dir: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents", "com.jllopis."+ServiceLabel+".plist"), nil
}

// darwinDomain returns the launchctl domain for the current GUI user
// session: "gui/<uid>". This is the modern replacement for the legacy
// `launchctl load` flow and lets us bootstrap/bootout cleanly.
func darwinDomain() string {
	return fmt.Sprintf("gui/%d", os.Getuid())
}

// darwinServiceTarget returns the full service target launchctl wants
// for kickstart/print/bootout: "gui/<uid>/com.jllopis.<label>".
func darwinServiceTarget() string {
	return fmt.Sprintf("gui/%d/com.jllopis.%s", os.Getuid(), ServiceLabel)
}

// RunServiceInstall writes the LaunchAgent plist and (unless NoStart
// is true) bootstraps it into the user's GUI session. Re-running is
// idempotent: the plist is overwritten and a previous bootstrap is
// booted out before re-bootstrapping with the new exec path.
func RunServiceInstall(ctx context.Context, opts CLIOptions, sopts ServiceInstallOptions) error {
	params, err := resolveServiceParams(opts, sopts)
	if err != nil {
		return err
	}
	if err := ensureLogsDir(params); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not create logs dir: %v\n", err)
	}
	plistPath, err := darwinPlistPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(plistPath), 0o755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}
	content := darwinPlistContent(params)
	if err := os.WriteFile(plistPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write plist: %w", err)
	}
	fmt.Printf("Wrote %s\n", plistPath)

	// Boot out a previous instance so the bootstrap below picks up
	// the new exec/workdir/config. Failure is expected (and ignored)
	// when no previous instance exists.
	_ = runLaunchctl(ctx, "bootout", darwinServiceTarget())

	if sopts.NoStart {
		fmt.Println("Plist installed. Run `osg service start` when you're ready to enable it.")
		return nil
	}
	if err := runLaunchctl(ctx, "bootstrap", darwinDomain(), plistPath); err != nil {
		return err
	}
	fmt.Printf("Service %s loaded and started. Use `osg service status` to check.\n", ServiceLabel)
	return nil
}

// RunServiceUninstall boots out the LaunchAgent and deletes its plist.
// Tolerates a missing plist or already-booted-out service.
func RunServiceUninstall(ctx context.Context, opts CLIOptions) error {
	_ = opts
	plistPath, err := darwinPlistPath()
	if err != nil {
		return err
	}
	_ = runLaunchctl(ctx, "bootout", darwinServiceTarget())
	if err := os.Remove(plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove plist: %w", err)
	}
	fmt.Printf("Service %s uninstalled.\n", ServiceLabel)
	return nil
}

// RunServiceStart kickstarts the agent. Required when KeepAlive
// missed a restart (rare) or when --no-start was used at install time.
func RunServiceStart(ctx context.Context, _ CLIOptions) error {
	plistPath, err := darwinPlistPath()
	if err != nil {
		return err
	}
	// Try kickstart first (faster path when already bootstrapped).
	// Fall back to a fresh bootstrap so a one-step `osg service start`
	// works even if the agent was never loaded.
	if err := runLaunchctl(ctx, "kickstart", darwinServiceTarget()); err == nil {
		return nil
	}
	return runLaunchctl(ctx, "bootstrap", darwinDomain(), plistPath)
}

// RunServiceStop boots out the agent. The plist stays on disk, so a
// later `osg service start` resurrects it without re-installing.
func RunServiceStop(ctx context.Context, _ CLIOptions) error {
	return runLaunchctl(ctx, "bootout", darwinServiceTarget())
}

// RunServiceStatus calls `launchctl print` on the service target and
// pipes the output verbatim. Exit codes from print are diagnostic
// (non-zero when the service isn't loaded), not OSG errors.
func RunServiceStatus(ctx context.Context, _ CLIOptions) error {
	cmd := exec.CommandContext(ctx, "launchctl", "print", darwinServiceTarget())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		// Surface the failure for the user to see, but keep the OSG
		// CLI exit clean — they'll read the launchctl output.
		fmt.Fprintf(os.Stderr, "(service likely not loaded; uid=%s)\n", strconv.Itoa(os.Getuid()))
	}
	return nil
}

// runLaunchctl invokes launchctl with the given args, piping
// stdout/stderr.
func runLaunchctl(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "launchctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("launchctl %v: %w", args, err)
	}
	return nil
}
