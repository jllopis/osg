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
  plugin/             WASM plugin host (wazero), hooks, SDK, bundled plugin embed
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
plugins-src/
  search/             Bundled search plugin source (Rust WASM, embedded in binary)
examples/
  sample-site/        Minimal CI example site (vault_path: "")
  plugins/
    feed/             Reference example plugin (RSS feed, not bundled)
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
  auto extracts first sentences, ai uses Kairos LLM with bounded concurrency
- **AI summaries (Kairos)**: `internal/summary/kairos.go` wraps Kairos `llm.Provider`;
  supports gemini/anthropic/openai/qwen/ollama; config via `ai` section;
  bounded parallelism with semaphore channel and per-request timeouts
- **Theme**: Nord color palette, dark mode (auto/light/dark via `color_scheme`),
  Inter + JetBrains Mono fonts, responsive CSS
- **Plugins**: WASM via wazero, hook-based. Search plugin bundled (embedded
  in binary via `//go:embed`, extracted to `plugins/` at build time).
  `EnsureBundledPlugins()` in `internal/plugin/bundled.go`.
  
  **CRITICAL**: Plugins MUST be compiled with `wasm32-wasip1` target (WASI).
  `wasm32-unknown-unknown` does NOT work (no filesystem access).
  
  10 hooks: `config.validate`, `content.transform`, `image.process`,
  `build.started`, `build.finished`, `after.build`, `page.render`,
  `section.render`, `taxonomy.list.render`, `taxonomy.term.render`.
  
  CLI + TUI management. See `docs/PLUGINS.md` for full documentation.
  
- **Search plugin**: Full-text search with:
  - Indexes complete HTML content (title, summary, content, tags)
  - Generates `/search.json`, `/search/index.html`, `/js/search.js`
  - Header search bar with dropdown results (Nord-styled)
  - Standalone `/search/` page with extended results
  - OSGSearch JS class for custom integration
  - Accent-normalized search, keyboard navigation, excerpt highlighting

- **Image lightbox/gallery**: Click-to-zoom for content images with:
  - Custom Goldmark renderer wraps standalone `![alt](src)` in `<figure data-lightbox>`
  - Zero-dependency JS lightbox (~120 lines): fullscreen overlay, keyboard/touch nav
  - Automatic gallery grouping: consecutive figures -> CSS grid
  - `lightbox: true` config (default enabled), conditional JS loading
  - Nord-styled overlay, captions from alt text, counter, `prefers-reduced-motion`
  - `internal/markdown/figure.go`, `internal/theme/default/static/js/lightbox.js`

- **Wikilinks processing**: `update-content` rewrites Obsidian wikilinks:
  - Image wikilinks `![[image.png]]` → markdown images
  - Text wikilinks `[[Note Title]]` → markdown links if page exists, plain text if not
  - Two-pass algorithm: build page index (including aliases), then resolve links
  - `internal/wikilink/` package handles both image and text wikilinks

- **WASI filesystem**: `public_dir` in `configView()` is converted to absolute
  path for plugin compatibility. WASI mount maps host `/` to guest `/`, so
  relative paths don't resolve correctly.
- **TUI**: Bubble Tea with 2-panel layout, slash commands, Nord palette

## Current State (as of last session)

All phases 1-10 complete. Phase 11 (Plugin ecosystem) in progress: Fase A, B, C, D, E done.
Standalone Pages, `osg new`, i18n, Kairos AI summaries, AI cache all complete.

### Recently completed
- Phase 11-E (Registry and remote install): `osg plugin install github.com/user/repo[@tag]`
  downloads .wasm from GitHub Releases API. Curated plugin index
  (`plugins-index.json`) with `osg plugin search [query]`. Lock file
  (`.osg/plugins.lock.json`) tracks source + version. `osg plugin update [name]`
  checks latest release and re-downloads. GITHUB_TOKEN support for private
  repos. TUI `/plugin search` and `/plugin update` commands.
  15 new tests (GitHub refs, lock file, index search, download, mock server).
- Phase 11-D (SDK Go / TinyGo scaffolds): Go SDK package (`internal/plugin/sdk/`)
  with Event/Response/PluginMeta types, Plugin struct with On() handlers, ABI
  helpers. TinyGo scaffold via `osg plugin init --lang=go` (main.go with
  `//go:wasmexport`, go.mod, build.sh, README). Rust scaffold updated with
  `plugin_info` export, `bytes_to_wasm` helper, all 10 hooks documented.
  CLI `--lang` flag (default rust), TUI `/plugin init <name> [dir] [lang]`.
  Fixed embed issue (go.mod.tmpl renaming, .tmpl stripping, //go:build ignore).
  29 new tests (17 SDK + 12 scaffold).
- Image lightbox/gallery: Custom Goldmark renderer wraps standalone images in
  `<figure data-lightbox>` with `<figcaption>`. Zero-dependency JS lightbox
  (~120 lines) with fullscreen overlay, keyboard/touch nav, captions, counter.
  Automatic gallery grouping for consecutive figures via CSS grid. Config
  `lightbox: true` (default enabled). Nord-styled, accessible, responsive.
- Phase 11-A (Plugin restructuring): Search plugin moved from `examples/`
  to `plugins-src/search/` (source) and `internal/plugin/bundled/` (embedded
  `.wasm`). `EnsureBundledPlugins()` extracts bundled plugins at build time
  without overwriting user-provided ones. Search enabled by default in
  `plugins_enabled`. Feed plugin reclassified as reference example.
- Phase 11-B (Tests and robustness): 26 new tests, WASI filesystem mount fix,
  per-call timeouts, parallel plugin execution, plugin metadata via `plugin_info`.
- Phase 11-C (New hooks): `config.validate`, `content.transform`, `image.process`,
  `after.build` implemented in `internal/build/hooks.go` with 11 new tests.
- Search plugin enhanced: Full-text indexing, `/js/search.js` module, header
  search bar with dropdown, standalone `/search/` page, keyboard navigation.

### Backlog (deferred, not started)
- (empty)

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
- `github.com/jllopis/kairos` - AI/LLM framework (multi-provider: gemini, anthropic, openai, qwen, ollama)
