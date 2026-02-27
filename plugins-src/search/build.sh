#!/usr/bin/env bash
set -euo pipefail

TARGET="wasm32-wasi"
if ! rustc --print target-list | grep -q "^wasm32-wasi$"; then
  TARGET="wasm32-wasip1"
fi

rustup target add "$TARGET"
cargo build --target "$TARGET" --release
cp "target/$TARGET/release/osg_search.wasm" search.wasm

# Also copy to bundled directory for embedding
cp search.wasm ../../internal/plugin/bundled/search.wasm
