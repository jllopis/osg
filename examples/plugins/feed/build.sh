#!/usr/bin/env bash
set -euo pipefail

rustup target add wasm32-wasi
cargo build --target wasm32-wasi --release
cp target/wasm32-wasi/release/osg_feed.wasm feed.wasm
