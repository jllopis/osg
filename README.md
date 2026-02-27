# OSG - Obsidian Site Generator

A fast, opinionated static site generator that turns your [Obsidian](https://obsidian.md) vault into a fully rendered website. Written in Go with zero runtime dependencies.

## Features

- **Obsidian-native** — reads your vault directly, respects wikilinks, images, and tags
- **Two-phase pipeline** — `update-content` syncs from vault, `build` renders HTML
- **Nord-themed default** — dark/light modes, Inter + JetBrains Mono fonts, responsive
- **Taxonomy system** — tags, categories, custom taxonomies with pagination and feeds
- **Full-text search** — bundled WASM plugin, zero-JS index, keyboard-navigable dropdown
- **Image pipeline** — WebP optimization, srcset, lazy loading, lightbox gallery
- **AI summaries** — auto/manual/AI strategies via Kairos (Gemini, Anthropic, OpenAI, etc.)
- **WASM plugins** — 10 hooks, Rust/Go SDKs, GitHub registry, hot-reload in TUI
- **i18n** — English and Spanish built-in, extensible translation system
- **SEO** — canonical URLs, Open Graph, Twitter Cards, sitemaps, RSS/Atom feeds
- **Sass compilation** — compressed output, theme overrides
- **TUI** — two-panel Bubble Tea interface with slash commands
- **Related posts** — automatic recommendations by shared taxonomy terms
- **Reading progress** — scroll progress bar on article pages

## Quick Start

### Install from source

```bash
go install osg/cmd/osg@latest
```

### Or build locally

```bash
git clone https://github.com/jllopis/osg.git
cd osg
make build
# Binary at ./build/osg
```

### Initialize a site

```bash
mkdir my-site && cd my-site
osg init
```

Edit `config.yaml` to set your `vault_path`, then:

```bash
osg update-content
osg build
osg serve
```

Open `http://localhost:1313`.

## Usage

```
osg                          # Launch TUI (default command)
osg init                     # Initialize project structure
osg update-content           # Sync content from Obsidian vault
osg build                    # Build static site to public/
osg serve                    # Serve with live reload
osg serve --watch --live-reload
osg new "My Post Title"      # Create a new post in the vault
osg doctor                   # Validate configuration
osg deploy                   # Deploy to Cloudflare/rsync/S3
osg version                  # Show version info
```

### Plugin management

```
osg plugin list              # List installed plugins
osg plugin install <path>    # Install from local .wasm or GitHub
osg plugin enable <name>     # Enable a plugin
osg plugin disable <name>    # Disable a plugin
osg plugin init <name>       # Scaffold a new plugin (Rust/Go)
osg plugin search [query]    # Search the curated plugin index
osg plugin update [name]     # Update plugins to latest version
```

### Shell completions

```bash
eval "$(osg completion bash)"   # Bash
eval "$(osg completion zsh)"    # Zsh
osg completion fish > ~/.config/fish/completions/osg.fish  # Fish
```

## Configuration

OSG reads `config.yaml` (override with `-c`). Key fields:

```yaml
base_url: "https://my-site.com"
site_title: "My Blog"
site_description: "A blog about things"
theme: default
color_scheme: auto          # auto | light | dark
default_language: es        # BCP-47 language code
vault_path: "../my-vault/"
```

### Obsidian frontmatter

OSG uses an `osg:` namespace in YAML frontmatter:

```yaml
---
title: My Post
tags:
  - philosophy
osg:
  publish: true          # true | "draft" | false
  featured: true         # highlight on homepage
  image: "header.jpg"    # hero image (vault name or relative path)
  path: "/about/"        # custom URL (standalone page)
  menu: true             # add to site navigation
---
```

### Taxonomies

```yaml
taxonomies:
  - name: tags
    paginate_by: 10
    feed: true
  - name: categories
    paginate_by: 0
```

### AI summaries

```yaml
summary_strategy: ai       # auto | manual | ai
ai:
  provider: gemini          # gemini | anthropic | openai | qwen | ollama
  model: gemini-2.0-flash
  api_key: ${GEMINI_API_KEY}
  timeout: 30
  concurrency: 3
```

## Theme

The default theme uses the [Nord color palette](https://www.nordtheme.com/) with:

- Responsive design (mobile-first)
- Dark mode via `color_scheme: auto|light|dark`
- Self-hosted Inter (variable) and JetBrains Mono fonts
- Image lightbox with gallery grouping
- Reading progress bar
- Prev/next post navigation
- Related posts grid
- Sticky header with search

Create a custom theme:

```bash
osg theme init my-theme
# Edit themes/my-theme/templates/ and themes/my-theme/sass/
```

## Plugins

WASM plugins (Rust or Go via TinyGo) with 10 lifecycle hooks:

| Hook | Phase | Purpose |
|------|-------|---------|
| `config.validate` | Pre-build | Validate configuration |
| `content.transform` | Pre-render | Modify Markdown |
| `image.process` | Build | Transform images |
| `build.started` | Build | React to build start |
| `page.render` | Build | Override page context |
| `section.render` | Build | Override section context |
| `taxonomy.list.render` | Build | Override taxonomy list |
| `taxonomy.term.render` | Build | Override taxonomy term |
| `build.finished` | Post-build | React to build completion |
| `after.build` | Post-build | Deploy, notify, etc. |

Install from GitHub:

```bash
osg plugin install github.com/user/plugin-repo@v1.0.0
```

See [docs/PLUGINS.md](docs/PLUGINS.md) for the full SDK documentation.

## Project Structure

```
cmd/osg/              CLI entry point (Kong)
internal/
  app/                CLI commands (init, tui, serve, new, deploy)
  assets/             Sass pipeline, static copy, cachebust
  build/              HTML build: hierarchy, pagination, feeds, templates
  config/             Config loading, validation, defaults
  content/            Content indexer, frontmatter parsing
  i18n/               Translation system (YAML bundles)
  image/              Image optimization (WebP, srcset)
  markdown/           Goldmark renderer (GFM, footnotes, heading IDs, lightbox)
  plugin/             WASM plugin host (wazero), SDK, bundled plugins
  render/             Template engine, FuncMap, resolution
  site/               Site model (Page, Section, hierarchy)
  summary/            Summary generation (auto/manual/AI via Kairos)
  taxonomy/           Taxonomy builder (tags, categories, custom)
  theme/              Embedded default theme (//go:embed)
  tui/                Bubble Tea TUI (12 modules)
  vault/              Vault reader, wikilink rewriting
themes/default/       Runtime theme (extracted from embedded)
plugins-src/search/   Bundled search plugin (Rust WASM)
docs/                 Specifications and plans
```

## Development

```bash
make test              # Run tests
make test-coverage     # Tests with HTML coverage report
make lint              # golangci-lint
make fmt               # go fmt
make build             # Build binary to ./build/osg
make install           # Install to ~/.local/bin
make build-all         # Cross-compile for all platforms
```

## Documentation

- [DESIGN.md](docs/DESIGN.md) — Architecture, data flow, decisions
- [TEMPLATES.md](docs/TEMPLATES.md) — Template system, context, functions
- [TAXONOMIES.md](docs/TAXONOMIES.md) — Taxonomy configuration
- [THEMES.md](docs/THEMES.md) — Theme structure, Nord palette
- [PLUGINS.md](docs/PLUGINS.md) — Plugin system, WASM SDK, hooks
- [QUICKSTART.md](docs/QUICKSTART.md) — Getting started guide
- [ROADMAP.md](docs/ROADMAP.md) — Project roadmap

## License

[Apache License 2.0](LICENSE)
