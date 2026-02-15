# Plan: TUI Redesign

> Rewrite the TUI from a 3-panel monolithic layout to a professional 2-panel
> design inspired by OpenCode, with scrollable output, slash commands, Nord
> palette, and modular code architecture.

## Problems with current TUI

1. **God file** — 1849 lines in `tui.go` with model, update, view, styles, commands, helpers all mixed.
2. **Wasted space** — 3 fixed panels: left (static shortcut list), center (duplicates info from both sides), right (static config).
3. **No scrolling** — output truncated to 8 lines, no viewport, no history.
4. **Fake progress** — oscillates 0-100% on a 10s timer, no real information.
5. **Wizard is a no-op** — toggle stored and displayed but zero behavioral effect.
6. **Duplicate info** — serve status in 3 places, workflow steps in 2 places.
7. **Non-Nord colors** — hardcoded `#8ECDF2`, `#F6A000`, etc. instead of Nord palette.
8. **Hardcoded version** — `v0.1.0` constant instead of `app.Version`.
9. **Dead code** — `formatMessage`, `serveStatus`, `minInt`/`maxInt`, duplicate `pathExists`/`normalizePluginName`.
10. **Zero tests**.

## New design

### Layout

```
+----------------------------------------------------------------------+
| OSG  site_title                  RUNNING :1313  |  build: 42 pages   |
+--------------------------------------+-------------------------------+
|                                      |                               |
|  Output / Activity Log               |  Sidebar                     |
|  (scrollable viewport)               |                               |
|                                      |  > Project                   |
|  [12:01:05] build started            |   vault: ../sample-vault     |
|  [12:01:05] parsed 5 pages           |   pages: 5                   |
|  [12:01:05] generated 4 placeholders |   scheme: auto               |
|  [12:01:05] rendered 42 files        |                               |
|  [12:01:05] build complete (0.3s)    |  > Workflow                  |
|  [12:01:10] doctor: 0 errors         |   [x] Init                   |
|                                      |   [x] Update                 |
|                                      |   [x] Build                  |
|                                      |   [ ] Serve                  |
|                                      |                               |
|                                      |  > Plugins                   |
|                                      |   none                       |
+--------------------------------------+-------------------------------+
| > _                                                                  |
+----------------------------------------------------------------------+
| /help commands  ctrl+c quit  tab sidebar  ctrl+l clear               |
+----------------------------------------------------------------------+
```

### Key decisions

- **2 panels** — main output (flex width) + sidebar (~30 cols, collapsible with `tab`)
- **1-line header** — site_title, serve badge, last build stats. No ASCII art.
- **Scrollable viewport** — full height output area using `bubbles/viewport`
- **Slash commands** — `/build`, `/serve`, `/doctor`, etc. with autocomplete popup
- **Bare commands also work** — `build` and `/build` both accepted
- **No prefix-key system** — removed entirely
- **Global keys** — `ctrl+c` (quit), `tab` (toggle sidebar), `ctrl+l` (clear output), `up/down/pgup/pgdown` (scroll)
- **Nord palette** — aligned with CSS theme
- **No wizard toggle** — `/next` command works always
- **No fake progress** — spinner + real task text inline in output

### Slash commands

| Command | Aliases | Action |
|---------|---------|--------|
| `/init` | `i` | Run init |
| `/update` | `update-content`, `a` | Run update-content |
| `/build` | `b` | Run build |
| `/serve` | `s` | Start/stop serve |
| `/stop` | | Stop serve |
| `/doctor` | | Run doctor |
| `/next` | `n` | Run next workflow step |
| `/theme init <name>` | | Scaffold theme |
| `/plugin enable\|disable\|toggle\|list\|install\|init` | | Plugin mgmt |
| `/version` | `v` | Show version |
| `/clear` | | Clear output |
| `/help` | `h` | Show help |
| `/quit` | `exit`, `q` | Exit |

Typing `/` shows autocomplete popup with matching commands.

### File structure

| File | Purpose |
|------|---------|
| `styles.go` | Nord palette, all lipgloss styles |
| `logsink.go` | LogSink (io.Writer -> channel bridge) |
| `model.go` | Model struct, types, Init(), Run() |
| `keys.go` | KeyMap with bubbles/key |
| `commands.go` | Command registry, parsing, autocomplete, dispatch |
| `update.go` | Update() + all message handlers |
| `view.go` | View() layout composition |
| `output.go` | Scrollable viewport wrapper for activity log |
| `sidebar.go` | Collapsible sections (Project, Workflow, Plugins) |
| `header.go` | 1-line compact header bar |
| `statusbar.go` | Bottom hint/status bar |
| `history.go` | Session log writer (unchanged) |

### Preserved interfaces

No changes to `internal/app/tui.go`:
- `Actions` struct — 12 function closures
- `Options` struct — config snapshot
- `LogSink` — io.Writer -> channel
- `History` — session log
- `Run()` signature

### Nord palette in TUI

```
Polar Night:  #2e3440 #3b4252 #434c5e #4c566a
Snow Storm:   #d8dee9 #e5e9f0 #eceff4
Frost:        #8fbcbb #88c0d0 #81a1c1 #5e81ac
Aurora:       #bf616a #d08770 #ebcb8b #a3be8c #b48ead
```

### Implementation order

1. Write all new files (clean rewrite)
2. Delete old `tui.go`
3. Update `app/tui.go` wiring if needed (Options field changes, etc.)
4. `make test && make build`
5. Manual verification with sample-site
