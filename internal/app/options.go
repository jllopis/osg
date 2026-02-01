package app

import "io"

type CLIOptions struct {
	ConfigPath    string
	Verbose       bool
	DryRun        bool
	IncludeDrafts *bool
	VaultPath     string
	OsgContentDir string
	PublicDir     string
	ServeAddr     string
	TUI           bool
	LogWriter     io.Writer
}
