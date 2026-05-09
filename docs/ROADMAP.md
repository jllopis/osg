# ROADMAP

## Phase 1 - MVP update-content (done)
- [done] config YAML + defaults
- [done] vault reader + file discovery
- [done] frontmatter parser + body passthrough
- [done] publish filter + include-drafts
- [done] slug/date derivation
- [done] content writer layout
- [done] CLI init/update-content
- [done] logging + dry-run
- [done] tests basicos
- [done] documentacion basica de uso

## Phase 2 - build HTML basico (done)
- [done] especificacion templates/taxonomias
- [done] indexado de contenido
- [done] templates base (index/section/page)
- [done] render a public/

## Phase 3 - contenido avanzado (done)
- [done] taxonomias
- [done] paginacion
- [done] feeds
- [done] sitemap, robots, 404
- [done] load_data + helpers

## Phase 4 - assets (done)
- [done] sass pipeline
- [done] static copy + cachebust
- [done] tema base por defecto (templates + CSS)

## Phase 5 - extensibilidad (done)
- [done] plugins WASM con wazero
- [done] hooks y filtros externos
- [done] TUI avanzada

## Phase 6 - producto y DX (done)
- [done] plugin WASM de ejemplo (RSS feed en Rust)
- [done] live reload + watch (serve + build incremental)
- [done] build incremental con cache de contenido
- [done] search index opcional (via plugin)
- [done] starter kit de theme (scaffold + docs)
- [done] SDK/CLI para plugins (plantillas + comandos)
- [done] tests del SDK/CLI de plugins
- [done] sample site + quickstart mejorado

## Phase 7 - Theme profesional (done)
- [done] plan detallado en docs/PLAN_THEME_UPGRADE.md
- [done] Go: site_title, site_description en config + word_count/reading_time en Page
- [done] embed recursivo (partials/, fonts/)
- [done] fonts self-hosted (Inter variable + JetBrains Mono)
- [done] partials DRY (head, header, footer, card)
- [done] refactor 6 templates con partials, breadcrumbs, reading time, pills enlazados
- [done] CSS rewrite (~1000 lineas): Nord palette, dark mode configurable, prose GFM, sticky header, responsive
- [done] sync themes/default/ desde embedded + verificacion

## Phase 8 - Image pipeline, osg frontmatter y color scheme (done)
- [done] bloque `osg` en frontmatter (publish, featured, image) con fallback a campos legacy
- [done] indice de imagenes del vault (internal/vault) con resolucion por basename o path relativo
- [done] reescritura de wikilinks de imagen (![[file|alt]]) a Markdown estandar
- [done] copia de imagenes del vault al directorio de contenido con rutas absolutas
- [done] placeholders SVG auto-generados (patron geometrico Nord, determinista por titulo)
- [done] Page.Image en struct, frontmatter y templates (hero, thumbnail, og:image)
- [done] soporte de multiples posts featured: el mas reciente como hero, el resto al inicio de la lista
- [done] color_scheme en config (auto/light/dark) con validacion
- [done] data-color-scheme en <html> para forzar tema claro u oscuro sin JS
- [done] tests: publish (10), content (6), wikilink (4), vault (2), placeholder (6)

## Phase 9 - TUI profesional (done)
- [done] plan detallado en docs/PLAN_TUI_REDESIGN.md
- [done] rewrite completo: god file (1849 lineas) -> 12 modulos enfocados
- [done] layout 2 paneles: viewport scrollable (output) + sidebar colapsable
- [done] header compacto 1 linea: site_title + serve badge + build stats
- [done] slash commands con autocompletado (/build, /serve, /doctor, etc.)
- [done] comandos bare tambien soportados (build, serve, etc.)
- [done] paleta Nord en TUI (alineada con CSS theme)
- [done] viewport scrollable para output (bubbles/viewport, auto-scroll)
- [done] sidebar con secciones colapsables (Project, Workflow, Plugins)
- [done] barra de hints en footer con atajos contextuales
- [done] eliminar: prefix-key system, ASCII banner, fake progress, wizard toggle, codigo muerto
- [done] fix version (usar app.Version en vez de hardcoded)
- [done] tests para command parsing y message handling

## Summary auto-generation + Featured overlay (done)
- [done] `internal/summary/` package: Provider interface, ExtractProvider, NoopProvider
- [done] PlainText() markdown stripper (6 regexes para bold/italic, RE2-safe)
- [done] truncateSentence() con corte en oracion/palabra (max 160 chars)
- [done] tres estrategias via `summary_strategy`: auto (default), manual, ai (via Kairos LLM)
- [done] Kairos AI provider con bounded concurrency y fallback a auto
- [done] integracion en build.go: fillSummaries() despues de BuildHierarchy()
- [done] 37 tests unitarios para summary package
- [done] featured overlay CSS: gradiente transparente, texto blanco, label frosted glass
- [done] verificado en sample-site: 5 summaries generados, visibles en homepage y OG tags

## Phase 10 - Consolidacion y feeds (done)
- [done] step 1: docs sync (ROADMAP, TASKS, DESIGN)
- [done] step 2: global site RSS/Atom feed (atom.xml + rss.xml en root, configurable)
- [done] step 3: doctor improvements (diagnosticos accionables, mas checks, severidad dev/prod)
- [done] step 4: theme polish (tipografia, spacing, responsive, dark mode contrast)
- [done] step 5: TUI + build tests (cobertura para internal/build/ e internal/tui/)
- [done] step 6: image optimization (WebP, srcset, <picture>, config)
- Plan detallado en docs/PLAN_PHASE10.md

## Completed (cross-cutting)
- [done] validacion de config (paths invalidos, taxonomias mal definidas, base_url vacia en prod)
- [done] limpieza de public/ para evitar archivos stale en builds incrementales
- [done] comando de estado/diagnostico (`osg doctor` o `osg status`)
- [done] TUI: vista de progreso guiada (wizard) y panel de estado no basado en logs -> Phase 9

## Standalone Pages & Menu Navigation (done)
- [done] `osg.path` en frontmatter para override de `content_layout` (URL personalizada)
- [done] `osg.menu` en frontmatter para marcar paginas como enlaces de navegacion
- [done] exclusion automatica de menu pages del listado del homepage
- [done] `menu_pages` en contexto de templates para renderizar enlaces de navegacion
- [done] header.html: renderizar menu_pages junto a taxonomias
- [done] tests para publish, content y site con los nuevos campos
- [done] documentacion y ejemplo en sample-site

## Comando `osg new` (done)
- [done] `internal/app/new.go`: RunNew() crea nota Markdown en vault con frontmatter Obsidian-native
- [done] opciones: --tags, --publish (default: draft), --dry-run, --vault-path override
- [done] filename = titulo original + .md (convencion Obsidian)
- [done] frontmatter: title, created, tags, osg.publish
- [done] CLI: `osg new <title>` via Kong (cmd/osg/main.go)
- [done] TUI: `/new <title>` slash command con autocompletado
- [done] 9 tests unitarios + 2 tests TUI command parsing
- [done] bloque osg expandido: publish activo + placeholders comentados (title, image, featured, path, permalink, menu, abstract, author)
- [done] frontmatter manual (sin yaml.Marshal) para soportar comentarios YAML
- [done] `yamlScalar()` helper para quoting seguro de valores YAML (colons, quotes, booleans, null)
- [done] auto-apertura de editor: `default_editor` config > `$EDITOR` env > skip silencioso
- [done] `--editor`/`--no-editor` flag (negatable) con auto-detect por defecto
- [done] `resolveEditor()` y `openEditor()` en new.go; errores no-fatales (archivo siempre creado)
- [done] `DefaultEditor` field en Config + seccion "New Post" en ConfigSchema()
- [done] `new_notes_dir` config: carpeta destino dentro del vault para notas nuevas
- [done] `--notes-dir` CLI override (prioridad: CLI > config > vault root)
- [done] auto-creacion de directorio destino via MkdirAll
- [done] 30 tests unitarios (9 originales + 21 nuevos)

## i18n en templates (done)
- [done] `internal/i18n/` package: Bundle struct, New(), LoadDir(), Trans(), DateFormat()
- [done] ficheros de traduccion en.yaml y es.yaml (~31 claves cada uno)
- [done] config: `default_language` field (default "es"), validacion, normalizacion
- [done] render/funcs.go: transFunc closure sobre Bundle, dateFormatFunc con meses localizados (es/fr/de/pt/it/ca)
- [done] build.go: carga i18n (tema -> usuario), wiring a render.Context, lang en todos los contextos
- [done] 10 plantillas del tema actualizadas con {{ trans }} y {{ date_format }}
- [done] builtins actualizados: 404.html (trans), rss.xml (trans)
- [done] dual-file sync: templates y YAML en internal/theme/default/ y themes/default/
- [done] 14 tests unitarios para i18n package

## Kairos AI summaries (done)
- [done] `internal/summary/kairos.go`: KairosProvider wrapping Kairos llm.Provider
- [done] Summarize() con PlainText() pre-processing, system+user messages, temperature 0.3
- [done] NewKairosProvider() factory: soporta gemini, anthropic, openai, qwen, ollama
- [done] AIConfig en config.go: provider, model, api_key, base_url, system_prompt, timeout, concurrency
- [done] Defaults: gemini provider, gemini-3-flash-preview model, 30s timeout, 3 concurrency
- [done] Validacion: provider name, positive timeout/concurrency
- [done] DefaultConfigYAML() actualizado con seccion AI completa y documentacion de todos los providers
- [done] fillSummaries() reescrito en build.go: AI path con bounded concurrency (semaphore channel)
- [done] fillWithAI(): goroutines con per-request timeout, collect results, log errors
- [done] Fallback graceful: si falla creacion de AI provider, cae a auto strategy con warning
- [done] go.mod: 5 require + 5 replace directives para Kairos (local development)
- [done] 20 tests unitarios en kairos_test.go (mock providers, factory, concurrency, context cancellation)
- [done] Build y test end-to-end verificados

## AI summary cache + language + serve isolation (done)
- [done] AI summary cache: `.osg/cache/ai-summaries.json`, SHA-256 content hash key
- [done] AICache struct thread-safe (sync.RWMutex), load/save JSON, lookup/store
- [done] fillWithAI() checks cache before LLM call, stores results back
- [done] `--force-ai-summaries` CLI flag: ignora cache, regenera todo (con confirmacion interactiva)
- [done] `--yes`/`-y` flag: bypass confirmacion (para CI/scripts)
- [done] Language-aware prompts: `buildDefaultPrompt(lang)` inyecta idioma en system prompt
- [done] `Language` field en AIConfig y KairosProvider, wired desde `default_language`
- [done] `langDisplayName()`: BCP-47 -> nombres en ingles (es->Spanish, etc.)
- [done] Custom system_prompt ignora inyeccion de idioma
- [done] Serve isolation: `opts.SkipAI=true` en RunServe(), fallback a auto strategy
- [done] BuildOptions struct en build.go con SkipAI y ForceAISummaries
- [done] 14 tests para AI cache + 10 tests para language-aware prompts
- [done] Documentacion: DESIGN, ROADMAP, TASKS, Funcional, AGENTS.md

