# TASKS

Formato: [todo|doing|done] Tarea

[done] Definir schema de config (YAML) y defaults
[done] Definir schema de frontmatter de salida
[done] Implementar lectura de vault y discovery de Markdown
[done] Implementar parse YAML frontmatter + split body
[done] Implementar filtro publish + include-drafts
[done] Implementar derivacion de slug y fecha
[done] Implementar layout writer a content/{YYYY/MM/DD}/{slug}
[done] Implementar CLI (kong) con init/update-content/build
[done] Implementar logging estructurado + verbose
[done] Implementar dry-run
[done] Tests unitarios (parser, filtro, mapper, layout)
[done] Documentacion basica de uso
[done] Especificacion de templates (contexto, filtros, funciones, resolucion)
[done] Especificacion de taxonomias (objetos, rutas, plantillas, flujo)

[done] (Phase 2) Indexado de contenido para build
[done] (Phase 2) Templates base (index/section/page) + overrides basicos
[done] (Phase 2) Render HTML a public/
[done] (Phase 3) Taxonomias + paginacion
[done] (Phase 3) Feeds por taxonomia
[done] (Phase 3) Sitemap (split index si aplica)
[done] (Phase 3) robots.txt + 404
[done] (Phase 3) load_data + helpers de templates
[done] (Phase 4) Sass + assets
[done] (Phase 4) Static copy + cachebust (get_url + get_hash)
[done] (Phase 5) WASM plugins + TUI avanzada
[done] Tema base por defecto (templates + CSS)
[done] (Phase 6) Plugin WASM de ejemplo (RSS feed en Rust)
[done] (Phase 6) Live reload + watch (serve + build incremental)
[done] (Phase 6) Build incremental con cache
[done] (Phase 6) Search index opcional (via plugin)
[done] (Phase 6) Starter kit de theme (scaffold + docs)
[done] (Phase 6) SDK/CLI para plugins (plantillas + comandos)
[done] (Phase 6) Tests del SDK/CLI de plugins (scaffold + comandos)
[done] (Phase 6) Sample site + quickstart

[done] (Next) Validacion de config (paths invalidos, taxonomias mal definidas, base_url vacia en prod)
[done] (Next) Limpieza de public/ para evitar archivos stale en builds incrementales
[done] (Next) Estado/diagnostico (`osg doctor` o `osg status`)
[done] (Next) TUI: vista guiada con progreso y estado (no solo logs) -> Phase 9

[done] (Phase 9) Plan detallado en docs/PLAN_TUI_REDESIGN.md
[done] (Phase 9) Rewrite TUI: god file -> 12 modulos enfocados
[done] (Phase 9) Layout 2 paneles: viewport scrollable + sidebar colapsable
[done] (Phase 9) Header compacto, slash commands, comandos bare, paleta Nord
[done] (Phase 9) Sidebar colapsable, barra hints, fix version
[done] (Phase 9) Eliminar prefix-key, ASCII banner, fake progress, codigo muerto

[done] (Summary) Package internal/summary/: Provider interface, ExtractProvider, NoopProvider
[done] (Summary) PlainText() markdown stripper (6 regexes RE2-safe)
[done] (Summary) truncateSentence() con corte en oracion/palabra (max 160 chars)
[done] (Summary) Tres estrategias via summary_strategy: auto, manual, ai
[done] (Summary) Integracion en build.go: fillSummaries() despues de BuildHierarchy()
[done] (Summary) 37 tests unitarios
[done] (Summary) Featured overlay CSS: gradiente, texto blanco, frosted glass label

[done] (Phase 10) Step 1: docs sync (ROADMAP, TASKS, DESIGN)
[done] (Phase 10) Step 2: global site RSS/Atom feed
[done] (Phase 10) Step 3: doctor improvements (diagnosticos accionables, mas checks)
[done] (Phase 10) Step 4: theme polish (tipografia, spacing, responsive, dark mode)
[done] (Phase 10) Step 5: TUI + build tests (cobertura internal/build/ e internal/tui/)
[done] (Phase 10) Step 6: image optimization (WebP, srcset, <picture>)

