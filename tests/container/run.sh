#!/usr/bin/env bash
# Run an isolated test container using the local Go toolchain and Playwright.
set -euo pipefail
cd "$(dirname "$0")/../.."
GO_ROOT="$(go env GOROOT)"
GO_PATH="$(go env GOPATH | cut -d: -f1)"
GO_CACHE="$(go env GOCACHE)"
PLAYWRIGHT_DIR="$(node -p "require('path').dirname(require('path').dirname(require.resolve('playwright/package.json')))")"
docker build -q -t haphazard-test -f tests/container/Dockerfile tests/container
docker run --rm --shm-size=1g \
  -v "$PWD:/src" -v "$GO_ROOT:/go:ro" -v "$GO_PATH/pkg/mod:/gopath/pkg/mod:ro" \
  -v "$GO_CACHE:/tmp/go-cache" -v "$PLAYWRIGHT_DIR:/node_modules:ro" haphazard-test sh -ec '
    go version
    go vet ./engine ./physics ./game ./client
    go test ./...
    HM_BUILD_ID=container-test scripts/build-dist.sh
    go build -o /tmp/haphazard-server ./cmd/server
    python3 tests/container/backend_smoke.py
    node tests/visual/scenes.cjs
  '
