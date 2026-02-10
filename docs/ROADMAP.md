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

## Next (recomendado)
- [done] validacion de config (paths invalidos, taxonomias mal definidas, base_url vacia en prod)
- [done] limpieza de public/ para evitar archivos stale en builds incrementales
- [done] comando de estado/diagnostico (`osg doctor` o `osg status`)
- [todo] TUI: vista de progreso guiada (wizard) y panel de estado no basado en logs
- [todo] `doctor` con perfiles (dev/prod) y warnings mas accionables
- [todo] refinado del theme base (layout, assets y estilos)