[done] (Standalone Pages) `osg.path` en frontmatter para override de content_layout
[done] (Standalone Pages) `osg.menu` en frontmatter para marcar paginas de navegacion
[done] (Standalone Pages) Exclusion de menu pages del listado de homepage/secciones
[done] (Standalone Pages) `menu_pages` en contexto global de templates
[done] (Standalone Pages) header.html: renderizar menu_pages junto a taxonomias
[done] (Standalone Pages) Tests: publish (GetOSGString/GetOSGBool), content (menu), site (6 tests)
[done] (Standalone Pages) Documentacion: DESIGN, TEMPLATES, THEMES, Funcional, TASKS, ROADMAP
[done] (Standalone Pages) Ejemplo: about/index.md en sample-site

[done] (osg new) RunNew() en internal/app/new.go: crea nota en vault con frontmatter Obsidian-native
[done] (osg new) Opciones: --tags, --publish (default: draft), --dry-run, --vault-path override
[done] (osg new) CLI: `osg new <title>` via Kong (cmd/osg/main.go)
[done] (osg new) TUI: `/new <title>` slash command (commands.go, model.go, update.go, app/tui.go)
[done] (osg new) 9 tests unitarios (new_test.go) + 2 tests TUI command parsing (commands_test.go)
[done] (osg new) Documentacion: ROADMAP, TASKS, DESIGN, Funcional, AGENTS.md
[done] (osg new) Bloque osg expandido: publish activo + placeholders comentados (title, image, featured, path, permalink, menu, abstract, author)
[done] (osg new) Frontmatter manual (sin yaml.Marshal) para soportar comentarios YAML en placeholders
[done] (osg new) yamlScalar() helper para quoting seguro de valores YAML
[done] (osg new) Auto-apertura de editor: default_editor config > $EDITOR env > skip silencioso
[done] (osg new) --editor/--no-editor flag (negatable) con auto-detect por defecto
[done] (osg new) resolveEditor() y openEditor() en new.go; errores no-fatales
[done] (osg new) DefaultEditor field en Config + seccion "New Post" en ConfigSchema()
[done] (osg new) new_notes_dir config: carpeta destino dentro del vault para notas nuevas
[done] (osg new) --notes-dir CLI override (prioridad: CLI > config > vault root)
[done] (osg new) auto-creacion de directorio destino via MkdirAll
[done] (osg new) 30 tests unitarios (9 originales + 21 nuevos)

[done] (i18n) Package internal/i18n/: Bundle struct, New(), LoadDir(), Trans(), DateFormat()
[done] (i18n) Ficheros de traduccion en.yaml y es.yaml (~31 claves cada uno)
[done] (i18n) Config: default_language field (default "es"), validacion, normalizacion
[done] (i18n) render/funcs.go: transFunc closure sobre Bundle, dateFormatFunc con meses localizados
[done] (i18n) build.go: carga i18n (tema -> usuario), wiring a render.Context, lang en todos los contextos
[done] (i18n) 10 plantillas del tema actualizadas con {{ trans }} y {{ date_format }}
[done] (i18n) Builtins actualizados: 404.html (trans), rss.xml (trans)
[done] (i18n) Dual-file sync: templates y YAML en internal/theme/default/ y themes/default/
[done] (i18n) 14 tests unitarios para i18n package
[done] (i18n) Documentacion: ROADMAP, TASKS, DESIGN, Funcional, AGENTS.md

[done] (Kairos AI) KairosProvider en internal/summary/kairos.go wrapping Kairos llm.Provider
[done] (Kairos AI) Summarize() con PlainText() strip, system+user messages, temperature 0.3
[done] (Kairos AI) NewKairosProvider() factory: gemini, anthropic, openai, qwen, ollama
[done] (Kairos AI) AIConfig en config.go: provider, model, api_key, base_url, system_prompt, timeout, concurrency
[done] (Kairos AI) Defaults y validacion: gemini provider, 30s timeout, 3 concurrency
[done] (Kairos AI) DefaultConfigYAML() con seccion AI completa y doc de todos los providers
[done] (Kairos AI) fillSummaries() reescrito en build.go: AI path con bounded concurrency
[done] (Kairos AI) fillWithAI(): goroutines, per-request timeout, semaphore channel
[done] (Kairos AI) Fallback graceful: fallo de AI provider cae a auto con warning
[done] (Kairos AI) go.mod: 5 require + 5 replace directives para Kairos local
[done] (Kairos AI) 20 tests unitarios (mock providers, factory, concurrency, context cancellation)
[done] (Kairos AI) Build y test end-to-end verificados
[done] (Kairos AI) Documentacion: ROADMAP, TASKS, DESIGN, Funcional, AGENTS.md

