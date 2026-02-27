#!/usr/bin/env bash
set -euo pipefail

# Build an OSG WASM plugin using TinyGo.
# Requires: tinygo (https://tinygo.org/getting-started/install/)

if ! command -v tinygo &>/dev/null; then
  echo "Error: tinygo not found. Install from https://tinygo.org/getting-started/install/" >&2
  exit 1
fi

tinygo build -o "{{name}}.wasm" -target=wasip1 -no-debug .
echo "Built {{name}}.wasm ($(wc -c < "{{name}}.wasm" | tr -d ' ') bytes)"
