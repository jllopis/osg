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
  app/                App struct, Version, CLI commands (init, tui, serve, new, api)
  api/                Interactions + Comments API: SQLite stores, OAuth2 auth, HTTP handlers, CORS
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
  tui/                Bubble Tea TUI (18 modules: viewport, sidebar, logpanel, configscreen, etc.)
  vault/              Vault reader, image index, file discovery
  wikilink/           Wikilink -> Markdown rewriter (![[img|alt]])
docs/                 Specs and plans (DESIGN, TEMPLATES, TAXONOMIES, etc.)
themes/default/       Runtime theme (extracted from embedded on each build)
plugins-src/
  search/             Bundled search plugin source (Rust WASM, embedded in binary)
  llmstxt/            Official llms.txt plugin (generates /llms.txt and /llms-full.txt)
  mermaid/            Official mermaid plugin (client-side diagram rendering)
  archives/           Official archives plugin (chronological archive pages)
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
  with fields: `publish`, `featured`, `image`, `path`, `permalink`, `menu`, `abstract`, `author`, `title`
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
  - **Service management**: F5/F6 toggle serve/API, 3 modes (static, serve+api, standalone api)
  - **Log panel**: F7 toggleable bottom panel, tabs Serve/API/All, independent scroll
  - **Config editor**: F8 full-screen modal, 24 config sections, inline editing,
    `yaml.Node` round-trip preserves YAML comments, dirty tracking, Ctrl+S save
  - **Multi-channel logs**: `LogSink` with source tags, `MergeChannels()` fan-in
  - **Config schema**: `ConfigSchema()` with 8 field types (String, Bool, Int,
    StringList, IntList, StringMap, Struct, StructList)
  - Slash commands: `/serve [--api]`, `/api`, `/stop`, `/logs`, `/config`
  - Contextual hint bar changes per mode (normal/log-focus/config)

- **Permalinks**: Configurable URL patterns with extended placeholders:
  - `content_layout` supports `{date}`, `{year}`, `{month}`, `{day}`, `{slug}`, `{title}`
  - `osg.permalink` per-page override (supports same placeholders)
  - Precedence: `osg.permalink` > `osg.path` > `content_layout`
  - `{title}` is run through `slug.Slugify()` for clean URLs
  - `internal/content/content.go`: `BuildOutputPath`, `ExpandPermalink`, `expandPlaceholders`

- **Interactions** (views + likes): Page interaction system with:
  - SQLite backend (`modernc.org/sqlite`, pure Go, no CGO)
  - `osg api` standalone server + `osg serve --api` embedded mode
  - 3 API endpoints: `POST /api/v1/pageview`, `POST /api/v1/vote`, `GET /api/v1/health`
  - Client-side fingerprinting (UUID in localStorage + browser characteristics → SHA-256)
  - No IP address collection (intentional: proxied users share IPs)
  - Respects `navigator.doNotTrack`
  - View dedup: same fingerprint + same page + same day = one unique view
  - Votes: like (+1), dislike (-1), retract (0), one vote per fingerprint per page
  - Nord-styled UI at end of article: eye icon + view count, thumbs up/down buttons
  - `internal/api/` package (store, server, middleware, validation)
  - `internal/app/api.go` (RunAPI, StartAPIHandler)
  - Config: `interactions:` block with `enabled`, `api_url`, `listen`, `db_path`, `cors_origins`, `view_dedup_hours`

- **Sharing**: Compact share button with popover dropdown (Medium-style):
  - Single "Share" button with network-nodes icon, toggles popover on click
  - Popover options: X, LinkedIn, Bluesky, Email, Copy link (with divider)
  - `share.js`: popover toggle, outside click/Escape close, clipboard, `resolveURL()`
  - `sharing: true` config (default enabled), conditional JS loading
  - Share URLs: X `intent/tweet`, LinkedIn `sharing/share-offsite`, Bluesky `intent/compose`
  - i18n keys: `share`, `share_on`, `share_via_email`, `copy_link`, `link_copied`
  - CSS: `.share-wrap` (relative), `.share-toggle` (pill button), `.share-popover` (absolute dropdown)
  - Merged into unified `.article-actions` bar with interactions (views+votes left, share right)
  - `internal/theme/default/static/js/share.js`, `page.html` block `page-actions`

