# REQUIREMENTS - OSG

## Goal
Construir un generador de sitios estaticos desde un vault de Obsidian, con un pipeline claro para sincronizar contenido en `content/`, renderizar HTML en `public/`, y ofrecer previsualizacion local.

## Scope
In-scope (actual):
- Lectura de vault y archivos Markdown
- Parse YAML frontmatter
- Filtro publish + include-drafts
- Normalizacion de frontmatter de salida
- Copia a `content/{YYYY/MM/DD}/{slug}/index.md`
- Build HTML con templates y theme
- Taxonomias, paginacion, feeds por taxonomia, sitemap, robots y 404
- Assets: static copy + sass (opcional)
- Plugins WASM (hooks en render)
- TUI para ejecutar comandos y ver logs
- Serve local de `public/` con watch y live reload
- Build incremental con cache

Out-of-scope (por ahora):
- Editor visual / CMS
- Search index integrado
- i18n completo

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

## Functional requirements
- Leer vault dado `--vault-path`.
- Parsear frontmatter YAML entre `---`.
- Filtro publish: true, "true", "draft".
- `--include-drafts` controla drafts.
- Write a `content/{YYYY/MM/DD}/{slug}/index.md`.
- Build HTML con templates y theme (prioridad: user > theme > builtins).
- Render taxonomias + paginacion.
- Generar feeds de taxonomia (atom/rss), sitemap, robots, 404 si hay templates.
- Copiar static y assets del theme.
- Compilar sass si `compile_sass=true` y `sass` disponible.
- `clean_public` permite limpiar `public/` en rebuilds completos o cuando se eliminan contenidos.
- `osg serve` sirve `public/`.
- TUI: acciones rapidas + prompt + logs.
- Plugins WASM opcionales con hooks definidos (activados via `plugins_enabled`).
- `osg doctor` valida configuracion y entorno.

## Non-functional requirements
- Go 1.25.x.
- Output determinista para el mismo input.
- Logs estructurados en JSON.
- Fallos de plugin no deben detener el build (warning).
- Dependencias externas opcionales (sass).

## Constraints / Dependencies
- Go modules.
- Librerias clave: kong (CLI), koanf (config), bubbletea/bubbles (TUI), wazero (plugins).
