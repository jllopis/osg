# AGENTS.md - AI Agent Context for OSG

## Project Overview

OSG (Obsidian Site Generator) is a Go-based static site generator that reads
an Obsidian vault and produces a fully rendered HTML site. It follows a
two-phase pipeline: `update-content` (vault -> content/) then `build`
(content/ -> public/).

- **Module**: `osg` (Go 1.25)
- **Entry point**: `cmd/osg/main.go` (Kong CLI)
- **Remote**: `ghp:jllopis/osg`

## Key Commands

```bash
go test ./...                              # run all tests
go build -o /tmp/osg-test ./cmd/osg        # quick build
make build                                 # build via Makefile -> ./build/osg
make test                                  # tests with -v
make test-coverage                         # tests + HTML coverage report
```

## Directory Layout

```
cmd/osg/              CLI entry (Kong)
internal/
  app/                App struct, Version, CLI commands (init, tui, serve, new)
  assets/             Sass pipeline, static copy, cachebust
  build/              HTML build: hierarchy, pagination, feeds, templates
  config/             Config struct, YAML loading, defaults, validation
  content/            Content indexer (reads content/ dir, frontmatter parse)
  date/               Date extraction from path/frontmatter
  frontmatter/        YAML frontmatter parser + body split
  i18n/               Translation YAML loader, trans() and date_format() for templates
  image/              Image pipeline (copy, rewrite, optimization)
  logging/            Structured logger (slog)
  markdown/           Goldmark renderer
  placeholder/        SVG placeholder generation (Nord palette)
  plugin/             WASM plugin host (wazero), hooks, SDK
  publish/            Publish filter, osg block extraction, slug derivation
  render/             Template renderer, FuncMap, template resolution
  site/               Site model: Page, Taxonomy, hierarchy, MenuPages()
  slug/               Slug generation (unicode-safe)
  summary/            Auto-summary: Extract/Noop/AI providers
  taxonomy/           Taxonomy builder (tags, categories, etc.)
  theme/              Theme embed (//go:embed), EnsureDefaultTheme()
  tui/                Bubble Tea TUI (12 modules: viewport, sidebar, etc.)
  vault/              Vault reader, image index, file discovery
  wikilink/           Wikilink -> Markdown rewriter (![[img|alt]])
docs/                 Specs and plans (DESIGN, TEMPLATES, TAXONOMIES, etc.)
themes/default/       Runtime theme (extracted from embedded on each build)
examples/
  sample-site/        Minimal CI example site (vault_path: "")
  plugins/            WASM plugin examples (feed, search)
```

## Critical: Template Dual-File Sync

Templates exist in TWO places that MUST stay in sync:

1. `themes/default/templates/` - runtime theme (extracted from embedded)
2. `internal/theme/default/templates/` - embedded source (//go:embed)

Similarly, i18n translation files exist in TWO places that MUST stay in sync:

1. `themes/default/i18n/` - runtime theme
2. `internal/theme/default/i18n/` - embedded source (//go:embed)

Any change to a template or i18n file in one location MUST be replicated to the other.

## External Test Directories (outside osg project)

These directories are NOT inside the osg project root:

- **Sample vault**: `/Users/jllopis/src/static-gen-from-obsidian/sample-vault/Sample-Vault/`
  - Obsidian vault with ~19 notes in `02_Notes/`, images in `99_System/Attachments/`
- **Sample site**: `/Users/jllopis/src/static-gen-from-obsidian/sample-site/`
  - Generated site, `config.yaml` points `vault_path: "../sample-vault/"`
  - Run osg commands from this directory for end-to-end testing

## Documentation Conventions

- Spec docs (DESIGN, TEMPLATES, TAXONOMIES, THEMES) use **Spanish headings/prose** with English code examples
- Plan docs and AGENTS.md use English
- `Funcional.md` is entirely in Spanish
- ROADMAP/TASKS use concise shorthand
- **Accents intentionally omitted** in Spanish text (ASCII-clean: "Especificacion" not "Especificacion")
- **No YAML frontmatter** in doc `.md` files - plain Markdown with ATX headings

## Tracking: ROADMAP.md and TASKS.md

These two files track the same work in different formats and MUST stay in sync:

- `docs/ROADMAP.md` - phased roadmap with `[done]`/`[todo]`/`[doing]` markers
- `docs/TASKS.md` - flat task list with `[done]`/`[doing]`/`[todo]` markers

## Commit Style

Imperative mood, verb-first, no conventional commit prefixes (no `feat:`, `fix:`, etc.).

Examples: "Add menu pages to header template", "Fix Phase 10 roadmap markers".

## Architecture Highlights

- **osg frontmatter block**: Notes can have an `osg:` block in YAML frontmatter
  with fields: `publish`, `featured`, `image`, `path`, `menu`
- **Standalone pages**: `osg.path` overrides the date-based content layout;
  `osg.menu: true` adds the page to nav and excludes it from post listings
- **Summary strategies**: `summary_strategy` in config (auto/manual/ai);
  auto extracts first sentences, ai is a placeholder for Kairos
- **Theme**: Nord color palette, dark mode (auto/light/dark via `color_scheme`),
  Inter + JetBrains Mono fonts, responsive CSS
- **Plugins**: WASM via wazero, hook-based (transform_content, etc.)
- **TUI**: Bubble Tea with 2-panel layout, slash commands, Nord palette

## Current State (as of last session)

All phases 1-10 complete. Standalone Pages feature complete. `osg new` command complete. i18n in templates complete.

### Recently completed
- i18n in templates: `internal/i18n/` package with `Bundle`, `Trans()`, `DateFormat()`.
  Translation YAML files (en.yaml, es.yaml) with ~31 keys. `default_language` config field
  (default "es"). All 10 theme templates and 2 builtins updated with `{{ trans }}` and
  `{{ date_format }}`. 14 unit tests.

### Backlog (deferred, not started)
- Kairos AI summaries (requires API key + rate limiting)
- Image gallery / lightbox

## Key Dependencies

- `github.com/alecthomas/kong` - CLI parsing
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - TUI styling
- `github.com/charmbracelet/bubbles` - TUI components
- `github.com/knadh/koanf` - config loading
- `github.com/yuin/goldmark` - Markdown rendering
- `github.com/tetratelabs/wazero` - WASM runtime
- `github.com/fsnotify/fsnotify` - file watching
- `github.com/tdewolff/minify` - HTML/CSS/JS minification