- **Comments**: OAuth2-authenticated comment system with threaded replies:
  - Separate SQLite database (`comments.db`) from interactions (`interactions.db`)
  - OAuth2 providers: GitHub (`read:user`), Google (`openid profile email`)
  - Auth flow: state cookie → provider redirect → callback → upsert user → session cookie
  - `osg_session` httpOnly cookie (SameSite=Lax, 30 days, Secure if HTTPS callback)
  - 6 API endpoints: auth login/callback/me/logout, comments list/create/delete
  - Unlimited nesting depth (CSS indents up to 5 levels then flattens)
  - Soft-delete: preserves comment in tree if it has non-deleted replies
  - Tree building: flat rows → two-pass map+link → prune deleted leaves
  - `comments.js`: IIFE, cookie-based auth, recursive rendering, reply inline
  - Config: `interactions.comments` block with `enabled`, `db_path`, `auth_session_days`,
    `auth_callback_url`, `providers` (each with `provider`, `client_id`, `client_secret`)
  - `internal/api/comment_store.go`, `auth.go`, `comments.go`
  - `NewServer()` accepts 5 args: `(store, cfg, logger, commentStore, authProviders)`
  - `StartAPIHandler()` returns 4 values: `(Server, Store, *CommentStore, error)`
  - Deployment: `Dockerfile`, `docker-compose.yml`, `deploy/k8s/` manifests

## Current State (as of last session)

All phases 1-16 complete plus stability/bugfix round.
Phase 11 (Plugin ecosystem) fully done: Fase A, B, C, D, E, F.
Standalone Pages, `osg new`, i18n, Kairos AI summaries, AI cache all complete.
Phase 13 (Draft preview), Phase 14 (Additional shortcodes), Phase 15 (Multi-language),
and Phase 16 (Performance & benchmarks) complete.
Test coverage expansion, shortcode docs, osg.title, menu_title, quote shortcode,
favicon support, configurable permalinks, interactions (views + likes),
sharing (social share popover), and comments (OAuth2 + threaded replies) all complete.

### Recently completed (post v0.99)
- **Enhanced `osg new` command**: Expanded osg frontmatter block with all
  recognized fields as commented-out YAML placeholders (title, image, featured,
  path, permalink, menu, abstract, author) plus active `publish` field.
  Manual frontmatter building (no yaml.Marshal) to support YAML comments.
  `yamlScalar()` helper for safe YAML value quoting. Auto-opens editor after
  file creation: `default_editor` config > `$EDITOR` env > silent skip.
  `--editor`/`--no-editor` negatable CLI flag with auto-detect default.
  `resolveEditor()` and `openEditor()` with non-fatal errors (file always created).
  `DefaultEditor` field in Config struct, "Editor" section in `ConfigSchema()`.
  22 tests (9 original + 13 new).
- **TUI enhancements (Phase 17)**: Full TUI overhaul with service management,
  log panel, and config editor. 6 phases (A-F) documented in `docs/TUI-ENHANCEMENTS.md`.
  Phase A: Multi-channel `LogSink` with source tags (general/serve/api),
  `TaggedLine` struct, `MergeChannels()` fan-in, per-source message buffers.
  Phase B: Process management for serve + API (3 modes: static, serve+api,
  standalone api), F5/F6 toggles, `/serve [--api]`, `/api`, `/stop` commands,
  badges in header/sidebar. Phase C: Log panel component (`logpanel.go`) with
  own viewport, tabs (Serve/API/All), Shift+arrow navigation, F7 toggle.
  Phase D: Config infrastructure — `ConfigSchema()` with 24 sections covering
  all config fields, `yaml.Node` CRUD (`LoadNode`/`SaveNode`/`GetNodeValue`/
  `SetNodeValue`/`SetNodeSequence`/`GetNodeSequence`/`DeleteNodeKey`) preserving
  YAML comments. Phase E: Config editor modal (`configscreen.go`, `configfields.go`)
  — full-screen 2-panel editor with section navigation, inline field editing,
  bool toggle, list add/delete, validation per type, dirty tracking, Ctrl+S save,
  confirm dialog on unsaved Esc. Phase F: Contextual hint bar (changes per
  mode: normal/log-focus/config), config reload after save updates sidebar,
  status bar shows running service addresses. 60+ TUI tests, 15+ config tests.
