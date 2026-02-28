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
