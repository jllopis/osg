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

	Init          struct{}  `cmd:"" help:"Initialize project structure"`
	UpdateContent struct{}  `cmd:"" help:"Sync content from vault"`
	Build         BuildCmd  `cmd:"" help:"Build static site (HTML)"`
	Serve         ServeCmd  `cmd:"" help:"Serve public directory"`
	New           NewCmd    `cmd:"" help:"Create a new post in the vault"`
	TUI           struct{}  `cmd:"" help:"Launch TUI"`
	Theme         ThemeCmd  `cmd:"" help:"Theme tools"`
	Plugin        PluginCmd `cmd:"" help:"Plugin tools"`
	Doctor        struct{}  `cmd:"" help:"Validate configuration and environment"`
	Version       struct{}  `cmd:"" help:"Show version information"`
}

type BuildCmd struct {
	ForceAISummaries bool `help:"Regenerate all AI summaries (bypasses cache)" name:"force-ai-summaries"`
	Yes              bool `help:"Skip confirmation prompts" short:"y"`
}

type ServeCmd struct {
	Addr       string `help:"Address to serve (host:port)" default:":1313"`
	Watch      *bool  `help:"Watch files and rebuild"`
	LiveReload *bool  `help:"Enable live reload (requires watch)"`
	DebounceMs *int   `help:"Debounce watch events in ms"`
}

type NewCmd struct {
	Title   string   `arg:"" help:"Post title"`
	Tags    []string `help:"Comma-separated tags" short:"t" sep:","`
	Publish bool     `help:"Mark as published (default: draft)"`
}

type ThemeCmd struct {
	Init ThemeInitCmd `cmd:"" help:"Create a new theme scaffold"`
}

type ThemeInitCmd struct {
	Name string `arg:"" help:"Theme name"`
}

type PluginCmd struct {
	Install PluginInstallCmd `cmd:"" help:"Install a WASM plugin"`
	Enable  PluginToggleCmd  `cmd:"" help:"Enable a plugin"`
	Disable PluginToggleCmd  `cmd:"" help:"Disable a plugin"`
	List    struct{}         `cmd:"" help:"List plugins"`
	Init    PluginInitCmd    `cmd:"" help:"Scaffold a plugin project"`
}

type PluginInstallCmd struct {
	Path string `arg:"" help:"Path to .wasm file"`
	Name string `help:"Optional plugin name (without extension)"`
}

type PluginToggleCmd struct {
	Name string `arg:"" help:"Plugin name"`
}

type PluginInitCmd struct {
	Name string `arg:"" help:"Plugin name"`
	Dir  string `help:"Base directory for plugin sources" default:"plugins_src"`
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
	if shouldDefaultToTUI(args) {
		args = append([]string{"tui"}, args...)
	}

	ctx, err := parser.Parse(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	opts := app.CLIOptions{
		ConfigPath:       cli.Config,
		Verbose:          cli.Verbose,
		DryRun:           cli.DryRun,
		IncludeDrafts:    cli.IncludeDrafts,
		VaultPath:        cli.VaultPath,
		OsgContentDir:    cli.OsgContentDir,
		PublicDir:        cli.PublicDir,
		ServeAddr:        cli.Serve.Addr,
		ServeWatch:       cli.Serve.Watch,
		ServeReload:      cli.Serve.LiveReload,
		ServeDebounce:    cli.Serve.DebounceMs,
		ForceAISummaries: cli.Build.ForceAISummaries,
	}

	var runErr error
	command := ctx.Command()
	switch {
	case command == "init":
		runErr = app.RunInit(context.Background(), opts)
	case command == "update-content":
		runErr = app.RunUpdateContent(context.Background(), opts)
	case command == "build":
		if cli.Build.ForceAISummaries && !cli.Build.Yes {
			fmt.Print("This will regenerate ALL AI summaries (may incur API costs). Continue? [y/N] ")
			var answer string
			fmt.Scanln(&answer)
			if strings.ToLower(strings.TrimSpace(answer)) != "y" {
				fmt.Fprintln(os.Stderr, "aborted")
				os.Exit(0)
			}
		}
		runErr = app.RunBuild(context.Background(), opts)
	case strings.HasPrefix(command, "new"):
		postOpts := app.NewPostOptions{
			Title:   cli.New.Title,
			Tags:    cli.New.Tags,
			Publish: cli.New.Publish,
		}
		runErr = app.RunNew(context.Background(), opts, postOpts)
	case command == "serve":
		runErr = app.RunServe(context.Background(), opts)
	case command == "tui":
		runErr = app.RunTUI(context.Background(), opts)
	case command == "doctor":
		runErr = app.RunDoctor(context.Background(), opts)
	case strings.HasPrefix(command, "theme init"):
		runErr = app.RunThemeInit(context.Background(), opts, cli.Theme.Init.Name)
	case strings.HasPrefix(command, "plugin install"):
		runErr = app.RunPluginInstall(context.Background(), opts, cli.Plugin.Install.Path, cli.Plugin.Install.Name)
	case strings.HasPrefix(command, "plugin enable"):
		runErr = app.RunPluginEnable(context.Background(), opts, cli.Plugin.Enable.Name)
	case strings.HasPrefix(command, "plugin disable"):
		runErr = app.RunPluginDisable(context.Background(), opts, cli.Plugin.Disable.Name)
	case command == "plugin list":
		runErr = app.RunPluginList(context.Background(), opts, os.Stdout)
	case strings.HasPrefix(command, "plugin init"):
		runErr = app.RunPluginInit(context.Background(), opts, cli.Plugin.Init.Name, cli.Plugin.Init.Dir)
	case command == "version":
		fmt.Println(app.VersionInfo())
	default:
		runErr = fmt.Errorf("unknown command: %s", command)
	}

	if runErr != nil {
		fmt.Fprintln(os.Stderr, runErr)
		os.Exit(1)
	}
}

func shouldDefaultToTUI(args []string) bool {
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