## Phase 11 - Plugin ecosystem (done)

### Fase A - Reestructuracion y bundled plugins (done)
- [done] A1: mover search plugin de examples/plugins/search/ a plugins-src/search/
- [done] A1: reclasificar examples/plugins/feed/ como ejemplo de referencia (no bundled)
- [done] A1: actualizar Makefile: target `plugins` compila desde plugins-src/, nuevo target `install-plugins`
- [done] A2: embeber search.wasm en binario (//go:embed) con EnsureBundledPlugins()
- [done] A3: habilitar search por defecto en plugins_enabled, link en header, claves i18n
- [done] A4: limpiar examples/plugins/, actualizar README y .gitignore

### Fase B - Tests y robustez del host (done)
- [done] B1: tests unitarios para manager.go (Load, Emit, Call, Merge, normalizePluginName)
- [done] B1: fix WASI filesystem mount (WithFSConfig + WithDirMount) para plugin file I/O
- [done] B2: timeouts por plugin call (PluginTimeout en config, context.WithTimeout)
- [done] B3: ejecucion paralela de plugins (WaitGroup, merge determinista en orden original)
- [done] B4: plugin metadata (export plugin_info, PluginMeta struct, Metadata() en Manager, osg plugin list mejorado)

### Fase C - Nuevos hooks (done)
- [done] C1: hook config.validate (emitido tras cargar config, errores detienen build)
- [done] C2: hook content.transform (modifica Markdown antes del render)
- [done] C3: hook image.process (transformacion de imagenes via WASI filesystem)
- [done] C4: hook after.build (post-build garantizado, para deploy/notificaciones)

### Fase D - SDK Go (TinyGo) (done)
- [done] D1: package osg-plugin-sdk-go (tipos Event/Response/PluginMeta, helpers, ABI) en internal/plugin/sdk/
- [done] D2: scaffold TinyGo (osg plugin init --lang=go, template main.go con wasmexport, build.sh, README)
- [done] D3: actualizar scaffold Rust con plugin_info, bytes_to_wasm y doc de 10 hooks
- [done] D4: CLI --lang flag en PluginInitCmd, TUI /plugin init <name> [dir] [lang]
- [done] D5: fix embed issue (go.mod.tmpl + .tmpl stripping, //go:build ignore en template main.go)
- [done] D6: 12 tests scaffold (Go+Rust content, .tmpl stripping, tinygo alias, default lang, errors) + 17 tests SDK

### Fase E - Registry e instalacion remota (done)
- [done] E1: instalacion desde GitHub (osg plugin install github.com/user/repo[@tag])
- [done] E1: deteccion automatica de GitHub refs, descarga de .wasm desde GitHub Releases API
- [done] E1: soporte GITHUB_TOKEN para repos privados y rate limits
- [done] E2: indice curado (plugins-index.json en repo, osg plugin search [query])
- [done] E2: busqueda por nombre, descripcion y autor (case-insensitive)
- [done] E3: lock file (.osg/plugins.lock.json) con source + version por plugin
- [done] E3: osg plugin update [name] con check contra latest GitHub release
- [done] E4: CLI --lang flag, TUI /plugin search y /plugin update
- [done] E5: 15 tests unitarios (GitHub refs, lock file, index search, download, mock server)

### Fase F - Documentacion y templates (done)
- [done] F1: actualizar docs/PLUGINS.md (SDK Go, registry, GitHub install, search, update)
- [done] F2: actualizar ROADMAP.md y TASKS.md con Phase 11 completo
- [done] F3: templates del tema ya incluyen link /search/ en header e i18n (desde Fase A)

## Image gallery / lightbox (done)
- [done] Custom Goldmark renderer: `<figure data-lightbox>` con `<figcaption>` para imagenes standalone
- [done] Lightbox JS: overlay fullscreen, navegacion teclado/touch, captions, counter
- [done] Galeria automatica: figures consecutivas en CSS grid responsive
- [done] Config `lightbox: true` (default habilitado)
- [done] CSS Nord-styled: overlay, botones, transiciones, responsive, prefers-reduced-motion
- [done] Tests unitarios: 10 tests para figure rendering + test de paragrafos normales
- [done] Dual-file sync: CSS, JS, templates en internal/theme/ y themes/default/

## Phase 12A - DX critico + SEO quick wins (done)
- [done] LICENSE file (Apache 2.0)
- [done] .editorconfig
- [done] .golangci.yml
- [done] SEO: canonical URL en head.html (<link rel="canonical">)
- [done] SEO: meta description con page.summary (fallback a site_description)
- [done] SEO: Twitter Card meta tags (twitter:card, twitter:title, twitter:description, twitter:image)
- [done] SEO: OG tags en todos los templates (index, section, taxonomy), og:site_name, og:locale
- [done] SEO: og:type article vs website segun tipo de pagina
- [done] SEO: article:published_time y article:modified_time para paginas
- [done] Sass: --style compressed para CSS minificado
- [done] Goldmark: heading IDs automaticos (parser.WithAutoHeadingID)
- [done] Goldmark: extension Footnote habilitada
- [done] Font preload: <link rel="preload"> para Inter y JetBrains Mono (woff2)
- [done] Dual-file sync: head.html en internal/theme/ y themes/default/
- [done] Tests: all passing

## Phase 12B - CI/CD + README + Content features (done)
- [done] GitHub Actions CI/CD pipeline (test, build, lint, vet en paralelo)
- [done] README profesional (features, install, usage, config, theme, plugins, structure)
- [done] Shell completions: `osg completion bash|zsh|fish`
- [done] .goreleaser.yml para releases multi-plataforma
- [done] Related posts: scoring por terms compartidos, top 3, grid en page.html
- [done] Prev/next navigation: cronologica (newest-first), excluye menu pages
- [done] Reading progress bar: JS scroll-based, CSS accent color, fixed top
- [done] i18n: claves newer_post, older_post, related_posts (en + es)
- [done] page.html: prev/next nav, related posts grid, progress bar
- [done] CSS: post-nav, related-card, reading-progress-bar, responsive
- [done] Dual-file sync: templates, i18n, CSS, JS
- [done] 6 tests unitarios para relatedPages()
- [done] Tests: all passing

## Phase 12C - Minificacion + TOC + Syntax + Shortcodes (done)
- [done] HTML/CSS/JS/JSON/SVG/XML minification (tdewolff/minify/v2, post-render batch in-place)
- [done] Config `minify: true` (default habilitado), campo Minify en Config struct
- [done] 5 tests unitarios para minificacion (HTML, CSS, skip images, empty, multiple types)
- [done] Table of Contents: ExtractTOC() regex h2-h6 desde HTML renderizado, TOCView() para templates
- [done] Partial template `partials/toc.html` con toggle colapsable
- [done] 7 tests unitarios para TOC (no headings, h1 ignored, basic, HTML in heading, entities, nil view, basic view)
- [done] Syntax highlighting: goldmark-highlighting/v2 con Chroma Nord style, CSS-class mode
- [done] css/syntax.css con colores Nord para tokens (CSS custom properties)
- [done] Shortcodes: `{{< name [args] >}}content{{< /name >}}` expandidos antes de Goldmark
- [done] 4 shortcodes built-in: note, warning, tip (admonitions), details (collapsible)
- [done] Per-name compiled regexes (Go regexp no soporta backreferences)
- [done] 8 tests unitarios para shortcodes (note, warning, tip, details, unknown, no shortcodes, multiple)
- [done] CSS: estilos TOC (~50 lineas), estilos admonitions (~50 lineas) con colores Nord
- [done] i18n: claves toc_title, toc_label en en.yaml y es.yaml
- [done] Dual-file sync: templates, i18n, CSS en internal/theme/ y themes/default/

## Theme system improvements (done)
- [done] theme.yaml metadata (name, description, author, version, min_osg_version, parent)
- [done] Theme inheritance: parent chain resolution for templates, static, i18n, sass
- [done] Cycle detection in parent chain
- [done] Block-based overridable sections in page.html, index.html, section.html
- [done] `osg theme init --parent <name>` scaffold for child themes
- [done] `osg theme list` CLI + TUI command
- [done] Doctor checks: theme.yaml validation, parent chain errors
- [done] ThemeMeta struct, LoadMeta(), ResolveChain(), ListThemes(), WriteMeta()
- [done] TemplateLoader.ThemeChain, render.NewWithChain(), assets.PrepareWithChain()
- [done] 19 tests unitarios (meta, chain, scaffold, list, cycle, edge cases)
- [done] Dual-file sync: theme.yaml, templates en internal/theme/ y themes/default/
- [done] Documentacion THEMES.md actualizada con herencia, bloques, child themes

## Backlog

- [done] i18n en templates

### Phase 13 - Draft preview mode (done)
- [done] Flag `--drafts` en `osg serve` para incluir notas con `publish: "draft"`
- [done] Banner visual en paginas draft (fondo rojo, texto "borrador / draft")
- [done] Badge "Draft" en listados (card, post-item, featured)
- [done] Excluir drafts de feeds RSS/Atom y sitemap incluso en preview
- [done] Claves i18n: `draft`, `draft_banner` (en + es)
- [done] CSS: `.draft-banner` (pagina), `.draft-badge` (listados), Nord red (#bf616a)
- [done] Tests: feedPages excluye drafts, collectSitemapEntries excluye drafts
- [done] Dual-file sync: templates, CSS, i18n

### Phase 14 - Shortcodes adicionales (done)
- [done] Refactor shortcode engine: block (paired) + inline (self-closing) types
- [done] `parseArgs()`: key="value", key='value', key=value, bare positional args
- [done] `youtube` shortcode: responsive 16:9 embed, youtube-nocookie.com, extractVideoID (bare ID, full URL, short URL, embed URL)
- [done] `twitter`/`x` shortcode: oEmbed blockquote + widgets.js, x.com → twitter.com normalization
- [done] `codepen` shortcode: iframe embed with height/theme/tab args, fallback link for invalid URLs
- [done] `figure` avanzado: src, caption, alt, class, width, link args; figcaption from inner content or caption arg
- [done] `tabs` + `tab` shortcodes: container with data-tab-title, JS tab switching with keyboard nav (arrows, Home, End)
- [done] CSS: embeds (responsive youtube, centered twitter, codepen), figure (.figure, .figure.wide), details, tabs (.tabs-nav, .tab-btn, .tab-content)
- [done] JS: tabs.js (zero-dependency, a11y: role=tablist/tab, aria-selected, keyboard nav)
- [done] 33 tests unitarios (8 existentes + 25 nuevos)
- [done] Dual-file sync: CSS, JS, templates

### Bugfixes and stability (done)
- [done] exclude_terms filtering in page templates ("Publicado en:", card pills, related pages)
- [done] Tilde expansion (~) in vault_path for file watcher in `osg serve`
- [done] Header scroll compaction: flicker-free, always-visible nav bar with smooth title collapse
- [done] Stale content cleanup: `update-content` removes orphaned content/ directories automatically
- [done] Watch loop fix: EnsureDefaultTheme skips writes when on-disk content is identical (prevents infinite rebuild)
- [done] Dual-file sync: themes/default/ fully synchronized with internal/theme/default/

### Phase 15 - Multi-idioma real (done)
- [done] Config: `LanguageConfig` struct (Code, Label), `Languages []LanguageConfig`, validacion (empty codes, duplicates of default), helpers `IsMultilingual()`, `AllLanguages()`, `LanguageLabel()`
- [done] Site model: `Translation` struct, `Translations []Translation` en Page, `LinkTranslations()` agrupa por slug y cross-referencia entre idiomas
- [done] Content export: inyeccion de `/{lang}/` prefix en output path para idiomas no-default
- [done] Build pipeline: `Page.Lang` default a `cfg.DefaultLanguage`, `LinkTranslations()` cuando multilingual, `languagesView()`, `multilingual` + `languages` en configView
- [done] Templates i18n: todas las 58 llamadas `{{ trans "key" }}` actualizadas a `{{ trans "key" .lang }}`, todos los `date_format` pasan `.lang`
- [done] hreflang alternates: `<link rel="alternate" hreflang>` en head.html con x-default apuntando al idioma por defecto
- [done] Language switcher: nav en header con idioma actual resaltado y links a traducciones, CSS Nord-styled
- [done] og:locale: usa `.lang` del contexto (idioma real de la pagina) en vez de siempre default_language
- [done] Feeds: `xml:lang` en Atom, `<language>` en RSS, `trans` con `.lang`
- [done] Sitemap: namespace `xhtml`, `<xhtml:link rel="alternate" hreflang>` para paginas traducidas
- [done] i18n keys: `aria_language` (en: "Language", es: "Idioma")
- [done] 11 tests: 5 site (LinkTranslations: 2 idiomas, mismo idioma, 3 idiomas, slug vacio, View) + 6 config (IsMultilingual, AllLanguages, LanguageLabel, validacion empty/duplicate/label-default)
- [done] Dual-file sync: templates, i18n, CSS en internal/theme/ y themes/default/

### Phase 16 - Performance y benchmarks (done)
- [done] Build timing: instrumentacion por stages (plan, theme, assets, plugins, parse, transform, images, taxonomy, templates, render, minify) con log estructurado
- [done] CPU profiling: `osg build --profile=cpu.prof` escribe perfil pprof analizable con `go tool pprof`
- [done] Paralelizacion de image optimization: worker pool con `runtime.NumCPU()` goroutines, fase discover + fase process
- [done] Paralelizacion de minification: worker pool con `runtime.NumCPU()` goroutines, `sync/atomic` counter
- [done] Paralelizacion de content parsing: worker pool para ParseFile + Markdown render, merge secuencial en siteIndex
- [done] Benchmark suite: 18 benchmarks en 5 packages (markdown, summary, build, frontmatter, slug)
- [done] 4 tests unitarios para BuildTimings (stage, multiple stages, log, log empty)

### Test coverage expansion (done)
- [done] 1,644 lineas de nuevos tests en 7 packages: date (100%), slug (100%), content (96.4%), vault (90.9%), config (88.9%), render (74.4%), theme (84.7%)

### Shortcode documentation (done)
- [done] `docs/SHORTCODES.md`: guia completa de uso de los 11 shortcodes con ejemplos, argumentos, referencia rapida
- [done] README.md: mencion de shortcodes en features + link a docs
- [done] QUICKSTART.md: seccion de shortcodes

### osg.title y menu_title (done)
- [done] `osg.title` en frontmatter como titulo de maxima precedencia (osg.title > fm.title > fm.name > filename)
- [done] `menu_title` derivado de `osg.path` cuando `osg.menu=true` para que el label del menu pueda diferir del titulo de pagina
- [done] Templates header/footer usan `menu_title` con fallback a `title`
- [done] 9 tests (3 osg.title + 6 menu_title)

### Quote shortcode (done)
- [done] Shortcode `quote` con atribucion opcional: `author` (posicional o key=value) y `source`
- [done] HTML: `<blockquote class="quote">` con `<footer class="quote-attribution">`, `<cite>`, `<span>`
- [done] CSS Nord-styled: borde izquierdo Nord blue, background sutil, fuente italic para cita
- [done] 4 tests unitarios (bare, author, author+source, source-only)
- [done] Documentacion en SHORTCODES.md
- [done] Dual-file sync: CSS en internal/theme/ y themes/default/

### Favicon support (done)
- [done] Campo `Favicon` en Config struct (`internal/config/config.go`)
- [done] `favicon` expuesto en `configView()` para templates
- [done] `head.html`: `<link rel="icon">` con fallback 3 niveles: config.favicon > config.logo > SVG inline (logo OSG en Nord blue #5e81ac)
- [done] Documentado en `DefaultConfigYAML()` (seccion Theme & appearance)
- [done] Dual-file sync: head.html en internal/theme/ y themes/default/

### Permalinks configurables (done)
- [done] Placeholders extendidos en `content_layout`: `{year}`, `{month}`, `{day}`, `{title}` (ademas de `{date}` y `{slug}` existentes)
- [done] `expandPlaceholders()` funcion interna compartida entre `BuildOutputPath` y `ExpandPermalink`
- [done] `{title}` pasa por `slug.Slugify()` para URLs limpias
- [done] `osg.permalink` en frontmatter: override per-page de URL con soporte de placeholders
- [done] Precedencia: `osg.permalink` > `osg.path` > `content_layout`
- [done] `osg.permalink` es solo URL (sin side-effects como `osg.path` que tambien setea `menu_title`)
- [done] Resolucion de titulo para placeholders: osg.title > fm.title > fm.name > filename
- [done] 15 tests unitarios (BuildOutputPath placeholders, ExpandPermalink, defaults, edge cases)

### Interacciones: vistas y likes (done)
- [done] `InteractionsConfig` struct en config.go: enabled, api_url, listen, db_path, cors_origins, view_dedup_hours
- [done] Defaults: enabled=false, listen=":8090", db_path=".osg/interactions.db", view_dedup_hours=24
- [done] Validacion y normalizacion en `Load()`: view_dedup_hours>=1, trailing slash en api_url
- [done] `interactions_enabled` y `interactions_api_url` expuestos en `configView()` para templates
- [done] 3 tests unitarios para config interactions (defaults, YAML loading, normalization)

#### Backend: API server
- [done] `internal/api/store.go`: SQLite store con `modernc.org/sqlite` (pure Go, sin CGO)
- [done] Schema: `page_views` (page_path, fingerprint, created_at) + `page_votes` (page_path PK, fingerprint PK, vote, timestamps)
- [done] `RecordView()`: INSERT total view + INSERT OR IGNORE para dedup por fingerprint/dia
- [done] `Vote()`: UPSERT voto (1=like, -1=dislike, 0=retract delete)
- [done] `GetStats()`: COUNT views, COUNT DISTINCT fingerprints (unique), SUM likes/dislikes, user_vote
- [done] WAL mode + busy timeout para concurrencia
- [done] `internal/api/validation.go`: PageViewRequest y VoteRequest con validacion (path, fingerprint, vote range)
- [done] `internal/api/middleware.go`: CORS middleware con allowlist de origenes, preflight OPTIONS
- [done] `internal/api/server.go`: HTTP server con 3 endpoints: POST /api/v1/pageview, POST /api/v1/vote, GET /api/v1/health
- [done] Cada endpoint retorna stats completos: {views, unique, likes, dislikes, user_vote}
- [done] 14 tests store + 13 tests server/CORS + 12 tests validation = 39 tests API

#### CLI: comandos osg api y osg serve --api
- [done] `internal/app/api.go`: RunAPI() servidor standalone, StartAPIHandler() para embedding
- [done] `cmd/osg/main.go`: `APICmd` struct con --listen flag, `osg api` dispatch
- [done] `ServeCmd.API` bool flag: `osg serve --api` embebe API endpoints en mismo servidor
- [done] `internal/app/serve.go`: refactored de FileServer a ServeMux, monta API cuando --api activo
- [done] Shell completion actualizado con `api` command

#### Frontend: fingerprinting + UI
- [done] `interactions.js`: client-side fingerprinting (UUID en localStorage + User-Agent, screen, devicePixelRatio, timezone, language, platform, hardwareConcurrency, colorDepth -> SHA-256)
- [done] Sin IP address: usuarios detras de proxy comparten IP, IPs domesticas cambian
- [done] Auto-registro de pageview al cargar pagina (respeta `navigator.doNotTrack`)
- [done] Botones like/dislike con toggle, retract (voto 0), estado aria-pressed
- [done] Actualizacion de contadores en tiempo real

#### Templates y tema
- [done] `page.html`: bloque `page-interactions` entre `page-taxonomies` y `page-nav`
- [done] SVG icons inline: ojo (views), pulgar arriba (like), pulgar abajo (dislike)
- [done] `style.css`: seccion INTERACTIONS (~80 lineas) con estilo Nord, responsive, focus-visible, active states
- [done] i18n: claves `interactions_like` y `interactions_dislike` en en.yaml y es.yaml
- [done] JS condicional: `{{ if .config.interactions_enabled }}<script src="interactions.js">{{ end }}`
- [done] Dual-file sync: page.html, interactions.js, style.css, en.yaml, es.yaml

### Sharing (social buttons + title copy-link) (done)
- [done] Copy-link icon next to article `<h1>` (hover on desktop, always visible on mobile)
- [done] Share section below article: X, LinkedIn, Bluesky, Email, Copy link buttons
- [done] `share.js`: clipboard helper, `resolveURL()` for absolute URLs from relative permalinks
- [done] `sharing: true` config (default enabled), conditional JS/template gating
- [done] i18n: 5 keys (share, share_on, share_via_email, copy_link, link_copied)
- [done] CSS: title copy-link opacity animation, share buttons with brand colors on hover
- [done] 2 config tests (default enabled, YAML disable)
- [done] Dual-file sync: page.html, share.js, style.css, en.yaml, es.yaml

### Comentarios: sistema de comentarios con OAuth2 (done)
- [done] `CommentsConfig` y `AuthProviderConfig` structs en config.go con defaults, normalizacion y validacion
- [done] `DefaultConfigYAML()` documentado con seccion comments completa
- [done] `CommentStore` en `internal/api/comment_store.go`: SQLite separada con tablas users, sessions, comments
- [done] User CRUD: UpsertUser, GetUserByProvider, GetUserByID
- [done] Session CRUD: CreateSession (crypto random), ValidateSession, DeleteSession, CleanExpiredSessions
- [done] Comment CRUD: CreateComment, GetComment, SoftDeleteComment, ListComments
- [done] Tree building: buildCommentTree (two-pass map+link), pruneDeletedLeaves
- [done] OAuth2 auth handlers en `internal/api/auth.go`: BuildAuthProviders (GitHub read:user, Google openid profile email)
- [done] AuthHandlers: HandleLogin, HandleCallback, HandleMe, HandleLogout
- [done] State cookie + return_to flow, code exchange, user info fetch, session creation
- [done] Comment HTTP handlers en `internal/api/comments.go`: HandleList, HandleCreate, HandleDelete
- [done] CreateCommentRequest validation (page_path, body max 10000, parent_id same page)
- [done] Ownership check for delete (solo propios comentarios)
- [done] Server 5-arg NewServer (commentStore + authProviders nil-safe), rutas condicionales
- [done] CORS middleware con withCredentials (GET/DELETE, Access-Control-Allow-Credentials)
- [done] `RunAPI` y `StartAPIHandler` (4 retornos) con CommentStore lifecycle
- [done] `configView()`: comments_enabled, comments_providers con display labels
- [done] comments.js: IIFE, cookie auth (credentials include), login URLs, recursive rendering, reply inline, delete, timeAgo, HTML escaping
- [done] page.html: bloque page-comments entre page-actions y page-nav, JS condicional
- [done] CSS: seccion COMMENTS (~200 lineas), nesting 5 niveles, avatars, responsive
- [done] i18n: 13 claves comentarios en en.yaml y es.yaml
- [done] 25 tests CommentStore + 21 tests auth + 19 tests comment handlers + 6 tests config = 71 tests nuevos
- [done] golang.org/x/oauth2 v0.35.0 como dependencia directa
- [done] Dual-file sync: page.html, comments.js, style.css, en.yaml, es.yaml
- [done] Dockerfile multi-stage (golang:1.25-alpine -> alpine:3.21, non-root, volumes)
- [done] docker-compose.yml con osg-data volume
- [done] deploy/k8s/: configmap, pvc, deployment (health probes, resource limits), service

### Phase 17 — TUI Enhancements: Server Management + Config Editor (done)

Spec: `docs/TUI-ENHANCEMENTS.md`

#### Fase A: Multi-channel log system
- [done] Refactor LogSink con campo source (general/serve/api)
- [done] taggedLogLineMsg reemplaza logLineMsg
- [done] 3 LogSinks en RunTUI (general, serve, api)
- [done] Model almacena mensajes separados por fuente
- [done] Tests multi-canal

#### Fase B: Serve + API process management
- [done] Nuevos campos Model: apiRunning, apiCancel, serveMode
- [done] Nuevas Actions: ServeWithAPI, RunAPI
- [done] Slash commands: /serve --api, /api, /stop serve, /stop api
- [done] Teclas dedicadas: F5 (serve), F6 (api)
- [done] Sidebar seccion "Services" con badges
- [done] Header badges separados [SERVE] [API]

#### Fase C: Log panel
- [done] Componente LogPanel con viewport propio y tabs (Serve/API/All)
- [done] Toggle con F7 y /logs
- [done] Layout: panel inferior 1/3 de terminal
- [done] Foco independiente (Shift+up/down para log panel)
- [done] Estilos Nord para tabs y panel

#### Fase D: Config infrastructure
- [done] ConfigSchema() en internal/config/schema.go: secciones, campos, tipos, descripciones
- [done] yaml.Node helpers en internal/config/yamlnode.go: LoadNode, GetNodeValue, SetNodeValue, SaveNode
- [done] Refactorizar UpdatePluginsEnabled para preservar comentarios YAML
- [done] Tests schema + yamlnode round-trip

#### Fase E: Config editor modal screen
- [done] ConfigScreenModel: layout 2 paneles (secciones + campos)
- [done] Editores inline por tipo: String, Bool, Int, Dropdown, StringList, IntList, StringMap, StructList
- [done] Dirty state tracking con indicador visual
- [done] Ctrl+S save con yaml.Node, Esc con dialogo unsaved changes
- [done] Slash command /config, tecla F8

#### Fase F: Integration and polish
- [done] Status bar contextual por modo (normal/config/logs)
- [done] Config reload tras guardar (actualizar sidebar)
- [done] Docs: DESIGN.md seccion TUI, AGENTS.md modulos nuevos

### Official plugins — release assets (done)

Plugins mantenidos en el mismo repo (`plugins-src/`), compilados por CI,
publicados como release assets en GitHub. El usuario instala con
`osg plugin install <nombre>`. No se embeben en el binario (solo `search`
esta embebido).

#### CI pipeline
- [done] Job `build-plugins` en CI: Rust + wasm32-wasip1, compila plugins-src/
- [done] Release assets incluyen .wasm junto con binarios y checksums
- [done] Documentar pipeline en PLUGINS.md

#### Auto-install en osg init
- [done] `osg init` detecta plugins en `plugins_enabled` que no estan en `plugins_dir`
- [done] Si el plugin es oficial (esta en el indice curado), lo descarga automaticamente del release
- [done] Logging claro: "installing plugin <name>... installed plugin <name>"
- [done] `osg init` extrae plugins bundled (search) via EnsureBundledPlugins

#### Paginated archives
- [done] Genera `/archive/` con listado cronologico por ano/mes
- [done] Pagina principal /archive/ con navegacion por anos y listado completo
- [done] Paginas por ano /archive/YYYY/ con agrupacion por mes
- [done] CSS Nord-styled, responsive, dark/light mode

#### LLMS.txt Generator
- [done] Genera `/llms.txt` con contenido del sitio en formato apto para LLMs
- [done] Genera `/llms-full.txt` con contenido completo (plain text) de cada pagina
- [done] Separa paginas de menu (standalone) de posts normales
- [done] Ordenado por fecha descendente, excluye drafts

#### Mermaid diagrams
- [done] Enfoque client-side: inyecta mermaid.js (CDN) y transforma bloques ```mermaid en markup renderizable
- [done] Hook `content.transform`: reescribe bloques de codigo mermaid a `<pre class="mermaid">`
- [done] Hook `build.finished`: genera `mermaid-init.js` que carga mermaid CDN solo si hay diagramas
- [done] Auto-deteccion de tema (dark/light) via prefers-color-scheme

---

## Phase 18 — Estabilidad, CI y deuda tecnica (done)

Prioridad maxima: corregir errores de CI, eliminar warnings del linter y aumentar cobertura
en los paquetes criticos que sostienen el pipeline de build y deploy.

### 18A — Fix CI lint errors (done)
- [done] Corregir 5 errores `errcheck` en `internal/app/api.go` y `internal/app/serve.go` (Close() sin comprobar error)
- [done] Verificar CI pipeline verde tras fix (GitHub Actions lint job)
- [done] Documentar politica de linting: zero tolerance en CI, `nolint` solo con justificacion (`docs/RELEASE.md`)

### 18B — Test coverage: build package (37% -> 67.4%) (done)
- [done] Tests para cleanupRemovedOutputs, removeEmptyParents, generatePlaceholders
- [done] Tests para baseContext, configView, commentsProvidersView, languagesView
- [done] Tests para fillSummaries (auto, manual strategies), fillWithProvider
- [done] Tests para feedPages (drafts filtering), siteFeedContext, collectSitemapEntries (hreflang)
- [done] Tests para hashConfig, hashContent, hashDir, hashPlugins, hashAssets, hashTemplates
- [done] Tests para buildCacheFrom, loadBuildCache, saveBuildCache round-trip
- [done] Tests para buildPlan.shouldRenderPage, buildOutputsIndex, buildStatsView
- [done] Tests para renderPages, renderSections, renderTaxonomies, renderSiteFeed, renderSitemap, renderRobots, renderNotFound
- [done] Tests para taxonomyPagePath, latestUpdated, sectionUpdated, taxonomyIndexUpdated
- [done] Tests para isNilInterface, pageContext, sectionContext, applyPluginOverrides
- [done] 3,838 lineas en build_coverage_test.go, cobertura 37.2% -> 67.4%

### 18C — Test coverage: deploy package (39% -> 88.7%) (done)
- [done] Tests para CloudflareProvider.Deploy error paths (nonexistent dir, empty dir, wrangler toml generation)
- [done] Tests para RsyncProvider.Deploy argument construction (port, keyfile, exclude, extra flags)
- [done] Tests para S3Provider.Deploy URL construction (bucket, path, endpoint, region, ACL)
- [done] Tests para runCommand (success, failure, not found, cancelled context)
- [done] Tests para Register custom provider, S3 Validate con profile en config
- [done] Cobertura 39% -> 88.7%

### 18D — Test coverage: assets package (52% -> 85.1%) (done)
- [done] Tests para PrepareWithChain (multiple theme dirs, static copy, sass disabled)
- [done] Tests para copyStaticChain (file override por child theme, merge de dirs)
- [done] Tests para compileSassChain (conflict detection, compile_sass=false)
- [done] Tests para compileSass/compileSassDir (dir inexistente, sass_dir vacio)
- [done] Tests para copyStatic (dir inexistente), Prepare (flujo basico)
- [done] Cobertura 51.6% -> 85.1%

### 18E — Test coverage: plugin package (62% -> 82.1%) (done)
- [done] Tests para Emit (nil plugins, multiple plugins, timeout, nil logger)
- [done] Tests para readPluginInfo (no plugin_info export)
- [done] Tests para Call (search plugin con mock)
- [done] Tests para CheckUpdate/UpdatePlugin (mock HTTP server, already up-to-date, fetch fails)
- [done] Tests para InstallFromGitHub, fetchGitHubRelease con GITHUB_TOKEN
- [done] Tests para LoadLockFile (corrupt JSON, null plugins, round-trip), SaveCreatesNestedDirs
- [done] Tests para FetchIndexFrom (invalid JSON, connection error, multiple plugins)
- [done] Tests para EnsureBundledPlugins, downloadFile, parseGitHubRef
- [done] Cobertura 61.7% -> 82.1%

### 18F — Documentacion tecnica (done)
- [done] ADR-001: unsafe.Pointer en plugin SDK WASM (`docs/adr/001-unsafe-pointer-wasm-sdk.md`)
- [done] ADR-002: dual-file sync strategy (`docs/adr/002-dual-file-sync-theme.md`)
- [done] Politica de versionado semantico, proceso de release y politica de linting (`docs/RELEASE.md`)

## Phase 19 — Validacion de contenido: `osg check` (done)

Nuevo comando para detectar problemas en el contenido antes de publicar.
Complementa `osg doctor` (que valida config/entorno) con validacion del contenido real.
Nota: broken wikilinks y large images ya cubiertos por `osg doctor`, no duplicados aqui.

### 19A — Links internos rotos (done)
- [done] Escanear contenido renderizado para detectar links internos (`href="/..."`) que no corresponden a ninguna pagina generada
- [done] Wikilinks rotos delegados a `osg doctor` (ya implementado, no duplicar)
- [done] Reporte con fichero fuente y link roto
- [done] Exit code != 0 si hay errores (integrable en CI)

### 19B — Imagenes huerfanas y referencias rotas (done)
- [done] Detectar imagenes referenciadas en contenido renderizado que no existen en static/, theme static/ ni content/
- [done] Detectar imagenes copiadas al directorio de contenido que no estan referenciadas por ninguna pagina (huerfanas)
- [done] Reporte con tamano de imagenes huerfanas (KB/MB)

### 19C — Frontmatter incompleto o inconsistente (done)
- [done] Detectar posts sin fecha (no se puede ordenar cronologicamente)
- [done] Detectar posts sin tags (posible contenido sin categorizar, excluye menu pages)
- [done] Detectar duplicados de slug (URLs colisionantes)
- [done] Detectar valores de `osg.permalink` que colisionan entre si
- [done] Severidad: slug duplicado y permalink collision = error; fecha/tags faltantes = warning

### 19D — Integracion (done)
- [done] CLI: `osg check` con flags `--links`, `--images`, `--frontmatter`, `--all` (default)
- [done] TUI: `/check` slash command
- [done] Formato de salida: texto (default), JSON (`--json`)
- [done] Exit code != 0 si hay errores (integrable en CI)
- [done] 17 tests unitarios en check_test.go
- [done] Shell completion actualizado con `check`

## Phase 20 — SEO avanzado (done)

Quick wins de alto impacto para mejorar indexacion y visibilidad en buscadores.

### 20A — JSON-LD structured data (schema.org)
- [done] `BlogPosting` schema para paginas (headline, author, datePublished, dateModified, image, description, wordCount)
- [done] `WebSite` schema en index con SearchAction (sitelinks search box)
- [done] `BreadcrumbList` schema en paginas con path hierarchy
- [done] `jsonld` template func en render/funcs.go (genera `<script type="application/ld+json">` como template.HTML)
- [done] 13 tests unitarios en jsonld_test.go

### 20B — Web Vitals optimization
- [done] `defer` para JS no critico (lightbox, interactions, share, comments, tabs, progress)
- [done] `fetchpriority="high"` para hero image / LCP (PictureHTML + picture template func con 4th arg)
- [done] 2 tests nuevos: fetchpriority sin y con variantes

### 20C — RSS por seccion
- [done] `renderSectionFeeds()`: genera atom.xml y rss.xml por seccion no-root
- [done] `<link rel="alternate">` en `<head>` de cada seccion (condicional via section_feeds + section + !is_root)
- [done] Config `section_feeds: true` (default habilitado)
- [done] `sectionFeedContext()` reusa feed_title/feed_description para templates existentes
- [done] Section View incluye `is_root` para gating en templates
- [done] check.go: per-section feed paths en known paths
- [done] 6 tests: renderSectionFeeds (4) + sectionFeedContext (2)
- [done] Dual-file sync: head.html, page.html

## Phase 21 — Mejoras de contenido y tema (done)

Funcionalidades orientadas al lector y a sitios con mucho contenido.

### 21A — Paginacion en index
- [done] Paginacion configurable en homepage (`posts_per_page`, default 10)
- [done] `renderPaginatedIndex()` genera `/`, `/page/2/`, `/page/3/` via `taxonomy.BuildPaginator`
- [done] Template `index.html` con navegacion prev/next y indicador de pagina
- [done] Compatible con featured posts (featured en primera pagina)
- [done] i18n: claves `page_of`, `newer_posts`, `older_posts`
- [done] CSS: `.pagination` component Nord-styled
- [done] 4 tests: no pagination, with pagination, disabled, context

### 21B — Table of Contents flotante (sticky TOC)
- [done] Desktop (>1200px): TOC en sidebar sticky con `position: sticky`, `border-left` accent
- [done] Mobile: TOC colapsable via `<details>/<summary>` (no JS necesario)
- [done] Scroll-spy: `toc-spy.js` con IntersectionObserver, marca heading activo con `.toc-active`
- [done] Grid layout: `.article-content--with-toc` (1fr + 220px) solo en desktop
- [done] Dual renderizado: `.toc-mobile-details` + `.toc-desktop` (CSS show/hide por breakpoint)

### 21C — Busqueda mejorada (done)
- [done] Filtros por fecha (rango), tags y seccion en el plugin de search
- [done] Destacar fragmentos coincidentes (highlight snippets) en resultados
- [done] Ordenar resultados por relevancia vs por fecha (toggle)
- [done] Navegacion con teclado en resultados (arrow keys, Enter)

### 21D — Reading list / Bookmarks
- [done] Boton "guardar para despues" en cada post (localStorage client-side)
- [done] Pagina `/bookmarks/` generada con template `bookmarks.html`
- [done] `bookmarks.js`: save/remove, badge count, client-side rendering, export/import JSON
- [done] CSS: bookmark button, badge, remove button, bookmarks page
- [done] i18n: claves `bookmark_save`, `bookmark_remove`, `bookmarks`, `no_bookmarks`
- [done] Dual-file sync: templates, JS, CSS, i18n

## Phase 22 — DX y experiencia de desarrollo (done)

Mejoras al flujo de trabajo del autor de contenido y del desarrollador de temas/plugins.

### 22A — Hot reload parcial (incremental serve) (done)
- [done] En `osg serve --watch`, detectar que fichero cambio y solo re-renderizar las paginas afectadas
- [done] Si cambia un template, re-renderizar solo las paginas que usan ese template
- [done] Si cambia un fichero Markdown, solo re-renderizar esa pagina + index/section que la listan
- [done] Si cambia un fichero Sass/CSS, solo recompilar assets (no re-renderizar HTML)
- [done] Log del motivo de rebuild: "rebuilding page X (content changed)" vs "full rebuild (template changed)"

### 22B — Dry-run para build (done)
- [done] `osg build --dry-run`: mostrar que ficheros se generarian sin escribir a disco
- [done] Incluir: paginas, feeds, sitemap, assets copiados, imagenes optimizadas
- [done] Formato tabla con ruta de salida y tamano estimado
- [done] Util para verificar permalinks y estructura antes de publicar

### 22C — Preview de fichero unico (done)
- [done] `osg preview <file.md>`: renderizar una sola nota y abrir en browser
- [done] Servidor temporal en puerto aleatorio, auto-cierre tras 5 min de inactividad
- [done] Abre browser automaticamente, CLI con --port y --timeout flags
- [done] PreviewBuild en build/preview.go: mini-build con theme, i18n, assets, una pagina

### 22D — Health dashboard en TUI (done)
- [done] ComputeStats en build/stats.go: parsea contenido y public dir
- [done] Panel con estadisticas del sitio: total posts, drafts, secciones, imagenes, tamano de output
- [done] Desglose por seccion: posts por seccion, posts sin tags, posts sin imagen
- [done] Histograma de publicaciones por mes (sparkline ASCII)
- [done] /stats slash command en TUI
- [done] Accesible via `/stats` slash command en TUI

## Phase 23 — Rendimiento avanzado (done)

Optimizaciones para sitios grandes (100+ posts, muchas imagenes).

### 23A — Incremental builds inteligentes (done)
- [done] Dependency tracking: si un template cambia, solo re-renderizar las paginas que lo usan
- [done] Si solo cambia contenido de una pagina, no re-renderizar secciones/taxonomias no afectadas
- [done] Cache de dependencias template->pagina persistente entre builds
- [done] Log de decisiones de cache: "skipping page X (no changes)" vs "rebuilding page X (template changed)"

### 23B — Lazy image optimization (done)
- [done] SHA-256 hash de imagen fuente vs cache antes de re-procesar
- [done] Solo generar variantes WebP/srcset de imagenes nuevas o modificadas
- [done] Cache persistente en `.osg/cache/images.json`
- [done] Metrica: "optimized N/M images (K cached)"
- [done] Verificacion de que variantes existen en disco antes de cache hit

### 23C — Worker pool acotado para template rendering (done)
- [done] Limitar goroutines de rendering a `runtime.NumCPU()` (evitar exceso en sitios con 500+ paginas)
- [done] Cola de trabajo con canal buffer para backpressure
- [done] Error propagation desde workers

### 23D — Metricas de build exportables (done)
- [done] `osg build --timing=<file.json>`: exportar timing por stage a JSON
- [done] BuildTimings.WriteJSON() con total_ms y stages array
- [done] Historial de builds: `.osg/build-history.json` (max 100 entradas, auto-trim)
- [done] Cada entrada: timestamp, total_ms, rendered, cached, errors, stages

## Phase 24 — Plugins y ecosistema avanzado (done)

Expandir las capacidades del sistema de plugins.

### 24A — Plugin hot-reload
- [done] En `osg serve`, detectar cambios en ficheros `.wasm` del directorio de plugins
- [done] Recargar plugin sin restart del servidor (Close + Load WASM via full rebuild)
- [done] ReloadPlugin method en Manager para recarga individual (uso futuro)
- [done] Emitir `config.validate` tras reload (se emite en cada build)
- [done] Log: plugin reload se registra en los logs de build

### 24B — Nuevos hooks
- [done] `page.before_render`: modificar contexto del template antes de renderizar (inyectar datos custom)
- [done] `feed.transform`: modificar entries del feed antes de serializar (personalizar feeds)
- [done] `sitemap.transform`: modificar entries del sitemap (excluir paginas, cambiar priorities)
- [done] Documentar nuevos hooks en PLUGINS.md

### 24C — Plugin marketplace web
- [done] Pagina estatica auto-generada desde `plugins-index.json`
- [done] Listado con nombre, descripcion, autor, version, hooks soportados
- [done] Filtrado por hook type (content.transform, build.finished, etc.)
- [done] Instrucciones de instalacion inline (`osg plugin install ...`) con click-to-copy
- [done] GenerateMarketplace() en internal/build/marketplace.go
- [done] Hooks metadata agregada a plugins-index.json

## Phase 25 — Integraciones y automatizacion (done)

### 25A — Webhooks y notificaciones
- [done] Hook `after.build` con soporte de webhooks: POST a URL configurable con payload JSON (stats del build)
- [done] Config `webhooks: [{url, events, secret}]` en WebhookConfig
- [done] Eventos: build.success, build.failure, deploy.success
- [done] HMAC signature en header X-OSG-Signature para verificacion

### 25B — Import desde otras plataformas
- [done] `osg import wordpress <export.xml>`: importar posts desde WordPress WXR export
- [done] `osg import hugo <content-dir>`: importar posts desde Hugo (frontmatter TOML/YAML + contenido)
- [done] Mapping de frontmatter: convertir campos especificos de cada plataforma a formato osg
- [done] Preservar fechas, tags, categorias, imagenes referenciadas
- [done] Modo `--dry-run` para previsualizar sin escribir

## Phase 26 — Observabilidad y operaciones (done)

Herramientas para monitorizar el sitio generado y el proceso de build.

### 26A — Analytics lightweight
- [done] Script de analytics propio (sin third-party): pageviews, referrers, browser/OS
- [done] Datos almacenados en SQLite junto con interactions (reutilizar API server)
- [done] Dashboard endpoint GET /api/v1/analytics/summary (JSON para herramientas externas)
- [done] Respetar DNT (Do Not Track) en script y en API handler
- [done] Config `analytics: true` (default deshabilitado)

### 26B — Site audit automatico
- [done] `osg audit`: analizar el sitio generado en public/ para problemas comunes
- [done] Checks: HTML validation (tags abiertos), accesibilidad basica (alt en imagenes, headings order), performance (tamano de paginas > 500KB)
- [done] Reporte con severidad (error/warning/info) y sugerencias de fix
- [done] Integrable en CI como quality gate (exit code 1 si hay errores, --json para parsing)

## Phase 27 — Layout y widgets de portada (done)

Layout de 3 columnas en la portada para widgets laterales.

### 27A — Grid 3 columnas en homepage (done)
- [done] Layout CSS grid en homepage: `|sidebar-left| content |sidebar-right|` con
  tracks fijos de 240px para sidebars y `minmax(0, 1080px)` para el centro;
  `justify-content: center` mantiene la columna central centrada en viewport
- [done] Sidebars opcionales: solo se renderizan cuando `sidebar_widgets` esta
  configurada; `display: contents` en el wrapper deja la portada single-column
  intacta cuando no hay widgets
- [done] Responsive: sidebars `display:none` por debajo del breakpoint 1400px,
  centro vuelve a comportamiento de `.container`
- [done] El bloque central no cambia de ancho ni de posicion al activar widgets
  (centrado en viewport en ambos modos)
- [done] Slot `block "sidebar-right"` con dispatcher inline que itera
  `sidebar_widgets` e incluye partials por nombre

### 27B — Widgets de sidebar (done)
- [done] `partials/widget-author.html`: avatar + nombre + bio + social links
  desde config; se oculta si no hay datos de autor
- [done] `partials/widget-newsletter.html`: formulario `<form action method=post>`
  con campo email y boton submit; se oculta si `newsletter_action` esta vacio
- [done] `partials/widget-popular.html`: top 5 paginas por views desde
  `interactions.db`; nuevo `Store.TopPages(limit)` y `popularPagesView` en
  build resuelve paths a {title, permalink, views}; se oculta sin datos
- [done] Config `sidebar_widgets: [author, newsletter, popular]` con
  normalizacion (trim, lowercase, dedupe, validacion de nombres)
- [done] Config `newsletter_action` (URL del form action)
- [done] i18n keys: widget_about_heading / widget_newsletter_* / widget_popular_heading
  + aria_sidebar_left/right (en + es)
- [done] Sync dual-file: internal/theme/default ↔ themes/default
- [done] Tests: normalisation de sidebar_widgets, trim de newsletter_action

## Phase 28 — Blog UX: copy code, breadcrumbs, author card

Sprint 1 de mejoras basado en analisis de blogs referentes (Julia Evans, Josh Comeau,
Tania Rascia, Kent C. Dodds, Gwern, etc.).

### 28A — Copy code button (done)
- [done] JS (`copy-code.js`): boton "Copy" en cada `<pre><code>` block
- [done] Icono de copia → checkmark durante 2s tras copiar
- [done] CSS: visible solo en hover/focus, accent-2 (verde) en estado "copied"
- [done] Cargado via `<script defer>` en page.html

### 28B — Breadcrumbs en articulos (done)
- [done] En `build.go`: mapa page→section, pass `page_section` al contexto de render
- [done] Template: nav con `Home / Section / Title` (solo si la pagina pertenece a una seccion no-root)
- [done] CSS ya existia (`style.css:1073`), aria-label con traduccion `aria_breadcrumb`
- [done] Mejora SEO y orientacion del lector

### 28C — Author bio card (done)
- [done] Config: `author_bio` y `author_avatar` en `config.go` + `config.yaml.example`
- [done] Template: card al final del articulo (tras related posts) con avatar + bio
- [done] CSS: flexbox, responsive (columna en mobile), Nord palette
- [done] Se oculta automaticamente si ambos campos estan vacios

### 28D — Series/colecciones (done)
- [done] Campo `series` y `series_order` en frontmatter + Page struct
- [done] `pickInt` helper para parsear series_order
- [done] `buildSeriesIndex`: agrupa paginas por serie, ordena por series_order/date
- [done] Template: lista ordenada de la serie, highlight del articulo actual, nav prev/next
- [done] CSS: series-nav con counter automático, item actual resaltado

### 28E — Sidenotes estilo Tufte (done)
- [done] Goldmark footnotes ya habilitado (`extension.Footnote`)
- [done] `sidenotes.js`: convierte footnotes en notas al margen en pantallas anchas (>=1200px)
- [done] No actua si hay TOC sidebar (margen derecho ya ocupado)
- [done] CSS: footnotes styled para mobile, sidenotes para desktop (float right, negative margin)
- [done] Fallback limpio: en mobile se ven como footnotes normales al final

### 28F — Backlinks (done)
- [done] `buildBacklinkIndex`: escanea HTML renderizado buscando `<a href="/...">` internos
- [done] Indice inverso: pagina destino -> paginas que enlazan a ella
- [done] Template: seccion "Enlazado desde" con cards linkables (titulo + summary)
- [done] CSS: backlinks cards con hover accent border
- [done] i18n: keys `backlinks`, `linked_from` en es/en

### 28G — Sprint 3 (done)
- [done] AVIF support en image optimizer (avifenc, `<picture>` AVIF > WebP > JPEG)
- [skip] Newsletter form embed — no newsletter actualmente
- [done] Feed XSLT styling (RSS/Atom legible en browser con Nord theme)
- [done] Prefetch on hover (Speculation Rules API, eagerness: moderate)
- [skip] Font subsetting — Inter ya es latin-only subset (23KB woff2)

## Phase 29 — SEO polish (done)

Segundo sprint de SEO sobre la base de Phase 20. Cubre directivas robots,
señales de crawl budget en sitemap, schema Organization, byline semántico y
reduccion de CSS render-blocking.

### 29A — Robots, canonical, keywords y meta seo (done)
- [done] Frontmatter `osg.robots` (cadena libre, ej. "noindex, nofollow")
- [done] Frontmatter `osg.noindex` (booleano, atajo para "noindex, follow")
- [done] Frontmatter `osg.canonical_url` (override de la URL canonica)
- [done] Frontmatter `osg.keywords` (lista; emite `<meta name="keywords">`)
- [done] head.html: `<meta name="robots">` derivado (page > draft > omitido)
- [done] head.html: canonical override desde frontmatter
- [done] Config `author` y `author_url` (defaults para `<meta name="author">` y Person schema)
- [done] Config `theme_color_light` / `theme_color_dark` con media queries de prefers-color-scheme
- [done] Sitemap excluye paginas con `noindex: true` o `robots` que contiene "noindex"
- [done] Tests: SEO directives y noindex shortcut en site/, robots/canonical en head

### 29B — Sitemap priority + changefreq (done)
- [done] `SitemapEntry` con `Priority` (float64) y `ChangeFreq` (string)
- [done] Defaults por tipo: home 1.0/daily, articulos 0.8/monthly, secciones 0.7/weekly, taxonomias 0.5/weekly
- [done] `builtins/sitemap.xml`: emite `<priority>` y `<changefreq>` solo cuando estan presentes
- [done] Tests: priority/changefreq en sitemapEntryViews + exclusion noindex

### 29C — Organization JSON-LD + Person enriquecido (done)
- [done] Config `organization: {name, url, logo, same_as}` (alias schema.org Organization)
- [done] `buildOrganizationSchema` en funcs.go (logo absoluto, sameAs[], @context solo top-level)
- [done] jsonldFunc emite Organization en home (junto a WebSite)
- [done] BlogPosting publisher usa Organization config cuando esta disponible (siteTitle como fallback)
- [done] BlogPosting author cae a `config.author` + `config.author_url` cuando la pagina no tiene author propio
- [done] Tests: TestBuildOrganizationSchema, TestBuildArticleSchema_AuthorFallbackAndPublisher, TestJsonldFunc_HomeWithOrganization

### 29D — Robots.txt configurable (done)
- [done] Config `robots: {disallow[], crawl_delay, extra}`
- [done] `builtins/robots.txt`: Disallow paths bajo User-agent: \*, Crawl-delay opcional, raw `extra` apendido (multi-User-agent, comentarios)
- [done] Comportamiento por defecto sin cambios (Allow: / + Sitemap)

### 29E — Address tag + critical CSS reduccion (done)
- [done] page.html: byline en `<address class="author" rel="author">` (semantica HTML5 + microformato)
- [done] CSS: reset de `font-style: italic` heredado de `<address>`
- [done] head.html: `syntax.css` y KaTeX cargados con `media="print"` + onload swap (no render-blocking)
- [done] `<noscript>` fallback a `<link rel="stylesheet">` para clientes sin JS
- [done] style.css se mantiene render-blocking (above-the-fold depende de el)

### 29F — Documentacion (done)
- [done] config.yaml.example: secciones SEO, theme-color, organization, robots
- [done] DefaultConfigYAML mirror en config.go
- [done] ROADMAP.md: Phase 29 con subtareas
- [done] TASKS.md: entrada Phase 29

## Phase 30 — Web UI dashboard (`osg ui`) (done)

Interfaz grafica local complementaria al CLI/TUI, cohesionada con el binario
y sin dependencias externas. Loopback-only por defecto; sin auth en v1.
Rationale: ofrecer al autor una vision profesional del estado del Vault, los
plugins y los servicios (serve, api), con orquestacion start/stop desde el
navegador y logs en vivo. El CLI sigue funcionando exactamente igual cuando
no se invoca `osg ui`.

### 30A — Esqueleto del comando y bind loopback (done)
- [done] Subcomando `osg ui` (Kong) con flag `--addr`
- [done] `internal/app/ui.go`: `RunUI` + `normalizeLoopbackAddr` (rechaza
  0.0.0.0/IP publica; `:1314` se normaliza a `127.0.0.1:1314`)
- [done] `config.UI{Addr}` con default `:1314` y normalizacion en `Load`
- [done] Tests unitarios del validador de loopback

### 30B — Dashboard SSR con datos reales (done)
- [done] Paquete `internal/ui/`: server, handlers, templates, assets
- [done] `//go:embed templates assets` (mismo patron que el tema default)
- [done] `html/template` SSR; sin SPA, sin toolchain Node, sin dependencias
- [done] CSS Nord palette + dark/light toggle (consistente con el tema)
- [done] Paginas: `/` (dashboard), `/vault`, `/plugins`, `/services`
- [done] Reutiliza `build.ComputeStats`, `plugin.Manager.Metadata`, `site.ParseFile`

### 30C — Supervisor de servicios (done)
- [done] `Supervisor` ejecuta `serve`/`api` como goroutines (no exec.Cmd)
- [done] Estados idle/starting/running/stopping/error con `LastError`
- [done] Start: ventana de 200ms para detectar fallos inmediatos (puerto
  ocupado) y devolverlos sincronicamente; Stop: cancel + wait con timeout 5s
- [done] StopAll en shutdown del dashboard (sin goroutine leaks)
- [done] Ring buffer de 500 lineas por servicio (io.MultiWriter a stderr)
- [done] Tests: lifecycle Start/Stop/error, StopAll, unknown service

### 30D — Logs en vivo (SSE) (done)
- [done] `ringBuffer.Subscribe(ctx)` con replay de historial + entrega live
- [done] Endpoint `GET /services/{name}/logs` (Go 1.22+ path patterns)
- [done] SSE: `event: log\ndata: <linea>\n\n` + heartbeat cada 15s
- [done] Frontend (vanilla JS): EventSource lazy en `<details>` toggle,
  auto-scroll si esta cerca del fondo, trim a 2000 lineas DOM
- [done] Test: subscribe history replay + live updates

### 30E — Documentacion (done)
- [done] config.yaml.example: seccion `ui:` con `addr`
- [done] DefaultConfigYAML mirror en config.go
- [done] ROADMAP.md: Phase 30

### 30F — Pulido v1 (done)
- [done] Auto-refresh de `/services` con polling JSON cada 2s (uptime live sin F5)
- [done] Endpoint GET /services.json (state, started_at, uptime, last_error)
- [done] Favicon SVG embebido + redirect /favicon.ico → /assets/favicon.svg
- [done] Filtro/busqueda en `/vault` (substring sobre title+path+section)
- [done] UI para enable/disable de plugins persistiendo en config.yaml (usa `config.UpdatePluginsEnabled` que preserva comentarios)
- [done] State.Plugins unificado (loaded + on-disk + enabled-but-missing) con badges enabled/disabled/missing

### 30H — Pulido v2 (done)
- [done] Audit log persistente del scheduler en `.osg/scheduler.db`
  (SQLite via `modernc.org/sqlite`, paquete `internal/scheduler`,
  pagina `/scheduler` con tabla de runs, columnas due_at/ran_at/status/error)
- [done] Auto-reload de servicios al togglear plugins:
  `Supervisor.Restart` reinicia serve/watcher/scheduler que estuviesen
  en estado running tras `UpdatePluginsEnabled`
- [done] Boton "Rebuild now" en `/assets`:
  POST `/rebuild` dispara `RunBuild` en goroutine (rebuilder con mutex,
  uno a la vez), `/rebuild.json` para polling, JS actualiza estado del
  boton + status mono
- [done] HTMX 2.0.4 vendorado en `internal/ui/assets/htmx.min.js` y
  cargado en el layout. No migro interacciones existentes (funcionan);
  queda disponible para fragments parciales futuros

### 30G — Capacidades futuras (done)
- [done] Watcher integrado en `osg ui` como tercer servicio del supervisor
  (reusa `startWatch`/`runWatchLoop` sin reload hub; rebuild automatico
  con debounce; logs en el ring buffer del servicio)
- [done] Scheduler interno para publish-on-date:
  - Frontmatter `osg.publish_at` (RFC3339 o `YYYY-MM-DD`); reusa `date.Parse`
  - `Page.IsScheduled()` y filtro en build (saltado salvo `IncludeDrafts`)
  - `SiteStats.Scheduled` y `SiteStats.NextScheduled`; card en dashboard
  - Servicio `scheduler` en el supervisor: duerme hasta el siguiente
    `publish_at` (clamp 5min) y dispara `RunBuild` cuando vence
  - Stateless (no DB): re-escanea frontmatter en cada iteracion
- [done] Asset management UI (read-only en v1):
  - Inventario de imagenes en content_dir y static_dir
  - Cards: total, total size, formats; tabla por formato (count+size)
  - Tabla de ficheros con filtro client-side (reusa `setupVaultFilter`)
  - Mutaciones (regenerar variants, convertir formato) deferred:
    `osg build` ya optimiza, y watcher/scheduler lo disparan

## Phase 31 — Web UI operativa: tasks + history (done)

`osg ui` deja de ser solo observable y pasa a ejecutar todos los comandos
del CLI desde el navegador. Audit log persistente y panel lateral de
inspeccion al estilo Linear/Kestra.

### 31A — Operations runner unificado (done)
- [done] `internal/operations.Runner` generaliza `Supervisor` (servicios
  long-running) y el rebuilder one-shot detras de la misma API:
  `Trigger`/`Stop`/`Snapshot`/`Logs`/`History`. Concurrencia por nombre
  (un build serializa con build, check + audit corren en paralelo)
- [done] `Definition{Name, Kind, Description, Run RunFunc}` con `Kind`
  ∈ {`service`, `task`} y closures inyectados desde `app/ui.go` para
  reusar `RunBuild`, `RunDeploy`, `RunCheck`, etc.
- [done] Ring buffer 500 lineas por run con `Subscribe(ctx)` para SSE
- [done] `internal/operations.Store` (SQLite via `modernc.org/sqlite`,
  WAL) con tabla `operations_runs` (id, name, kind, params JSON,
  started_at, ended_at, status, error). `Begin`/`Finish`/`Recent`/
  `MarkInterruptedRunning` para shutdown sin runs colgando
- [done] Migracion automatica del legado `.osg/scheduler.db`:
  bootstrap copia rows al esquema nuevo y deja `.osg/scheduler.db.bak`

### 31B — Tasks one-shot (done)
- [done] Definitions registradas: `init`, `update-content`, `build`,
  `deploy`, `check`, `audit`, `new`, `theme-init`, `plugin-install`,
  `import-wordpress`, `import-hugo`. Cada una cierra sobre los closures
  de `app.Run*` que ya existen
- [done] `POST /operations/{name}/run` (303 a Referer; JSON cuando se
  pide por Accept). `POST /operations/{name}/stop` cancela contexto
- [done] `POST /operations/{name}/run-flow` ejecuta secuencia
  `init → update-content → check → build → deploy` desde la operacion
  pulsada en adelante (aborta al primer error)
- [done] `GET /operations/{name}/logs` SSE con replay del ring buffer
  + heartbeat 15s

### 31C — UI: /actions, /history, drawer (done)
- [done] `/actions` como pipeline horizontal (5 nodos init→deploy)
  con flechas, status pill por estado, botones "Run" y "Run from here →"
- [done] `/history` tabla cronologica con filtros (name/kind/status),
  pills coloreadas (ok/error/cancelled/running), duracion mono
- [done] Drawer lateral derecho con tabs Output/Params/Details: log
  en vivo via SSE para active runs, snapshot estatico para finalizadas,
  Re-run/Stop en footer. HTMX `hx-get → hx-target=#drawer`
- [done] Iconos sprite SVG (`internal/ui/assets/icons.svg`) con `<symbol>`
  por operacion (build=hammer, deploy=cloud-upload, check=clipboard,
  audit=search, new=plus, theme=palette, etc.)

### 31D — Parameter forms + confirmacion (done)
- [done] `operationParamRegistry` (mapa name → []ParamDef) define
  campos bool/string/select por operacion. Plantilla `op-field.html`
  renderiza el control correspondiente; `paramsFromForm` coerce a
  map[string]any que llega al RunFunc
- [done] Modal de confirmacion (`<dialog>` nativo) para Deploy y los
  importadores via `data-confirm` en el form
- [done] Forms colapsables (`<details>` cerrado por defecto) en
  /vault, /plugins, /themes, /import para que la pagina principal sea
  el inventario y el formulario quede oculto hasta que se necesita

### 31E — Pipeline view + reorganizacion (done)
- [done] /actions deja de ser un grid de tareas; pasa a ser la
  visualizacion del flow canonico (`actionFlow = [init, update-content,
  check, build, deploy]`)
- [done] Tasks domain-specific movidas a su pagina natural: `new` en
  /vault, `plugin-install` en /plugins, `theme-init` en /themes,
  `import-{wordpress,hugo}` en /import. `audit` con su propia pagina
  y tabla de findings (severity/category/file/message/fix)
- [done] Quick-action banner en /dashboard con cards Build/Deploy
  (mismo partial `quick-button.html`, mismas pills + Stop)
- [done] Boton "Inspect" en cada card abre el drawer con la run
  activa o la ultima

### 31F — Pulido v3 (done)
- [done] `Runner.lastLogs` retiene el tail de la ultima ejecucion
  finalizada en memoria (por nombre, sustituido en cada finish), de
  forma que el drawer muestre el log de la run anterior tras volver
  a idle. Logs no se persisten en disco (in-memory only)
- [done] `drawerViewForHistory` cruza el row solicitado contra
  `Runner.Snapshot()`: si coincide con `Active.ID` se streamea, si
  coincide con `LastRun.ID` se muestra el tail capturado, runs mas
  antiguas siguen mostrando "No log data captured"
- [done] Card swap completo en transiciones de estado: cuatro
  partials (`flow-node`, `op-card`, `quick-button`, `task-form`)
  llevan `data-card-style` + `data-state`; el poller compara contra
  `/operations.json` y, cuando el estado cambia, hace fetch a
  `/operations/{name}/card?style=...` y `replaceWith` para refrescar
  meta line + boton Run/Stop sin recargar la pagina
- [done] Drop del bloque "Services placeholder" del dashboard
  (cubierto por la entrada Services del nav)

## Phase 32 — Image pipeline: encoders embebidos (done)

Eliminada la dependencia externa de las CLIs `cwebp` y `avifenc`. La
optimizacion de imagenes ya no requiere instalar nada en el sistema.

### 32A — Encoders WebP/AVIF embebidos (done)
- [done] `github.com/gen2brain/webp` v0.5.5 (libwebp compilado a WASM,
  ejecutado via wazero — el mismo runtime que ya usamos para plugins)
- [done] `github.com/gen2brain/avif` v0.4.4 (libavif analogo)
- [done] `writeWebP`/`writeAVIF` toman `image.Image` directamente,
  sin pasos intermedios JPEG en disco. Bucle por width re-encodea
  desde la imagen redimensionada en RAM
- [done] `Options.WebP`/`Options.AVIF` siguen siendo kill switches
  para el operador, pero ya no estan gateados por `exec.LookPath`
- [done] Tests sin skips por CLI ausente (encoders siempre disponibles)

### 32B — Variants y calidad del original (done)
- [done] Dropped JPEG variants en `<picture>`: cualquier navegador
  que entiende `<picture>` entiende WebP, asi que las srcset de JPEG
  eran codigo muerto en la practica
- [done] El original (downsizeado a max width) se guarda con
  `quality - 5` (clamp 50): solo lo consumen navegadores antiguos,
  RSS readers y crawlers — no compensa el ancho de banda extra

### 32C — Downsize del original PNG (done)
- [done] `downsizeIfNeeded` acepta `.png` ademas de jpg/jpeg/webp:
  capturas de pantalla y diagramas vectorizados se quedaban con
  dimensiones gigantes en `public/` aunque las variantes responsive
  ya se generaban
- [done] Re-encode lossless con `png.BestCompression` (PNG no tiene
  knob de calidad). Mantiene la extension `.png` para no romper
  referencias en el HTML

## Phase 33 — Resilencia y mantenimiento de resumenes IA (done)

Cuando la generacion de resumen via IA falla (timeouts, 5xx, 429) la
build cae al modo extractivo. Esta fase añade reintentos automaticos
para fallos transitorios y una accion manual para regenerar resumenes
desde la UI sin tener que tocar el cache a mano.

### 33A — Reintentos en kairos.Summarize (done)
- [done] `chatWithRetry` envuelve `llm.Provider.Chat` con 3 intentos
  totales y backoff exponencial 250ms / 500ms. El context del caller
  se respeta entre intentos (cancel inmediato)
- [done] `isRetryable` clasifica errores: 4xx-style (401/403/400/404,
  "invalid api key", "permission denied", "unauthorized", "forbidden")
  fallan rapido; resto (timeouts, 5xx, 429, errores de red) reintenta
- [done] Logging via `slog.Default()` para reportar cada reintento

### 33B — Invalidacion de cache de resumen desde la UI (done)
- [done] `AICache.Remove(hash)` y helper publico
  `build.InvalidateAISummary(cfg, hash, logger)` para borrar la entrada
  con un load → drop → save
- [done] `POST /summary/invalidate` con sanitize del path (Clean +
  relativity check para evitar escape de `content_dir`); 303 a
  Referer; tras borrar, la siguiente build re-llama al LLM
- [done] Boton "re-summarize" por fila en `/vault` con `data-confirm`
  (modal de confirmacion ya existente). `PageView.Source` (path
  relativo a content_dir) sirve como identificador estable

### 33C — Editor de resumen + chained build/deploy (done)
- [done] `internal/frontmatter.UpdateField`: reemplaza/elimina un
  dotted key (`osg.summary`) preservando el resto del frontmatter
  YAML mediante `yaml.Node` (orden, comentarios, body intactos);
  soporta archivos sin frontmatter (lo crea); estilo de bloque
  literal para summaries multilinea
- [done] `site.ParseFile` reconoce `osg.summary` como alias del
  ya existente `osg.abstract` para que la UI hable el mismo nombre
  que escribe la persistencia
- [done] `build.LookupAISummary` (read-only) para que la UI muestre
  como referencia el ultimo summary generado por IA
- [done] Pagina `/vault/page?source=…` con metadata read-only,
  bloque "currently used" (resumen efectivo), bloque "AI cache"
  cuando difiere, y textarea sobre `osg.summary` con tres botones:
  Save / Save &amp; rebuild / Save, build &amp; deploy
- [done] `POST /summary/save` reescribe el `.md` atomicamente
  (tmpfile + rename), opcionalmente dispara `build` o
  `build → deploy` via `Runner.RunFlow` y redirige a `/actions` para
  que el flow drawer muestre el progreso
- [done] El boton "Save, build &amp; deploy" lleva el `data-confirm`
  modal existente; deploy implica build automaticamente
- [done] Link "edit" por fila en `/vault` apunta al editor

### 33D — Migracion del cache de resumenes a SQLite (done)
- [done] `internal/build/summary_store.go`: tabla
  `summaries(hash PK, summary, provider, model, created_at)`
  en `.osg/cache/summaries.db` (modernc.org/sqlite + WAL, mismo
  patron que `internal/operations.Store`)
- [done] La build sigue cargando todo a un map en memoria al
  arrancar (1 SELECT) y haciendo upsert por fila al terminar
- [done] Operaciones de un solo registro en la UI
  (`LookupAISummary`, `InvalidateAISummary`,
  `UpsertAISummary`) van directas al store sin cargar el map
- [done] Sin migracion: el JSON queda en disco como reliquia, el
  nuevo codigo lo ignora (el usuario regenera bajo demanda)
- [done] Tests reescritos para SQLite (round-trip, store vacio,
  dirs anidados, nil cache, invalidate, lookup)

### 33E — Lista de pages como cards + acciones in situ (done)
- [done] `/vault` deja de ser tabla y pasa a ser lista de cards
  (`<details>` por fila): header con titulo + section + fecha +
  status pill + pill semantica del summary
  (`override` / `ai-cached` / `no summary`)
- [done] Body desplegable con bloques "Override (osg.summary)" y
  "AI cache" (los que aplican) y un footer con tres acciones:
  **Editar** abre `/vault/page`, **Resumir** llama al LLM ahora
  mismo y guarda en `summaries.db`, **Eliminar** dropea la
  entrada del cache
- [done] `POST /summary/regenerate`: kairos provider one-shot,
  honra `cfg.AI.Timeout`, persiste via
  `build.UpsertAISummary` y redirige a `/vault`
- [done] Boton "Sugerencia IA" en `/vault/page`: `POST
  /summary/suggest` devuelve `{suggestion}` en JSON y la JS
  rellena la textarea sin guardar (el usuario revisa y pulsa
  Save). Boton se deshabilita mientras la llamada esta en vuelo
- [done] Iconos nuevos en el sprite: `icon-edit`, `icon-trash`,
  `icon-sparkles`
- [done] `setupVaultFilter` ahora cae a `[data-search]` cuando
  no hay `tbody tr` (sirve cards igual que la tabla anterior)
- [done] `PageView` extendido con `Summary` / `HasOverride` /
  `AICached` / `HasAICached`; `state.collectPages` carga el
  cache una sola vez via `build.LoadAISummaries`

## Phase 34 — Deferred publications: drafts y scheduled fuera del deploy (done)

Con `include_drafts: true` el build procesa drafts y posts con
`publish_at` futuro (ideal para pre-cachear resumenes IA, optimizar
imagenes, validar HTML local) pero acababa subiendolos al hosting.
Esta fase desacopla "render local" de "publicar": el deploy lee la
lista de paths diferidos del build state y los excluye antes de
invocar al provider.

### 34A — build_state.db con `deferred_publications` (done)
- [done] Nueva DB SQLite (`.osg/cache/build_state.db`, modernc/sqlite
  + WAL, mismo patron que summaries.db) con la tabla
  `deferred_publications(path PK, source, reason, publish_at,
  recorded_at)`
- [done] `BuildStateStore`: `OpenBuildStateStore`, `LoadDeferred`,
  `ReplaceDeferred`, `Close`. La build llama `ReplaceDeferred` al
  final dejando el set siempre alineado con la ultima ejecucion
- [done] Helper publico `build.LoadDeferredPaths(cfg, logger)` para
  el deploy: leer-y-cerrar, devuelve `nil` ante cualquier error
  (un DB roto no bloquea el deploy — vuelve al comportamiento previo)
- [done] `recordDeferredPublications` en build.go recorre el site
  index post-render: drafts → `reason="draft"`; scheduled futuros
  (`page.IsScheduled()`) → `reason="scheduled"` con `publish_at`

### 34B — deploy staging via hardlinks (done)
- [done] `internal/deploy/staging.go`: `Stage(publicDir, excludes,
  logger)` devuelve `*Staging{Dir, Cleanup, Excluded}`
- [done] Sin exclusiones → devuelve `publicDir` verbatim y un
  `Cleanup` no-op (overhead cero en producción)
- [done] Con exclusiones → crea `.osg-deploy-staging/` junto a
  `public/`, recorre el arbol con `os.Link` (hardlink: zero copy,
  instantaneo) saltando los subarboles excluidos. Fallback a
  `copyFile` en cross-device (EXDEV)
- [done] `RunDeploy` en `app/deploy.go` llama a `Stage` antes de
  invocar al provider y pasa el dir resultante. Imprime al log las
  rutas excluidas para que el usuario vea exactamente que no se
  publica
- [done] Defensa contra typos: una entrada `"/"` se ignora (jamas
  vacia el deploy). Entries vacias o whitespace-only tambien
- [done] Tests: round-trip del store, replace que limpia rows
  previas, replace vacio que vacia tabla, `LoadDeferredPaths` con
  store ausente; staging que excluye subarboles, mantiene siblings,
  cleanup elimina el dir

