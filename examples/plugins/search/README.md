# Search Plugin (WASM, Rust)

Genera:
- `public/search.json` con el indice de busqueda
- `public/search/index.html` con una pagina HTML + JS basica

## Build

```bash
# Usa wasm32-wasi si esta disponible, o wasm32-wasip1 en toolchains nuevas.
rustc --print=target-list | grep -q "^wasm32-wasi$" && TARGET="wasm32-wasi" || TARGET="wasm32-wasip1"
rustup target add "$TARGET"
cargo build --target "$TARGET" --release
cp target/$TARGET/release/osg_search.wasm search.wasm
```

Luego copia `search.wasm` a `plugins/`.

## Notas
- Requiere Rust + target `wasm32-wasi`.
- Usa `site.pages` y `page.taxonomies` si existen.
