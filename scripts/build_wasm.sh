#!/bin/bash

# Build script for Fluxor WASM EventBus Client

set -e

echo "Building Fluxor WASM EventBus Client..."

# Create output directory
OUT_DIR="pkg/wasm/dist"
mkdir -p "$OUT_DIR"

# Build WASM module
echo "Compiling Go to WASM..."
GOOS=js GOARCH=wasm go build -o "$OUT_DIR/fluxor.wasm" ./pkg/wasm/main.go

# Copy WASM exec helper
echo "Copying WASM exec helper..."
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" "$OUT_DIR/"

# Copy JavaScript bindings
echo "Copying JavaScript bindings..."
cp pkg/wasm/bindings.js "$OUT_DIR/"

echo "Build complete!"
echo "Output files:"
echo "  - $OUT_DIR/fluxor.wasm"
echo "  - $OUT_DIR/wasm_exec.js"
echo "  - $OUT_DIR/bindings.js"

