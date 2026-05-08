package app

import (
	"io"

	"osg/internal/logging"
)

type CLIOptions struct {
	ConfigPath       string
	Verbose          bool
	DryRun           bool
	IncludeDrafts    *bool
	VaultPath        string
	OsgContentDir    string
	PublicDir        string
	ServeAddr        string
	ServeWatch       *bool
	ServeReload      *bool
	ServeDebounce    *int
	ServeAPI         bool
	TUI              bool
	UIAddr           string
	LogWriter        io.Writer
	ForceAISummaries bool
	SkipAI           bool
	Progress         logging.Progress
	Profile          string
	TimingJSON       string
}
