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

## Backlog (deferred)
- [done] i18n en templates
- [todo] galeria de imagenes / lightbox
