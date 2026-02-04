# DESIGN - OSG (Obsidian Site Generator)

## Overview
OSG es un generador de contenido estatico a partir de un vault de Obsidian. El core en Go 1.25 sincroniza Markdown con frontmatter YAML hacia `content/`, y luego renderiza HTML en `public/` con templates, taxonomias (feeds atom/rss opcionales), sitemap y assets.

El sistema soporta themes, plugins WASM para extender el pipeline y un modo TUI para orquestar tareas y ver logs.

## Architecture
Componentes principales:

- cmd/osg: CLI y wiring
- internal/app: comandos (init, update-content, build, serve, tui)
- internal/config: carga y merge de config (YAML + env + flags)
- internal/vault: descubrimiento de archivos y lectura
- internal/frontmatter: parse YAML y split frontmatter/body
- internal/publish: filtro publish + drafts
- internal/site: modelos y indexado del sitio
- internal/content: writer a `content/`
- internal/build: pipeline de render y generacion
- internal/render: templates, helpers, builtins y overrides
- internal/taxonomy: indices, paginacion y feeds por taxonomia
- internal/assets: static + sass
- internal/plugin: host WASM y hooks
- internal/tui: UI con Bubble Tea
- internal/logging: logging estructurado

## Data flow
1) Load config
2) update-content: discover vault -> parse frontmatter -> filter -> normalize -> write a `content/`
3) build: parse `content/` -> index site -> build taxonomies -> render templates -> public/
4) assets: copy static/theme static + compile sass si aplica
5) plugins: hooks en build.started/build.finished y render de pages/sections/taxonomies (solo si estan activados en config)
6) build incremental: cache de inputs para saltar renders cuando no hay cambios
7) serve/tui: previsualizacion y control del flujo

## APIs / Interfaces
CLI:
- osg init
- osg update-content (default)
- osg build
- osg serve
- osg tui
- osg doctor
- osg theme init <name>
- osg plugin install/enable/disable/list/init
- osg version

Flags (core):
- --vault-path
- --include-drafts
- --dry-run
- --verbose
- --osg-content-dir
- --public-dir
- -c / --config

Config schema (resumen):
- base_url, theme, vault_path
- content_dir, public_dir, templates_dir, static_dir, themes_dir, plugins_dir, plugins_enabled, sass_dir
- content_layout, include_drafts, compile_sass
- tui_prefix, tui_prefix_ms
- clean_public
- logging (level, format)
- taxonomies (name, paginate_by, paginate_path, feed, render)

Frontmatter output:
- title, date, updated, slug, draft, summary, template, lang
- taxonomies (map: tags/area/type por defecto)
- extra (frontmatter extendido)

## Decisions (trade-offs)
- YAML como formato de config: reduce dependencias y alinea con frontmatter.
- update-content como comando por defecto: evita confusiones y favorece CI simple.
- Theme por defecto: entrega salida usable sin obligar a templates custom.
- Plugins WASM: extensibilidad con ABI reducido y sin dependencias nativas.
- TUI: feedback inmediato sin dependencias externas.
- Build incremental: usa stamps (mtime/size) y requiere full rebuild si cambian templates/assets/plugins.

## Risks / Non-goals
Riesgos:
- Vault grande: rendimiento y memoria en lectura masiva.
- Frontmatter inconsistente: fechas mixtas, campos faltantes.
- Colisiones de slug/fecha -> rutas duplicadas.
- Plugins defectuosos: deben fallar en modo warning.
- Eliminacion de contenido no limpia `public/` automaticamente (stale files).

No goals (por ahora):
- CMS completo o editor online.
- Live reload avanzado.
- Search index integrado.

## Open questions
- Empaquetado del theme por defecto al distribuir binarios.
- Estrategia de rebuild incremental.
- i18n real en templates.
