# Plugins (WASM)

## Objetivo
Permitir ampliar OSG mediante plugins WASM seguros (Wazero), con hooks sobre el pipeline.

## Directorio
- `plugins/`: colocar archivos `.wasm`

## Instalar vs activar
- **Instalar**: copiar el `.wasm` en `plugins/` (o usar `osg plugin install <path>`).
- **Activar**: añadir el nombre del plugin (sin `.wasm`) en `plugins_enabled` del config, o ejecutar `osg plugin enable <name>`.
- **Desactivar**: `osg plugin disable <name>` o eliminarlo de `plugins_enabled`.
- El orden de carga es alfabetico por nombre de archivo.
- Si un plugin esta activado pero el `.wasm` no existe, se ignora y se registra un warning.

### CLI
```bash
osg plugin install ./dist/my_plugin.wasm --name my-plugin
osg plugin enable my-plugin
osg plugin disable my-plugin
osg plugin list
osg plugin init my-plugin --dir plugins_src
```

### TUI
En el prompt de la TUI:
```
plugin enable my-plugin
plugin disable my-plugin
plugin toggle my-plugin
plugin list
```

## ABI
El plugin debe exportar:
- `alloc(size: i32) -> i32`
- `handle_event(ptr: i32, len: i32) -> i64`
- `dealloc(ptr: i32, len: i32)` (opcional)

`handle_event` devuelve un `i64` con:
- 32 bits altos: puntero de salida
- 32 bits bajos: longitud de salida

La salida es JSON.

### Entrada JSON

```json
{
  "type": "page.render",
  "payload": { ... }
}
```

### Salida JSON

```json
{
  "payload": {
    "page": { "title": "Nuevo titulo" }
  }
}
```

## Eventos
- `build.started`
- `build.finished`
- `page.render`
- `section.render`
- `taxonomy.list.render`
- `taxonomy.term.render`

## Ciclo de vida
1) OSG carga todos los `.wasm` al inicio del build (orden alfabetico por nombre de archivo).
2) Se emite `build.started` antes de renderizar.
3) Por cada seccion/pagina/taxonomia se emiten `section.render`, `page.render`, `taxonomy.list.render` y `taxonomy.term.render`.
4) Se emite `build.finished` al final.
5) OSG cierra los plugins al terminar el build.

Errores en un plugin no detienen el build; se registran como warning.

## Payloads (resumen)
- `build.started`: { config, site, taxonomies, stats }
- `build.finished`: { config, site, taxonomies, stats }
- `page.render`: { config, page, current_path, current_url, lang }
- `section.render`: { config, section, current_path, current_url, lang }
- `taxonomy.list.render`: { config, taxonomy, terms, current_path, current_url, lang }
- `taxonomy.term.render`: { config, taxonomy, term, paginator?, current_path, current_url, lang }

`stats` incluye: total, rendered, skipped, cached, errors, pages, sections.

## Notas
- El host mergea `payload` sobre el contexto actual (merge profundo de mapas).
- Plugins que fallen no detienen el build, solo generan warning.

## Ejemplos (Rust)

### RSS feed
En `examples/plugins/feed` hay un plugin Rust que:
- escucha `build.finished`
- genera `public/rss.xml` a partir de `site.pages`

Build:
```bash
cd examples/plugins/feed
rustc --print=target-list | grep -q "^wasm32-wasi$" && TARGET="wasm32-wasi" || TARGET="wasm32-wasip1"
rustup target add "$TARGET"
cargo build --target "$TARGET" --release
cp target/$TARGET/release/osg_feed.wasm feed.wasm
```

### Search index
En `examples/plugins/search` hay un plugin Rust que:
- escucha `build.finished`
- genera `public/search.json` y `public/search/index.html`

Build:
```bash
cd examples/plugins/search
rustc --print=target-list | grep -q "^wasm32-wasi$" && TARGET="wasm32-wasi" || TARGET="wasm32-wasip1"
rustup target add "$TARGET"
cargo build --target "$TARGET" --release
cp target/$TARGET/release/osg_search.wasm search.wasm
```

Copia `feed.wasm` / `search.wasm` a `plugins/` para activarlos.
