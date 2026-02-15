# Plan: Default Theme Upgrade

> Upgrade the embedded default theme from basic to professionally designed,
> with readability, ease of navigation and minimalism as the primary goals.
>
> **Status: COMPLETE** — All phases implemented and verified.

## Principles

- **Readability first** — optimised line length (70ch), generous whitespace, professional typefaces.
- **No JavaScript** — dark mode via `prefers-color-scheme` media query + `color_scheme` config option with `data-color-scheme` attribute; no toggle.
- **Self-hosted fonts** — Inter (variable, latin, ~95 KB woff2) + JetBrains Mono (regular, latin, ~20 KB woff2), embedded in the binary via `//go:embed`.
- **Accessibility** — skip-to-content link, WCAG AA contrast, `focus-visible`, `prefers-reduced-motion`.
- **DRY templates** — shared partials (`head`, `header`, `footer`, `card`) eliminate duplication across the 6 page templates.
- **Dynamic navigation** — header nav iterates configured taxonomies instead of hard-coding tags/area/type.
- **Nord palette** — consistent color system across light mode, dark mode, and placeholder images.

## Phase 1 — Go prerequisites (done)

| File | Change |
|---|---|
| `internal/config/config.go` | Added `SiteTitle` (default `"OSG"`), `SiteDescription` (default `""`), and `ColorScheme` (default `"auto"`, validated: auto/light/dark). Updated `Default()` and `DefaultConfigYAML()`. |
| `internal/site/site.go` | Added `WordCount` and `ReadingTime` to `Page` struct. Computed in `ParseFile()` as `len(strings.Fields(body))` and `wordCount / 200` (min 1). Added `Image` and `Featured` support via `osg` frontmatter block. |
| `internal/build/build.go` | Exposed `site_title`, `site_description`, and `color_scheme` in `configView()`. Added `generatePlaceholders()` for SVG placeholder images. |
| `internal/theme/default.go` | Changed embed directive to `//go:embed default` so subdirectories (`partials/`, `fonts/`) are included. `EnsureDefaultTheme` uses `overwrite=true`. |

## Phase 2 — Template partials & refactor (done)

### Partials

| Partial | Purpose |
|---|---|
| `partials/head.html` | Shared `<head>` — charset, viewport, title (using `site_title`), OG meta (including `og:image`), `@font-face` CSS, stylesheet link. |
| `partials/header.html` | Skip-to-content, brand link (configurable `site_title`), dynamic nav from `.taxonomies`. |
| `partials/footer.html` | Shared footer with `site_title` and current year. |
| `partials/card.html` | Reusable article card with title, date, reading time badge, summary, taxonomy pills, thumbnail image. |

### Refactored templates

All 6 templates (`index`, `page`, `section`, `taxonomy_list`, `taxonomy_single`, `404`) refactored to:
- Call partials via `{{ template "partials/..." . }}`
- Include `data-color-scheme="{{ .config.color_scheme }}"` on `<html>`
- Add breadcrumbs (section / taxonomy)
- Use `<time datetime>` elements
- Show reading time badge and word count
- Display linked taxonomy pills
- Use semantic HTML5 landmarks, ARIA attributes

### Image layout in templates

| Context | Element | Source |
|---|---|---|
| Homepage hero | `.section.featured_page.image` | Most recent featured post |
| Post list thumbnail | `.image` (in range) | Each post's image |
| Article hero | `.page.image` | Current page's image |
| OpenGraph | `<meta property="og:image">` | Page/featured image |

## Phase 3 — CSS rewrite (~1000 lines) (done)

- Custom properties: spacing scale, **Nord palette** colours for light + dark (replacing original stone/amber).
- `@font-face` for Inter + JetBrains Mono, `font-display: swap`.
- **Configurable dark mode**: forced via `html[data-color-scheme="dark"]`, auto via `@media (prefers-color-scheme: dark)` with `:not()` selectors.
- Comprehensive `.prose` for all goldmark GFM output (headings, blockquotes, code blocks with syntax highlighting background, tables, task lists, images, horizontal rules, lists, strikethrough).
- Sticky header with `backdrop-filter: blur`.
- Card hover transitions, pill/chip variants.
- **Image layout**: featured hero, post thumbnails, article hero, responsive sizing.
- Three responsive breakpoints (480 px, 640 px, 1024 px).
- Accessibility: skip-link styling, `:focus-visible` outlines, `prefers-reduced-motion`.

## Phase 4 — Fonts (done)

Downloaded and placed in `internal/theme/default/static/fonts/`:

| Font | File | Size |
|---|---|---|
| Inter | `inter-latin-var.woff2` | ~95 KB |
| JetBrains Mono | `jetbrains-mono-latin-400.woff2` | ~20 KB |

## Phase 5 — Image pipeline (done)

| Component | Package | Description |
|---|---|---|
| Vault image index | `internal/vault` | `BuildImageIndex` indexes all images by basename + vault-relative path |
| Wikilink rewriting | `internal/wikilink` | `RewriteImageLinks` converts `![[file.png\|alt]]` to `![alt](file.png)` |
| Image copying | `internal/app` | Resolves and copies vault images with absolute URL paths |
| Placeholder SVGs | `internal/placeholder` | Deterministic 1200x630 SVGs with Nord geometric patterns |
| `osg` frontmatter | `internal/publish` | `GetOSGBlock` extracts `osg.image`, `osg.featured`, `osg.publish` |

## Phase 6 — Multiple featured posts (done)

`Section.View()` rewritten with `isFeaturedPage()` helper:
- Most recent featured post -> `featured_page` (hero)
- Remaining featured -> promoted to top of `pages` list
- No featured -> most recent post becomes hero

## Phase 7 — Sync & verify (done)

1. `themes/default/` on disk synced from embedded theme via `EnsureDefaultTheme(overwrite=true)`.
2. `osg build` verified — 5 pages, 2 vault images, 4 placeholders, 42 HTML rendered, 0 errors.
3. `osg serve`, `osg theme init` verified with new theme.
4. All Go tests pass: publish (10), content (6), wikilink (4), vault (2), placeholder (6), + all existing.
5. Sample site builds correctly with `color_scheme: auto`.
