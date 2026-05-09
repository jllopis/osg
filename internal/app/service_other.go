//go:build !linux && !darwin

package app

import (
	"context"
	"fmt"
	"runtime"
)

// On platforms without first-class support (Windows, FreeBSD, ...)
// the service subcommand returns a clear error rather than silently
// succeeding. Users on those systems can still run `osg ui` manually
// or set up their own equivalent via Task Scheduler / rc.d / etc.

func unsupportedService() error {
	return fmt.Errorf("osg service: %s is not a supported platform (use linux or darwin)", runtime.GOOS)
}

func RunServiceInstall(_ context.Context, _ CLIOptions, _ ServiceInstallOptions) error {
	return unsupportedService()
}

func RunServiceUninstall(_ context.Context, _ CLIOptions) error {
	return unsupportedService()
}

func RunServiceStart(_ context.Context, _ CLIOptions) error {
	return unsupportedService()
}

func RunServiceStop(_ context.Context, _ CLIOptions) error {
	return unsupportedService()
}

func RunServiceStatus(_ context.Context, _ CLIOptions) error {
	return unsupportedService()
}
