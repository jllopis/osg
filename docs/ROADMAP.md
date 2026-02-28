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

## Phase 6 - producto y DX (todo)
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

## Backlog (deferred)
- [done] i18n en templates
- (vacio)
