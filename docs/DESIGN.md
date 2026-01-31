# DESIGN - OSG (Obsidian Site Generator)

## Overview
OSG es un generador de contenido estatico a partir de un vault de Obsidian. El core en Go 1.25 lee Markdown con frontmatter YAML, filtra por publicacion, normaliza metadatos y sincroniza el contenido a `content/` para su posterior render a HTML en `public/`.

Se prioriza un MVP estable y simple (update-content) y se difiere la complejidad (plantillas avanzadas, taxonomias, plugins WASM) por fases.

## Architecture
Componentes principales (core):

- cmd/osg: CLI y wiring
- internal/config: carga y merge de config (file + env + flags)
- internal/vault: descubrimiento de archivos y lectura
- internal/frontmatter: parse YAML y split frontmatter/body
- internal/publish: filtro por publish + draft
- internal/normalize: mapeo a frontmatter de salida
- internal/slug: derivacion de slug
- internal/date: derivacion de fecha
- internal/content: escritor de destino y layout
- internal/log: logging estructurado
- internal/tui: modo TUI minimo (fase posterior)
- internal/plugin: host WASM (fase posterior)

## Data flow
1) Load config
2) Discover Markdown files in vault
3) Read file content
4) Split frontmatter YAML + body
5) Filter by publish
6) Normalize frontmatter (output schema)
7) Compute date and slug
8) Write to content/{YYYY/MM/DD}/{slug}/index.md
9) Report results

## APIs / Interfaces
CLI (MVP):
- osg init
- osg update-content (default)
- osg build

Flags (MVP):
- --vault-path
- --obsidian-vault-base (alias)
- --vault (alias)
- --osg-content-dir
- --dry-run
- --verbose
- --include-drafts

Config file (propuesto: YAML):
- config.yaml (preferido por unificar dependencias con frontmatter)
- Opcional futuro: config.toml

Config schema (minimo):
- vault_path
- content_dir
- public_dir
- templates_dir
- static_dir
- themes_dir
- include_drafts
- layout (path pattern)
- logging (level, format)

Frontmatter output (minimo):
- title
- date
- slug
- draft
- tags
- categories
- template
- summary
- lang
- obsidian (objeto con campos originales relevantes)

## Decisions (trade-offs)
- YAML como formato de config: reduce dependencias y alinea con frontmatter. Riesgo: usuarios esperan TOML. Mitigacion: soporte opcional TOML mas adelante.
- update-content como default: evita confusiones con build y permite CI eficiente.
- draft solo con flag: evita publicar contenido inacabado por defecto.
- TUI minima en MVP: evita sobrecarga de features no usadas (agentes/MCP).
- Sistema WASM en fase posterior: primero definir hooks claros en el pipeline.

## Risks / Non-goals
Riesgos:
- Vault grande: rendimiento y memoria en lectura masiva.
- Frontmatter inconsistente: campos faltantes o formatos de fecha mixtos.
- Conflictos de slug y fechas -> colisiones de paths.

No goals (MVP):
- Render completo de HTML y pipeline de assets
- Taxonomias y paginacion
- Plugins WASM
- TUI avanzada

## Open questions
- Config final: solo YAML o YAML + TOML desde inicio?
- Regla exacta de colisiones de slug (sufijos, hash, fail-hard)?
- Soporte de multilenguaje en MVP?