- **Sharing**: Social sharing buttons and title copy-link for articles.
  Copy-link icon next to `<h1>` (hover on desktop, always visible on mobile).
  Share section below article: X, LinkedIn, Bluesky, Email, Copy link buttons.
  `share.js` with `resolveURL()` for absolute URLs from relative permalinks.
  `sharing: true` config (default enabled), conditional JS/template gating.
  i18n: 5 keys (share, share_on, share_via_email, copy_link, link_copied).
  CSS: title copy-link animation, share buttons with brand hover colors.
  2 config tests (default enabled, YAML disable). Dual-file sync.
  **Redesigned** to Medium-style popover: single share button + dropdown.
  Removed title copy-link from `<h1>`. Compact `.share-toggle` pill button
  with network-nodes icon opens `.share-popover` (absolute positioned dropdown).
  Options: X, LinkedIn, Bluesky, Email, Copy link with "copied" feedback.
  Close on outside click or Escape. `aria-expanded`/`aria-haspopup` a11y.
- **Comments (OAuth2 + threaded replies)**: Full comment system with OAuth2 auth.
  Separate SQLite DB (`comments.db`). OAuth2 providers: GitHub, Google.
  Auth flow: state cookie → redirect → callback → upsert user → session cookie.
  `osg_session` httpOnly cookie (30 days, SameSite=Lax, Secure if HTTPS).
  6 API endpoints: auth login/callback/me/logout, comments list/create/delete.
  Unlimited nesting depth. Soft-delete preserves tree structure.
  `comments.js` IIFE: cookie-based auth, recursive rendering, reply inline.
  Config: `interactions.comments` block with providers, db_path, auth settings.
  `NewServer()` 5 args, `StartAPIHandler()` 4 returns, CORS with credentials.
  71 new tests (25 store + 21 auth + 19 handlers + 6 config).
  `golang.org/x/oauth2` v0.35.0. Deployment: Dockerfile, docker-compose,
  Kubernetes manifests. Dual-file sync for all theme files.
- **Permalinks**: Configurable URL patterns with extended placeholders.
  `content_layout` supports `{date}`, `{year}`, `{month}`, `{day}`, `{slug}`, `{title}`.
  `osg.permalink` per-page override with same placeholders.
  Precedence: `osg.permalink` > `osg.path` > `content_layout`.
  `{title}` slugified via `slug.Slugify()`. 15 tests.
- **Interactions (views + likes)**: SQLite-backed page interaction system.
  `internal/api/` package: store, server, CORS middleware, validation.
  3 endpoints: `POST /api/v1/pageview`, `POST /api/v1/vote`, `GET /api/v1/health`.
  Client-side fingerprinting (UUID + browser chars → SHA-256, no IP).
  `osg api` standalone server + `osg serve --api` embedded mode.
  `InteractionsConfig` in config: enabled, api_url, listen, db_path, cors_origins,
  view_dedup_hours. Nord-styled UI block in `page.html`. interactions.js with
  DoNotTrack support. 39 API tests + 3 config tests.
- **Test coverage expansion**: 1,644 lines of new tests across 7 packages.
  date (100%), slug (100%), content (96.4%), vault (90.9%), config (88.9%),
  render (74.4%), theme (84.7%).
