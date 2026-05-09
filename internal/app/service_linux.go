//go:build linux

package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// linuxUnitPath returns the absolute path of the systemd user unit
// file. ~/.config/systemd/user/<label>.service.
func linuxUnitPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home dir: %w", err)
	}
	return filepath.Join(home, ".config", "systemd", "user", ServiceLabel+".service"), nil
}

// RunServiceInstall writes the systemd user unit file and (unless
// NoStart is true) reloads, enables and starts it. Re-running is
// idempotent: the unit file is overwritten and `systemctl --user
// enable --now` either starts a fresh service or restarts the
// existing one with the new exec path.
func RunServiceInstall(ctx context.Context, opts CLIOptions, sopts ServiceInstallOptions) error {
	params, err := resolveServiceParams(opts, sopts)
	if err != nil {
		return err
	}
	if err := ensureLogsDir(params); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not create logs dir: %v\n", err)
	}
	unitPath, err := linuxUnitPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(unitPath), 0o755); err != nil {
		return fmt.Errorf("create systemd user dir: %w", err)
	}
	content := linuxUnitContent(params)
	if err := os.WriteFile(unitPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write unit file: %w", err)
	}
	fmt.Printf("Wrote %s\n", unitPath)

	if err := runSystemctl(ctx, "daemon-reload"); err != nil {
		return err
	}
	if sopts.NoStart {
		fmt.Println("Unit installed. Run `osg service start` when you're ready to enable it.")
		return nil
	}
	if err := runSystemctl(ctx, "enable", "--now", ServiceLabel+".service"); err != nil {
		return err
	}
	fmt.Printf("Service %s enabled and started. Use `osg service status` to check.\n", ServiceLabel)
	return nil
}

// RunServiceUninstall stops the service, disables auto-start, and
// removes the unit file. Tolerates a missing unit (returns success
// after the daemon-reload).
func RunServiceUninstall(ctx context.Context, opts CLIOptions) error {
	_ = opts
	unitPath, err := linuxUnitPath()
	if err != nil {
		return err
	}
	// Best-effort stop+disable: if the unit doesn't exist these
	// commands fail with a recognisable error — we keep going.
	_ = runSystemctl(ctx, "disable", "--now", ServiceLabel+".service")
	if err := os.Remove(unitPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove unit file: %w", err)
	}
	if err := runSystemctl(ctx, "daemon-reload"); err != nil {
		return err
	}
	fmt.Printf("Service %s uninstalled.\n", ServiceLabel)
	return nil
}

// RunServiceStart, RunServiceStop and RunServiceStatus delegate to
// systemctl. They surface its output verbatim so the user gets
// systemd's own diagnostics.
func RunServiceStart(ctx context.Context, _ CLIOptions) error {
	return runSystemctl(ctx, "start", ServiceLabel+".service")
}

func RunServiceStop(ctx context.Context, _ CLIOptions) error {
	return runSystemctl(ctx, "stop", ServiceLabel+".service")
}

func RunServiceStatus(ctx context.Context, _ CLIOptions) error {
	// status returns non-zero when the service is inactive or failed;
	// that's diagnostic data, not an OSG error, so we discard the
	// exit code and let stdout/stderr inform the user.
	cmd := exec.CommandContext(ctx, "systemctl", "--user", "status", ServiceLabel+".service")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	_ = cmd.Run()
	return nil
}

// runSystemctl invokes systemctl --user with the given args, piping
// stdout/stderr so the user sees systemd's own messages.
func runSystemctl(ctx context.Context, args ...string) error {
	full := append([]string{"--user"}, args...)
	cmd := exec.CommandContext(ctx, "systemctl", full...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("systemctl %v: %w", args, err)
	}
	return nil
}