[done] (AI Cache) AI summary cache: `.osg/cache/ai-summaries.json`, SHA-256 content hash key
[done] (AI Cache) AICache struct thread-safe con load/save JSON, lookup/store
[done] (AI Cache) fillWithAI() checks cache before LLM, stores results back
[done] (AI Cache) `--force-ai-summaries` CLI flag con confirmacion interactiva
[done] (AI Cache) `--yes`/`-y` flag para bypass confirmacion (CI/scripts)
[done] (AI Cache) 14 tests unitarios para ai_cache.go
[done] (Language) buildDefaultPrompt(lang) inyecta idioma en system prompt
[done] (Language) Language field en AIConfig y KairosProvider, wired desde default_language
[done] (Language) langDisplayName(): BCP-47 -> nombres en ingles
[done] (Language) Custom system_prompt ignora inyeccion de idioma
[done] (Language) 10 tests unitarios para language-aware prompts
[done] (Serve) Serve isolation: opts.SkipAI=true en RunServe(), fallback a auto strategy
[done] (Serve) BuildOptions struct en build.go con SkipAI y ForceAISummaries
[done] (AI Cache) Documentacion: ROADMAP, TASKS, DESIGN, Funcional, AGENTS.md

[done] (Phase 11-A) Mover search plugin a plugins-src/search/, feed a ejemplo de referencia
[done] (Phase 11-A) Actualizar Makefile: plugins desde plugins-src/, target install-plugins
[done] (Phase 11-A) Embeber search.wasm en binario con EnsureBundledPlugins() (//go:embed)
[done] (Phase 11-A) Habilitar search por defecto, link en header.html, claves i18n nav.search
[done] (Phase 11-A) Limpiar examples/plugins/, actualizar README y .gitignore
[done] (Phase 11-B) Tests unitarios para manager.go (Load, Emit, Call, Merge) + fix WASI mount
[done] (Phase 11-B) Timeouts por plugin call (PluginTimeout config, context.WithTimeout)
[done] (Phase 11-B) Ejecucion paralela de plugins (WaitGroup, merge determinista)
[done] (Phase 11-B) Plugin metadata (export plugin_info, PluginMeta, Metadata(), osg plugin list)
[done] (Phase 11-C) Hook config.validate (post-config, errores detienen build)
[done] (Phase 11-C) Hook content.transform (modifica Markdown pre-render)
[done] (Phase 11-C) Hook image.process (transformacion imagenes via WASI)
[done] (Phase 11-C) Hook after.build (post-build garantizado)
[done] (Phase 11-D) Package osg-plugin-sdk-go: Event/Response/PluginMeta types, Plugin struct, On() handlers, ABI helpers
[done] (Phase 11-D) 17 tests unitarios para SDK Go (handler dispatch, plugin_info, helpers, edge cases)
[done] (Phase 11-D) Scaffold TinyGo: osg plugin init --lang=go (main.go con wasmexport, go.mod, build.sh, README)
[done] (Phase 11-D) Actualizar scaffold Rust con plugin_info, bytes_to_wasm, doc de 10 hooks
[done] (Phase 11-D) CLI --lang flag en PluginInitCmd (default rust), TUI /plugin init <name> [dir] [lang]
[done] (Phase 11-D) Fix embed: go.mod.tmpl renaming, .tmpl extension stripping, //go:build ignore en template
[done] (Phase 11-D) 12 tests scaffold: Go+Rust content, .tmpl stripping, tinygo alias, default lang, naming, errors
[done] (Phase 11-E) Instalacion desde GitHub: osg plugin install github.com/user/repo[@tag]
[done] (Phase 11-E) Deteccion automatica de GitHub refs, GitHub Releases API, GITHUB_TOKEN
[done] (Phase 11-E) Indice curado: plugins-index.json en repo, osg plugin search [query]
[done] (Phase 11-E) Lock file: .osg/plugins.lock.json con source + version por plugin
[done] (Phase 11-E) Comando update: osg plugin update [name], check contra latest release
[done] (Phase 11-E) TUI: /plugin search [query] y /plugin update [name]
[done] (Phase 11-E) 15 tests: GitHub refs, lock file CRUD, index search, download, mock server
[done] (Phase 11-F) Actualizar docs/PLUGINS.md: SDK Go, registry, GitHub install, search, update
[done] (Phase 11-F) Templates ya incluyen link /search/ en header e i18n (desde Fase A)
[done] (Phase 11-F) ROADMAP y TASKS sincronizados con Phase 11 completo

[done] (Lightbox) Custom Goldmark renderer: figure[data-lightbox] con figcaption para imagenes standalone
[done] (Lightbox) Lightbox JS: overlay fullscreen Nord, nav teclado/touch, captions, counter
[done] (Lightbox) Galeria automatica: figures consecutivas en CSS grid responsive
[done] (Lightbox) Config lightbox: true (default habilitado), JS condicional en page.html
[done] (Lightbox) CSS: overlay, botones, transiciones, responsive, prefers-reduced-motion
[done] (Lightbox) 10 tests unitarios figure rendering + test paragrafos normales
[done] (Lightbox) Dual-file sync: CSS, JS, templates en internal/theme/ y themes/default/
[done] (Lightbox) Documentacion: ROADMAP, TASKS, DESIGN, AGENTS.md

[done] (Phase 12A) LICENSE file Apache 2.0
[done] (Phase 12A) .editorconfig
[done] (Phase 12A) .golangci.yml
[done] (Phase 12A) SEO: canonical URL en head.html
[done] (Phase 12A) SEO: meta description con page.summary (fallback site_description)
[done] (Phase 12A) SEO: Twitter Card meta tags
[done] (Phase 12A) SEO: OG tags en todos los templates (index, section, taxonomy)
[done] (Phase 12A) SEO: og:site_name, og:locale, og:type article vs website
[done] (Phase 12A) SEO: article:published_time y article:modified_time
[done] (Phase 12A) Sass: --style compressed para CSS minificado
[done] (Phase 12A) Goldmark: heading IDs automaticos (AutoHeadingID)
[done] (Phase 12A) Goldmark: extension Footnote
[done] (Phase 12A) Font preload: Inter y JetBrains Mono (woff2)
[done] (Phase 12A) Dual-file sync: head.html

[done] (Phase 12B) GitHub Actions CI/CD pipeline (test, build, lint, vet)
[done] (Phase 12B) README profesional (features, install, usage, config, theme, plugins)
[done] (Phase 12B) Shell completions: osg completion bash|zsh|fish
[done] (Phase 12B) .goreleaser.yml para releases multi-plataforma
[done] (Phase 12B) Related posts: scoring por terms compartidos, grid en page.html
[done] (Phase 12B) Prev/next navigation cronologica (excluye menu pages)
[done] (Phase 12B) Reading progress bar (JS scroll-based, CSS accent)
[done] (Phase 12B) i18n: claves newer_post, older_post, related_posts
[done] (Phase 12B) page.html: prev/next nav, related posts, progress bar
[done] (Phase 12B) CSS: post-nav, related-card, reading-progress-bar, responsive
[done] (Phase 12B) Dual-file sync: templates, i18n, CSS, JS
[done] (Phase 12B) 6 tests unitarios relatedPages()

[done] (Phase 12C) HTML/CSS/JS/JSON/SVG/XML minification (tdewolff/minify/v2, batch in-place)
[done] (Phase 12C) Config minify: true (default habilitado), campo Minify en Config
[done] (Phase 12C) 5 tests unitarios minificacion
[done] (Phase 12C) Table of Contents: ExtractTOC() regex h2-h6, TOCView(), partial toc.html
[done] (Phase 12C) 7 tests unitarios TOC
[done] (Phase 12C) Syntax highlighting: goldmark-highlighting/v2, Chroma Nord, CSS classes
[done] (Phase 12C) css/syntax.css con colores Nord para tokens
[done] (Phase 12C) Shortcodes: note, warning, tip (admonitions), details (collapsible)
[done] (Phase 12C) Per-name compiled regexes (Go regexp no soporta backreferences)
[done] (Phase 12C) 8 tests unitarios shortcodes
[done] (Phase 12C) CSS: estilos TOC y admonitions con colores Nord
[done] (Phase 12C) i18n: claves toc_title, toc_label (en + es)
[done] (Phase 12C) Dual-file sync: templates, i18n, CSS
[done] (Theme System) theme.yaml metadata: name, description, author, version, parent
[done] (Theme System) ThemeMeta struct, LoadMeta(), WriteMeta() en internal/theme/meta.go
[done] (Theme System) ResolveChain(): parent chain resolution con cycle detection
[done] (Theme System) ListThemes(): escanea themes dir y retorna metadata
[done] (Theme System) TemplateLoader.ThemeChain: carga templates de root ancestor a child
[done] (Theme System) assets.PrepareWithChain(): static y sass desde chain completa
[done] (Theme System) i18n loading desde chain (ancestor primero, child sobreescribe)
[done] (Theme System) Block-based templates: page-header, page-content, index-posts, etc.
[done] (Theme System) ScaffoldChildTheme(): osg theme init --parent (tema minimal con herencia)
[done] (Theme System) osg theme list (CLI + TUI)
[done] (Theme System) Doctor: checkThemeMeta (theme.yaml, parent chain validation)
[done] (Theme System) render.NewWithChain() para usar cadena de herencia
[done] (Theme System) 19 tests unitarios (meta, chain, scaffold, list, cycle, edge cases)
[done] (Theme System) Dual-file sync: theme.yaml, templates
[done] (Theme System) Documentacion THEMES.md actualizada

[done] (Phase 13) Flag --drafts en osg serve (ServeCmd.Drafts, wiring a IncludeDrafts)
[done] (Phase 13) Banner visual en paginas draft (draft-banner en page.html)
[done] (Phase 13) Badge draft en listados (card.html, index.html featured + post-item)
[done] (Phase 13) Excluir drafts de feeds RSS/Atom (feedPages filtra page.Draft)
[done] (Phase 13) Excluir drafts de sitemap (collectSitemapEntries filtra page.Draft)
[done] (Phase 13) CSS: draft-banner, draft-badge, draft-badge--featured (Nord red)
[done] (Phase 13) i18n: claves draft, draft_banner en en.yaml y es.yaml
[done] (Phase 13) Tests: feedPages excludes drafts, collectSitemapEntries excludes drafts
[done] (Phase 13) Dual-file sync: templates, CSS, i18n

[done] (Phase 14) Refactor shortcode engine: block (paired) + inline (self-closing) types
[done] (Phase 14) parseArgs(): key="value", key='value', key=value, bare positional
[done] (Phase 14) youtube shortcode: responsive 16:9 embed, youtube-nocookie.com, extractVideoID
[done] (Phase 14) twitter/x shortcode: oEmbed blockquote + widgets.js, x.com normalization
[done] (Phase 14) codepen shortcode: iframe embed, height/theme/tab args, fallback link
[done] (Phase 14) figure avanzado: src, caption, alt, class, width, link args
[done] (Phase 14) tabs + tab shortcodes: container with data-tab-title, JS tab switching
[done] (Phase 14) CSS: embeds, figure, details, tabs (Nord-styled)
[done] (Phase 14) JS: tabs.js (zero-dependency, a11y keyboard nav)
[done] (Phase 14) 33 tests unitarios (8 existentes + 25 nuevos)
[done] (Phase 14) Dual-file sync: CSS, JS, templates

[done] (Bugfix) exclude_terms: filtrar terms excluidos de page.Taxonomies antes de pasar a templates
[done] (Bugfix) FilterPageTaxonomies() en taxonomy.go, llamada tras taxonomy.Build() en build.go
[done] (Bugfix) 3 tests: filtrado basico, case-insensitive, sin exclusiones
[done] (Bugfix) Tilde expansion (~) en vault_path para file watcher: config.ExpandTilde() exportada
[done] (Bugfix) normalizePath() en serve_watch.go llama a ExpandTilde antes de filepath.Abs
[done] (Bugfix) Header scroll: nav bar siempre visible, solo titulo grande colapsa con hysteresis
[done] (Bugfix) CSS: grid-template-rows 1fr->0fr, max-width para brand-sm, transiciones sincronizadas
[done] (Bugfix) JS: threshold con hysteresis (compact a 80px, expand bajo 10px), requestAnimationFrame
[done] (Bugfix) Stale content cleanup: removeStaleContent() elimina content/ huerfanos tras update-content
[done] (Bugfix) 4 tests: stale removal, skip _index.md, empty dir, no stale
[done] (Bugfix) Watch loop: EnsureDefaultTheme compara bytes antes de escribir (evita loop infinito de rebuild)
[done] (Bugfix) Dual-file sync: themes/default/ completamente sincronizado con internal/theme/default/

[done] (Phase 15) Config: LanguageConfig struct, Languages field, validacion, helpers IsMultilingual/AllLanguages/LanguageLabel
[done] (Phase 15) Site model: Translation struct, Translations en Page, LinkTranslations() por slug
[done] (Phase 15) Content export: /{lang}/ prefix en output path para idiomas no-default
[done] (Phase 15) Build pipeline: Page.Lang default, LinkTranslations() multilingual, languagesView()
[done] (Phase 15) Templates: 58 llamadas trans actualizadas a trans .lang, date_format con .lang
[done] (Phase 15) hreflang: <link rel="alternate" hreflang> en head.html con x-default
[done] (Phase 15) Language switcher: nav en header + CSS, i18n key aria_language
[done] (Phase 15) og:locale usa .lang del contexto en vez de siempre default_language
[done] (Phase 15) Feeds: xml:lang en Atom, <language> en RSS
[done] (Phase 15) Sitemap: xmlns:xhtml, xhtml:link hreflang alternates para paginas traducidas
[done] (Phase 15) 11 tests: 5 site (LinkTranslations) + 6 config (language validation)
[done] (Phase 15) Dual-file sync: templates, i18n, CSS

[done] (Phase 16) Build timing: instrumentacion por stages con log estructurado al final de cada build
[done] (Phase 16) CPU profiling: `osg build --profile=cpu.prof`, pprof compatible
[done] (Phase 16) Paralelizacion de image optimization: worker pool NumCPU goroutines
[done] (Phase 16) Paralelizacion de minification: worker pool NumCPU goroutines, atomic counter
[done] (Phase 16) Paralelizacion de content parsing + markdown rendering: worker pool, merge secuencial
[done] (Phase 16) Benchmark suite: 18 benchmarks en 5 packages (markdown, summary, build, frontmatter, slug)
[done] (Phase 16) 4 tests unitarios para BuildTimings

[done] (Coverage) 1,644 lineas de nuevos tests en 7 packages: date, slug, content, vault, config, render, theme
[done] (Docs) SHORTCODES.md: guia completa de 11 shortcodes con ejemplos, args, referencia rapida
[done] (Docs) README.md features + QUICKSTART.md seccion shortcodes
[done] (osg.title) osg.title en frontmatter como titulo de maxima precedencia (> fm.title > fm.name > filename)
[done] (osg.title) 3 tests unitarios para osg.title
[done] (menu_title) menu_title derivado de osg.path cuando osg.menu=true, templates usan menu_title con fallback
[done] (menu_title) 6 tests unitarios para menu_title
[done] (Quote) Shortcode quote con author (posicional/key=value) y source, <blockquote class="quote">
[done] (Quote) CSS Nord-styled: borde izquierdo, background sutil, italic
[done] (Quote) 4 tests unitarios + documentacion en SHORTCODES.md
[done] (Quote) Dual-file sync: CSS
[done] (Favicon) Campo Favicon en Config, favicon en configView() para templates
[done] (Favicon) head.html: <link rel="icon"> con fallback config.favicon > config.logo > SVG inline Nord blue
[done] (Favicon) Documentado en DefaultConfigYAML(), dual-file sync head.html

[done] (Permalinks) Placeholders extendidos en content_layout: {year}, {month}, {day}, {title} ademas de {date} y {slug}
[done] (Permalinks) expandPlaceholders() funcion interna compartida, {title} pasa por slug.Slugify()
[done] (Permalinks) osg.permalink en frontmatter: override per-page de URL con soporte de placeholders
[done] (Permalinks) Precedencia: osg.permalink > osg.path > content_layout
[done] (Permalinks) Resolucion de titulo: osg.title > fm.title > fm.name > filename
[done] (Permalinks) 15 tests unitarios (BuildOutputPath, ExpandPermalink, defaults, edge cases)

[done] (Interactions) InteractionsConfig struct: enabled, api_url, listen, db_path, cors_origins, view_dedup_hours
[done] (Interactions) Defaults: enabled=false, listen=":8090", db_path=".osg/interactions.db", view_dedup_hours=24
[done] (Interactions) Validacion y normalizacion en Load(), 3 tests config
[done] (Interactions) internal/api/store.go: SQLite store con modernc.org/sqlite (pure Go, sin CGO)
[done] (Interactions) Schema: page_views (dedup por fingerprint/dia) + page_votes (PK page_path+fingerprint)
[done] (Interactions) RecordView(), Vote() (UPSERT, retract=delete), GetStats()
[done] (Interactions) internal/api/validation.go: PageViewRequest y VoteRequest con validacion
[done] (Interactions) internal/api/middleware.go: CORS middleware con allowlist de origenes
[done] (Interactions) internal/api/server.go: 3 endpoints (pageview, vote, health), stats en respuesta
[done] (Interactions) 14 tests store + 13 tests server + 12 tests validation = 39 tests API
[done] (Interactions) internal/app/api.go: RunAPI() standalone + StartAPIHandler() para embedding
[done] (Interactions) cmd/osg/main.go: osg api command + osg serve --api flag
[done] (Interactions) internal/app/serve.go: refactored a ServeMux, monta API cuando --api
[done] (Interactions) interactions.js: fingerprinting client-side (UUID + browser chars -> SHA-256), sin IP
[done] (Interactions) page.html: bloque page-interactions con views, like/dislike (SVG icons)
[done] (Interactions) style.css: seccion INTERACTIONS Nord-themed, responsive, a11y
[done] (Interactions) i18n: claves interactions_like, interactions_dislike (en + es)
[done] (Interactions) Dual-file sync: page.html, interactions.js, style.css, en.yaml, es.yaml

[done] (Sharing) Copy-link icon next to article title (hover on desktop, always visible on mobile)
[done] (Sharing) Share section below article: X, LinkedIn, Bluesky, Email, Copy link buttons
[done] (Sharing) share.js: clipboard helper, resolveURL() for absolute URLs from relative permalinks
[done] (Sharing) sharing: true config (default enabled), conditional JS/template gating via .config.sharing
[done] (Sharing) i18n: 5 keys (share, share_on, share_via_email, copy_link, link_copied) en + es
[done] (Sharing) CSS: title copy-link opacity animation, share buttons with brand colors on hover
[done] (Sharing) 2 config tests (default enabled, YAML disable) + sharing key in TestConfigView
[done] (Sharing) Dual-file sync: page.html, share.js, style.css, en.yaml, es.yaml

[done] (Comments) CommentsConfig y AuthProviderConfig structs en config.go: defaults, normalizacion, validacion
[done] (Comments) DefaultConfigYAML() con seccion comments documentada
[done] (Comments) CommentStore en internal/api/comment_store.go: SQLite separada (users, sessions, comments)
[done] (Comments) User CRUD: UpsertUser, GetUserByProvider, GetUserByID
[done] (Comments) Session CRUD: CreateSession (crypto random), ValidateSession, DeleteSession, CleanExpiredSessions
[done] (Comments) Comment CRUD: CreateComment, GetComment, SoftDeleteComment, ListComments
[done] (Comments) Tree building: buildCommentTree (two-pass), pruneDeletedLeaves
[done] (Comments) OAuth2 auth handlers: BuildAuthProviders (GitHub read:user, Google openid profile email)
[done] (Comments) AuthHandlers: HandleLogin, HandleCallback, HandleMe, HandleLogout
[done] (Comments) State cookie + return_to flow, code exchange, user info fetch, session creation
[done] (Comments) Comment HTTP handlers: HandleList, HandleCreate, HandleDelete
[done] (Comments) CreateCommentRequest validation (page_path, body max 10000, parent_id same page)
[done] (Comments) Ownership check for delete (solo propios comentarios)
[done] (Comments) Server 5-arg NewServer (commentStore + authProviders nil-safe), rutas condicionales
[done] (Comments) CORS middleware con withCredentials (GET/DELETE, credentials header)
[done] (Comments) RunAPI y StartAPIHandler (4 retornos) con CommentStore lifecycle
[done] (Comments) configView(): comments_enabled, comments_providers con display labels
[done] (Comments) comments.js: IIFE, cookie auth, login URLs, recursive rendering, reply, delete, timeAgo
[done] (Comments) page.html: bloque page-comments, JS condicional
[done] (Comments) CSS: seccion COMMENTS (~200 lineas), nesting 5 niveles, avatars, responsive
[done] (Comments) i18n: 13 claves comentarios en en.yaml y es.yaml
[done] (Comments) 25 tests CommentStore + 21 tests auth + 19 tests comment handlers + 6 tests config
[done] (Comments) golang.org/x/oauth2 v0.35.0 como dependencia directa
[done] (Comments) Dual-file sync: page.html, comments.js, style.css, en.yaml, es.yaml
[done] (Comments) Dockerfile multi-stage (golang:1.25-alpine -> alpine:3.21, non-root, volumes)
[done] (Comments) docker-compose.yml con osg-data volume
[done] (Comments) deploy/k8s/: configmap, pvc, deployment (health probes), service

[done] (TUI-A) Refactor LogSink con campo source (general/serve/api)
[done] (TUI-A) taggedLogLineMsg reemplaza logLineMsg en el Model
[done] (TUI-A) 3 LogSinks en RunTUI (general, serve, api), mensajes separados
[done] (TUI-A) Tests multi-canal logsink

[done] (TUI-B) Model: apiRunning, apiCancel, serveMode; Actions: ServeWithAPI, RunAPI
[done] (TUI-B) Slash commands: /serve --api, /api, /stop serve, /stop api
[done] (TUI-B) Teclas F5 (serve), F6 (api); sidebar Services; header badges

[done] (TUI-C) LogPanel componente: viewport propio, tabs Serve/API/All
[done] (TUI-C) Toggle F7 y /logs, layout panel inferior, foco independiente

[done] (TUI-D) ConfigSchema() en config/schema.go: secciones, campos, tipos, descripciones
[done] (TUI-D) yaml.Node helpers en config/yamlnode.go: LoadNode, Get/SetNodeValue, SaveNode
[done] (TUI-D) Refactorizar UpdatePluginsEnabled para preservar comentarios YAML

[done] (TUI-E) ConfigScreenModel: layout 2 paneles, editores inline por tipo
[done] (TUI-E) Dirty state, Ctrl+S save, Esc unsaved dialog, /config, F8

[done] (TUI-F) Status bar contextual, config reload tras guardar, docs

[done] (Official plugins) CI pipeline: compilar plugins-src/ y publicar .wasm como release assets
[done] (Official plugins) Documentar instalacion via osg plugin install <nombre>
[done] (Official plugins) osg init auto-instala plugins oficiales activos en plugins_enabled que falten en plugins_dir

[done] (Archives) Genera /archive/ cronologico por ano/mes con paginas por ano
[done] (LLMS.txt) Genera /llms.txt y /llms-full.txt con contenido del sitio para LLMs
[done] (Mermaid) Client-side: content.transform reescribe bloques mermaid, build.finished genera mermaid-init.js CDN loader
