package app

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"osg/internal/audit"
	"osg/internal/config"
)

// RunAudit audits the generated site in public/ for quality issues.
func RunAudit(_ context.Context, opts CLIOptions, jsonOutput bool) error {
	cfg, err := config.Load(opts.ConfigPath)
	if err != nil {
		return fmt.Errorf("audit: loading config: %w", err)
	}

	publicDir := cfg.PublicDir
	if opts.PublicDir != "" {
		publicDir = opts.PublicDir
	}

	if _, err := os.Stat(publicDir); os.IsNotExist(err) {
		return fmt.Errorf("audit: public directory %q does not exist (run osg build first)", publicDir)
	}

	report, err := audit.Run(publicDir)
	if err != nil {
		return fmt.Errorf("audit: %w", err)
	}

	if jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(report)
	}

	fmt.Print(audit.FormatReport(report))

	if report.ErrorCount() > 0 {
		return fmt.Errorf("%d errors found", report.ErrorCount())
	}
	return nil
}
