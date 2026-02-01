package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/alecthomas/kong"

	"osg/internal/app"
)

type CLI struct {
	Config        string `help:"Path to config file" short:"c" default:"config.yaml"`
	Verbose       bool   `help:"Enable verbose logging"`
	DryRun        bool   `help:"Do not write files"`
	IncludeDrafts *bool  `help:"Include drafts (publish: \"draft\")"`
	VaultPath     string `help:"Path to Obsidian vault"`
	OsgContentDir string `help:"Content directory override"`
	PublicDir     string `help:"Public directory override"`

	Init          struct{} `cmd:"" help:"Initialize project structure"`
	UpdateContent struct{} `cmd:"" help:"Sync content from vault"`
	Build         struct{} `cmd:"" help:"Build static site (HTML)"`
	Serve         ServeCmd `cmd:"" help:"Serve public directory"`
	TUI           struct{} `cmd:"" help:"Launch TUI"`
	Version       struct{} `cmd:"" help:"Show version information"`
}

type ServeCmd struct {
	Addr       string `help:"Address to serve (host:port)" default:":1313"`
	Watch      *bool  `help:"Watch files and rebuild"`
	LiveReload *bool  `help:"Enable live reload (requires watch)"`
	DebounceMs *int   `help:"Debounce watch events in ms"`
}

func main() {
	cli := CLI{}

	parser, err := kong.New(&cli,
		kong.Name("osg"),
		kong.Description("OSG - Obsidian Site Generator"),
	)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	args := os.Args[1:]
	if shouldDefaultToUpdate(args) {
		args = append([]string{"update-content"}, args...)
	}

	ctx, err := parser.Parse(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	opts := app.CLIOptions{
		ConfigPath:    cli.Config,
		Verbose:       cli.Verbose,
		DryRun:        cli.DryRun,
		IncludeDrafts: cli.IncludeDrafts,
		VaultPath:     cli.VaultPath,
		OsgContentDir: cli.OsgContentDir,
		PublicDir:     cli.PublicDir,
		ServeAddr:     cli.Serve.Addr,
		ServeWatch:    cli.Serve.Watch,
		ServeReload:   cli.Serve.LiveReload,
		ServeDebounce: cli.Serve.DebounceMs,
	}

	var runErr error
	switch ctx.Command() {
	case "init":
		runErr = app.RunInit(context.Background(), opts)
	case "update-content":
		runErr = app.RunUpdateContent(context.Background(), opts)
	case "build":
		runErr = app.RunBuild(context.Background(), opts)
	case "serve":
		runErr = app.RunServe(context.Background(), opts)
	case "tui":
		runErr = app.RunTUI(context.Background(), opts)
	case "version":
		printVersion()
	default:
		runErr = fmt.Errorf("unknown command: %s", ctx.Command())
	}

	if runErr != nil {
		fmt.Fprintln(os.Stderr, runErr)
		os.Exit(1)
	}
}

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func printVersion() {
	fmt.Printf("version: %s\ncommit:  %s\ndate:    %s\n", version, commit, date)
}

func shouldDefaultToUpdate(args []string) bool {
	if hasHelpFlag(args) {
		return false
	}

	for _, arg := range args {
		if strings.TrimSpace(arg) == "" {
			continue
		}
		if strings.HasPrefix(arg, "-") {
			continue
		}
		return false
	}

	return true
}

func hasHelpFlag(args []string) bool {
	for _, arg := range args {
		if arg == "-h" || arg == "--help" || arg == "help" {
			return true
		}
	}
	return false
}
