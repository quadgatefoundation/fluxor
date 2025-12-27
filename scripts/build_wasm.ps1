# Build script for Fluxor WASM EventBus Client (PowerShell)

Write-Host "Building Fluxor WASM EventBus Client..." -ForegroundColor Green

# Create output directory
$OUT_DIR = "pkg\wasm\dist"
New-Item -ItemType Directory -Force -Path $OUT_DIR | Out-Null

# Build WASM module
Write-Host "Compiling Go to WASM..." -ForegroundColor Yellow
$env:GOOS = "js"
$env:GOARCH = "wasm"
go build -o "$OUT_DIR\fluxor.wasm" .\pkg\wasm\main.go

# Copy WASM exec helper
Write-Host "Copying WASM exec helper..." -ForegroundColor Yellow
$GOROOT = go env GOROOT
Copy-Item "$GOROOT\misc\wasm\wasm_exec.js" "$OUT_DIR\"

# Copy JavaScript bindings
Write-Host "Copying JavaScript bindings..." -ForegroundColor Yellow
Copy-Item "pkg\wasm\bindings.js" "$OUT_DIR\"

Write-Host "Build complete!" -ForegroundColor Green
Write-Host "Output files:"
Write-Host "  - $OUT_DIR\fluxor.wasm"
Write-Host "  - $OUT_DIR\wasm_exec.js"
Write-Host "  - $OUT_DIR\bindings.js"

