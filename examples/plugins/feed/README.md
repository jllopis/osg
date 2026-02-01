# Feed Plugin (WASM, Rust)

Genera un RSS basico en `public/rss.xml` al finalizar el build.

## Build

```bash
# Usa wasm32-wasi si esta disponible, o wasm32-wasip1 en toolchains nuevas.
rustc --print=target-list | grep -q "^wasm32-wasi$" && TARGET="wasm32-wasi" || TARGET="wasm32-wasip1"
rustup target add "$TARGET"
cargo build --target "$TARGET" --release
cp target/$TARGET/release/osg_feed.wasm feed.wasm
```

Luego copia `feed.wasm` a `plugins/`.

## Notas
- Requiere Rust + target `wasm32-wasi`.
- Usa `config.base_url` para construir los links del feed.
