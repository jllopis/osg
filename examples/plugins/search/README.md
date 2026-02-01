# Search Plugin (WASM, Rust)

Genera:
- `public/search.json` con el indice de busqueda
- `public/search/index.html` con una pagina HTML + JS basica

## Build

```bash
rustup target add wasm32-wasi
cargo build --target wasm32-wasi --release
cp target/wasm32-wasi/release/osg_search.wasm search.wasm
```

Luego copia `search.wasm` a `plugins/`.

## Notas
- Requiere Rust + target `wasm32-wasi`.
- Usa `site.pages` y `page.taxonomies` si existen.
