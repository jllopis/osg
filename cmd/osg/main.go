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
	Config            string `help:"Path to config file" short:"c" default:"config.yaml"`
	Verbose           bool   `help:"Enable verbose logging"`
	DryRun            bool   `help:"Do not write files"`
	IncludeDrafts     *bool  `help:"Include drafts (publish: \"draft\")"`
	VaultPath         string `help:"Path to Obsidian vault"`
	ObsidianVaultBase string `help:"Base path for Obsidian vaults"`
	Vault             string `help:"Vault name (used with --obsidian-vault-base)"`
	OsgContentDir     string `help:"Content directory override"`

	Init          struct{} `cmd:"" help:"Initialize project structure"`
	UpdateContent struct{} `cmd:"" help:"Sync content from vault"`
	Build         struct{} `cmd:"" help:"Build static site (HTML)"`
	Version       struct{} `cmd:"" help:"Show version information"`
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
		ConfigPath:        cli.Config,
		Verbose:           cli.Verbose,
		DryRun:            cli.DryRun,
		IncludeDrafts:     cli.IncludeDrafts,
		VaultPath:         cli.VaultPath,
		ObsidianVaultBase: cli.ObsidianVaultBase,
		Vault:             cli.Vault,
		OsgContentDir:     cli.OsgContentDir,
	}

	var runErr error
	switch ctx.Command() {
	case "init":
		runErr = app.RunInit(context.Background(), opts)
	case "update-content":
		runErr = app.RunUpdateContent(context.Background(), opts)
	case "build":
		runErr = app.RunBuild(context.Background(), opts)
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
