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
  with fields: `publish`, `featured`, `image`, `path`, `menu`, `abstract`, `author`
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

All phases 1-16 complete plus stability/bugfix round.
Phase 11 (Plugin ecosystem) fully done: Fase A, B, C, D, E, F.
Standalone Pages, `osg new`, i18n, Kairos AI summaries, AI cache all complete.
Phase 13 (Draft preview), Phase 14 (Additional shortcodes), Phase 15 (Multi-language),
and Phase 16 (Performance & benchmarks) complete.

### Recently completed (v0.99)
- **exclude_terms in page templates**: `FilterPageTaxonomies()` strips excluded
  terms from `page.Taxonomies` before any `View()` call, so "Publicado en:",
  card pills, related pages, and prev/next never show excluded terms.
  `config.ExcludeTerms` was already filtering taxonomy index pages; now it
  also filters per-page display. 3 new tests.
- **Tilde expansion in watcher**: `config.ExpandTilde()` exported; called in
  `normalizePath()` in `serve_watch.go` so `vault_path: "~/..."` resolves
  correctly for file watching (was concatenating `~` as relative path).
- **Header scroll compaction**: Replaced direction-aware hide/show with simple
  always-visible nav bar. Only the large title row collapses via CSS
  `grid-template-rows` animation. Hysteresis (compact >80px, expand <10px)
  prevents rapid toggling. All transitions use same cubic-bezier timing.
  `brand-sm` uses `max-width` (animable) instead of `width` (not animable).
- **Stale content cleanup**: `removeStaleContent()` in `update_content.go`
  walks `content/` after export and removes directories whose `index.md`
  was not produced in the current run. Prevents duplicate pages from
  renamed/moved vault notes. 4 new tests.
- **Watch rebuild loop fix**: `EnsureDefaultTheme()` in `internal/theme/default.go`
  now compares on-disk content with embedded before writing. If identical,
  skips the write, preventing mtime changes that triggered the watcher
  into an infinite rebuild loop during `osg serve`.
- **Dual-file sync**: `themes/default/` fully synchronized with
  `internal/theme/default/` (CSS shortcode styles and tabs.js were missing).
- Phase 13: Draft preview mode (`--drafts` en `osg serve`): banner rojo en paginas draft,
  badge en listados, exclusion de feeds/sitemap, i18n draft/draft_banner
- Phase 14: Shortcodes adicionales: refactored engine (block + inline types),
  `parseArgs()` with key=value and positional args. New shortcodes: `youtube`
  (responsive 16:9, youtube-nocookie.com, extractVideoID), `twitter`/`x`
  (oEmbed + widgets.js, x.com normalization), `codepen` (iframe embed with
  height/theme/tab args, fallback link), `figure` (src/caption/alt/class/width/link),
  `tabs`+`tab` (JS tab switching, keyboard nav, a11y). CSS for all new shortcodes
  (embeds, figure, details, tabs). `tabs.js` zero-dependency script.
  33 tests (8 existing + 25 new). Dual-file sync.

### Recently completed (Phase 16)
- **Build timing**: Every `osg build` logs per-stage timing breakdown
  (plan, theme, assets, plugins, parse, transform, images, taxonomy,
  templates, render, minify) with total elapsed time.
- **CPU profiling**: `osg build --profile=cpu.prof` writes a pprof CPU
  profile. Analyze with `go tool pprof cpu.prof`.
- **Parallel image optimization**: Worker pool sized to `runtime.NumCPU()`.
  Two-phase: discover files (fast walk), then process in parallel.
- **Parallel minification**: Worker pool with `sync/atomic` counter.
  `tdewolff/minify.M` is safe for concurrent use.
- **Parallel content parsing**: ParseFile + Markdown render run in a
  worker pool; results merged sequentially into siteIndex.
- **Benchmark suite**: 18 benchmarks across 5 packages: markdown (Render,
  ExpandShortcodes, ExtractTOC), summary (PlainText, ExtractProvider,
  truncateSentence), build (MinifyDir, MinifyFile, TimingStage),
  frontmatter (SplitFrontmatter), slug (Slugify, Derive).
  Run with `go test -bench=. -benchmem ./internal/...`
- **4 tests** for BuildTimings (stage, multiple stages, log, log empty).

### Recently completed (Phase 15)
- **Multi-language config**: `LanguageConfig` struct (Code, Label), `Languages`
  field on Config, validation (empty codes, duplicate of default), helper methods
  `IsMultilingual()`, `AllLanguages()`, `LanguageLabel()`.
- **Translation linking**: `Translation` struct on Page, `LinkTranslations()`
  groups pages by slug and cross-references across languages.
- **Content export**: Non-default language pages get `/{lang}/` prefix injected
  into their content output path.
- **Build pipeline**: `Page.Lang` defaults to `cfg.DefaultLanguage`,
  `LinkTranslations()` called when multilingual, `languagesView()` helper,
  `multilingual` + `languages` exposed in configView.
- **Templates i18n**: All 58 `{{ trans "key" }}` calls updated to
  `{{ trans "key" .lang }}`. All `date_format` calls pass `.lang`.
- **hreflang alternates**: `<link rel="alternate" hreflang>` in head.html
  with `x-default` pointing to default language version.
- **Language switcher**: Nav element in header showing current language
  and links to translations. CSS Nord-styled. i18n key `aria_language`.
- **og:locale**: Uses `.lang` from render context (actual page language)
  instead of always using `default_language`.
- **Feeds**: `xml:lang` attribute on Atom `<feed>`, `<language>` element
  in RSS `<channel>`, `trans` calls pass `.lang`.
- **Sitemap**: `xmlns:xhtml` namespace, `<xhtml:link rel="alternate"
  hreflang>` for pages with translations.
- **11 tests**: 5 site tests (LinkTranslations: 2 langs, same lang,
  3 langs, empty slug, View) + 6 config tests (IsMultilingual,
  AllLanguages, LanguageLabel, validation empty/duplicate/label-default).
- **Dual-file sync**: templates, i18n YAML, CSS synchronized.

### Backlog (planned, not started)
- Paginated archives plugin (repo externo, no parte del core)

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
