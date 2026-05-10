# REQUIREMENTS - OSG

## Goal
Construir un generador de sitios estaticos desde un vault de Obsidian, con un pipeline claro para sincronizar contenido en `content/`, renderizar HTML en `public/`, y ofrecer previsualizacion local.

## Scope
In-scope (actual):
- Lectura de vault y archivos Markdown
- Parse YAML frontmatter
- Filtro publish + include-drafts; auto-promocion de drafts con `publish_at` cuando vence
- Normalizacion de frontmatter de salida
- Copia a `content/{YYYY/MM/DD}/{slug}/index.md`
- Build HTML con templates y theme
- Taxonomias, paginacion, feeds por taxonomia, sitemap, robots y 404
- Assets: static copy + sass (opcional)
- Optimizacion de imagenes (WebP/AVIF embebidos, sin dependencias CLI)
- Plugins WASM (hooks en render) con SDK Rust/Go y registro GitHub
- TUI (terminal) para ejecutar comandos y ver logs
- Web UI dashboard (`osg ui`, loopback) con editor de resumenes por pagina,
  pipeline `init→update-content→check→build→deploy` y audit log persistente
- Serve local de `public/` con watch y live reload
- Build incremental con cache (SQLite)
- Resumenes IA (auto/manual/AI via Kairos) con cache persistente y reintentos
- Scheduler service que despierta a la fecha mas proxima de `publish_at`
- Deploy con staging: drafts y scheduled-future excluidos del upload
- i18n basico (en/es) extensible
- Search index (plugin WASM bundled, opt-in)

Out-of-scope (por ahora):
- Editor multimedia / WYSIWYG (la UI edita solo `osg.summary` por pagina; el
  body del articulo se sigue escribiendo en Obsidian)
- Multi-tenant / colaboracion en tiempo real
- i18n con auto-deteccion / fallbacks complejos

## User stories (con aceptacion)
US01 - Sincronizar contenido
- Como autor, quiero exportar notas desde Obsidian a `content/`.
- Aceptacion: se generan rutas deterministas y frontmatter normalizado.

US02 - Render HTML
- Como operador, quiero generar HTML en `public/`.
- Aceptacion: `osg build` crea paginas, taxonomias y archivos especiales.

US03 - Previsualizar
- Como editor, quiero levantar un servidor local para revisar el sitio.
- Aceptacion: `osg serve` sirve `public/` y se puede detener desde la TUI.

US04 - Theme base
- Como usuario, quiero un tema por defecto usable.
- Aceptacion: `theme: default` entrega HTML y CSS sin templates custom.

US05 - Plugins
- Como developer, quiero enganchar hooks para extender el build.
- Aceptacion: plugins WASM pueden modificar contexto sin romper el build.

US06 - TUI operativa
- Como operador, quiero lanzar init/update/build/serve desde una UI.
- Aceptacion: TUI ejecuta comandos y muestra logs en el panel central.

US07 - Web UI operativa
- Como autor, quiero un dashboard web local para ver el vault, editar
  resumenes y ejecutar el pipeline sin abrir terminal.
- Aceptacion: `osg ui` levanta un servidor en loopback con paginas
  `/` `/vault` `/vault/page` `/actions` `/history` `/services` que
  invocan los mismos comandos que el CLI y muestran logs en vivo.

US08 - Schedule de publicaciones
- Como autor, quiero marcar un draft con una fecha de publicacion
  futura para que se publique automaticamente el dia indicado.
- Aceptacion: con `osg.publish: draft` + `osg.publish_at: <date>` el
  scheduler service despierta el dia indicado, flipea el flag en el
  vault, sincroniza y construye. Drafts y scheduled-futuros nunca se
  publican antes de tiempo (deferred_publications).

US09 - Dashboard persistente
- Como autor, quiero que `osg ui` y sus servicios (scheduler, watcher)
  arranquen al login y sobrevivan a reboots / crashes para no perder
  publish_at por tener el terminal cerrado.
- Aceptacion: `osg service install` escribe una LaunchAgent (macOS)
  o servicio systemd --user (Linux); `ui.autostart` declara que
  servicios deben arrancar dentro del dashboard al boot.

## Functional requirements
- Leer vault dado `--vault-path`.
- Parsear frontmatter YAML entre `---`.
- Filtro publish: true, "true", "draft". Drafts con `osg.publish_at`
  se sincronizan a `content/` aunque `IncludeDrafts=false`.
- `--include-drafts` controla drafts sin `publish_at`.
- Write a `content/{YYYY/MM/DD}/{slug}/index.md`.
- Build HTML con templates y theme (prioridad: user > theme > builtins).
- Render taxonomias + paginacion.
- Generar feeds de taxonomia (atom/rss), sitemap, robots, 404 si hay templates.
- Copiar static y assets del theme.
- Optimizacion de imagenes: encoders WebP y AVIF embebidos via wazero
  (sin requerir `cwebp`/`avifenc` del sistema). PNG originales se
  redimensionan al maximo configurado.
- Compilar sass si `compile_sass=true` y `sass` disponible.
- `clean_public` permite limpiar `public/` en rebuilds completos o cuando se eliminan contenidos.
- `osg serve` sirve `public/`.
- `osg ui` levanta dashboard web en loopback (rechaza binds no-loopback).
- `ui.autostart` (lista) arranca servicios automaticamente al boot
  del dashboard (typical: scheduler, watcher).
- `osg service {install,uninstall,start,stop,status}` instala el
  dashboard como servicio de usuario (LaunchAgent en macOS, systemd
  --user en Linux). Plataformas no soportadas devuelven error claro.
- TUI y Web UI ambas ejecutan los mismos `RunBuild`/`RunDeploy`/...
- Plugins WASM opcionales con hooks definidos (activados via `plugins_enabled`).
- `osg doctor` valida configuracion y entorno.
- `doctor_profile` define severidad `dev|prod`.
- TUI: modo guiado para flujo init → update → build → serve.
- Resumenes IA: provider via `ai.provider` (gemini/anthropic/openai/qwen/ollama),
  cache persistente en `.osg/cache/summaries.db`, reintentos con backoff
  exponencial en errores transitorios.
- Scheduler service: detecta `publish_at` mas proximo, duerme hasta
  vencer (clamp 5min), promociona el draft (vault rewrite) y dispara
  update-content + build.
- Deploy: drafts y scheduled-future se registran en
  `deferred_publications` y se excluyen del upload via staging dir
  (hardlinks). Comportamiento sin overhead cuando no hay nada que
  excluir.

## Non-functional requirements
- Go 1.25.x.
- Output determinista para el mismo input.
- Logs estructurados en JSON.
- Fallos de plugin no deben detener el build (warning).
- Dependencias externas opcionales (sass).

## Constraints / Dependencies
- Go modules, Go 1.25+.
- Librerias clave: kong (CLI), koanf (config), bubbletea/bubbles (TUI),
  wazero (plugins + image encoders), modernc.org/sqlite (audit DB,
  summary cache, build state, sin CGO), gen2brain/webp + gen2brain/avif
  (encoders embebidos), kairos (LLM unificado).
- Sin dependencias CLI externas para imagenes (antes requeria cwebp/avifenc).
