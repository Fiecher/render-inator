WEB    := docs
GOROOT := $(subst \,/,$(shell go env GOROOT))

.DEFAULT_GOAL := help
.PHONY: help build wasm wasm-exec serve serve-exe test vet fmt clean

help: ## List available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	 awk 'BEGIN{FS=":.*?## "}{printf "  %-11s %s\n", $$1, $$2}'

build: wasm-exec wasm ## Copy the wasm shim, then compile WASM into docs/

wasm: ## Compile Go -> docs/main.wasm
	GOOS=js GOARCH=wasm go build -o $(WEB)/main.wasm ./cmd/wasm

wasm-exec: ## Copy the Go wasm JS shim matching this toolchain
	@cp "$(GOROOT)/lib/wasm/wasm_exec.js" $(WEB)/wasm_exec.js 2>/dev/null || \
	 cp "$(GOROOT)/misc/wasm/wasm_exec.js" $(WEB)/wasm_exec.js

serve: ## Run the local dev server on http://127.0.0.1:8080
	go run ./cmd/serve $(WEB) 127.0.0.1:8080

serve-exe: ## Build cmd/serve/serve.exe (used by the .claude preview config)
	go build -o cmd/serve/serve.exe ./cmd/serve

test: ## Run all Go tests
	go test ./...

vet: ## Vet host packages and the js/wasm package
	go vet ./internal/... ./cmd/serve/...
	GOOS=js GOARCH=wasm go vet ./cmd/wasm/...

fmt: ## Format the tree with gofmt
	gofmt -l -w .

clean: ## Remove generated build artifacts
	rm -f $(WEB)/main.wasm $(WEB)/wasm_exec.js cmd/serve/serve.exe
