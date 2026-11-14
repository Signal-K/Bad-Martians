.PHONY: all wasm server run test desktop clean
GOROOT_DIR := $(shell go env GOROOT)

all: wasm server

# Browser build: WASM + the matching wasm_exec.js from the same toolchain.
wasm:
	GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o web/game.wasm ./cmd/wasm
	install -m 644 "$(GOROOT_DIR)/lib/wasm/wasm_exec.js" web/wasm_exec.js

server:
	go build -o bin/server ./cmd/server

# Local backend + PWA at http://127.0.0.1:8090 (admin UI at /_/).
run: wasm
	go run ./cmd/server serve --http 127.0.0.1:8090

test:
	go test ./...

desktop:
	go run ./cmd/desktop

clean:
	rm -rf bin web/game.wasm web/wasm_exec.js pb_data

# Browser-rendered scene coverage; install Playwright and Chromium first.
visual-test: wasm
	node tests/visual/scenes.cjs