- **Shortcode documentation**: `docs/SHORTCODES.md` with full usage guide for
  all 12 shortcodes, README features mention, QUICKSTART section.
- **menu_title from osg.path**: `MenuTitle` field on `Page`, derived from
  `osg.path` when `osg.menu=true`. Templates use `menu_title` with fallback
  to `title`. Allows menu label to differ from page title. 6 tests.
- **osg.title**: Highest-precedence page title override. Both
  `NormalizeFrontmatter` (update-content) and `ParseFile` (build) respect it.
  Precedence chain: osg.title > fm.title > fm.name > filename. 3 tests.
- **Quote shortcode**: `{{< quote author="..." source="..." >}}` generates
  `<blockquote class="quote">` with `<footer class="quote-attribution">`.
  CSS: Nord left border, subtle background, italic. 4 tests.
- **Favicon support**: `Favicon` config field, `<link rel="icon">` in
  head.html with 3-tier fallback: config.favicon > config.logo > inline
  SVG data URI (OSG logo in Nord blue #5e81ac). Dual-file sync.

### Previously completed (v0.99)
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

### Recently completed (plugin release infrastructure)
- **CI pipeline for WASM plugins**: `build-plugins` job in `.github/workflows/ci.yml`
  installs Rust + `wasm32-wasip1`, compiles each plugin in `plugins-src/`,
  uploads `.wasm` artifacts. Release job collects `.wasm` alongside Go
  binaries and includes them in checksums and release notes.
- **Auto-install on osg init**: `RunInit()` now calls `EnsureBundledPlugins()`
  to extract embedded plugins (search) and `EnsureOfficialPlugins()` to
  download missing official plugins from GitHub releases. Checks
  `plugins_enabled` against `plugins_dir`; consults curated index
  (`plugins-index.json`) to resolve repos. Non-fatal on network failure.
- **Plugin documentation**: PLUGINS.md updated with official plugins section,
  auto-install behavior, CI pipeline description, and manual install commands.
- **Tests**: 3 init tests (dirs+config, bundled plugins, no-overwrite),
  5 EnsureOfficialPlugins tests (empty args, skip bundled, skip installed,
  download from index, not in index).

### Recently completed (official plugins)
- **LLMS.txt plugin** (`plugins-src/llmstxt/`): Generates `/llms.txt` (summary with
  links) and `/llms-full.txt` (full plain-text content) following llms.txt spec.
  Hook: `build.finished`. Separates menu pages from posts, date-sorted descending,
  excludes drafts, strips HTML to plain text, decodes entities.
- **Mermaid plugin** (`plugins-src/mermaid/`): Client-side Mermaid diagram rendering.
  Hook `content.transform`: rewrites ` ```mermaid ` code blocks to `<pre class="mermaid">`.
  Hook `build.finished`: generates `/js/mermaid-init.js` that lazy-loads mermaid.js
  v11.4.1 from CDN only when diagrams exist on the page. Auto-detects dark/light theme
  via `prefers-color-scheme`. `securityLevel: 'strict'`.
- **Archives plugin** (`plugins-src/archives/`): Generates `/archive/index.html`
  (full chronological listing with year navigation) and `/archive/YYYY/index.html`
  (per-year pages with month grouping). Nord-styled CSS, responsive, dark/light mode.
  Hook: `build.finished`. Excludes drafts and menu pages.
- All three registered in `plugins-index.json`, Makefile targets added
  (`plugins-llmstxt`, `plugins-mermaid`, `plugins-archives`, `plugins-all`,
  `install-plugins`). CI auto-discovers new `plugins-src/*/` directories.
- Documentation: PLUGINS.md updated with full sections for each plugin.

### Backlog (planned, not started)
- (No remaining plugin backlog items)

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
- `modernc.org/sqlite` - SQLite driver (pure Go, no CGO) for interactions API
- `golang.org/x/oauth2` - OAuth2 client for comments authentication (GitHub, Google)
