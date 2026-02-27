# Feed Plugin - Example (WASM, Rust)

> **Nota**: Este plugin es un ejemplo de referencia para aprender a escribir
> plugins WASM para OSG. Los feeds RSS/Atom se generan de forma nativa por
> OSG (ver `site_feed` en config) y no necesitan este plugin.

Genera un RSS basico en `public/rss.xml` al finalizar el build.
Demuestra como:
- Escuchar el evento `build.finished`
- Acceder a `config` y `site.pages` desde el payload
- Escribir ficheros en `public/` via WASI filesystem

## Build

```bash
# Usa wasm32-wasi si esta disponible, o wasm32-wasip1 en toolchains nuevas.
rustc --print=target-list | grep -q "^wasm32-wasi$" && TARGET="wasm32-wasi" || TARGET="wasm32-wasip1"
rustup target add "$TARGET"
cargo build --target "$TARGET" --release
cp target/$TARGET/release/osg_feed.wasm feed.wasm
```

Luego copia `feed.wasm` a `plugins/` y habilitalo en `plugins_enabled`.

## Notas
- Requiere Rust + target `wasm32-wasi`.
- Usa `config.base_url` para construir los links del feed.
- Desde Phase 10 los feeds nativos hacen lo mismo y mas (Atom + RSS).
  Usa este plugin como base para escribir el tuyo propio.
