# Feed Plugin (WASM, Rust)

Genera un RSS basico en `public/rss.xml` al finalizar el build.

## Build

```bash
rustup target add wasm32-wasi
cargo build --target wasm32-wasi --release
cp target/wasm32-wasi/release/osg_feed.wasm feed.wasm
```

Luego copia `feed.wasm` a `plugins/`.

## Notas
- Requiere Rust + target `wasm32-wasi`.
- Usa `config.base_url` para construir los links del feed.
