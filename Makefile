.PHONY: wasm build-wasm clean-wasm

# Build WASM module
wasm: build-wasm

build-wasm:
	@echo "Building Fluxor WASM EventBus Client..."
	@mkdir -p pkg/wasm/dist
	@GOOS=js GOARCH=wasm go build -o pkg/wasm/dist/fluxor.wasm ./pkg/wasm/main.go
	@cp $$(go env GOROOT)/misc/wasm/wasm_exec.js pkg/wasm/dist/
	@cp pkg/wasm/bindings.js pkg/wasm/dist/
	@echo "Build complete!"
	@echo "Output: pkg/wasm/dist/"

clean-wasm:
	@rm -rf pkg/wasm/dist
	@echo "Cleaned WASM build artifacts"

