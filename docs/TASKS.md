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
