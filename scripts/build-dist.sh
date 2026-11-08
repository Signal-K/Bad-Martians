#!/usr/bin/env bash
# Build the static site into dist/ for Cloudflare (or any static host).
#   HM_API_URL   public URL of the backend; empty (default) builds the no-backend demo
#   HM_BUILD_ID  cache version for the service worker; defaults to the git short sha
set -euo pipefail
cd "$(dirname "$0")/.."

BUILD_ID="${HM_BUILD_ID:-$(git rev-parse --short HEAD 2>/dev/null || date +%s)}"
API_URL="${HM_API_URL:-}"
MAX_WASM_BYTES=$((23 * 1024 * 1024)) # Cloudflare allows 25 MiB per static file

make wasm >/dev/null
rm -rf dist && mkdir dist
cp -r web/. dist/
rm -f dist/embed.go

# Static hosting config: the backend URL is public; secrets never go in here.
printf 'window.HM_CONFIG = { api: %s };\n' "$(printf '%s' "$API_URL" | python3 -c 'import json,sys; print(json.dumps(sys.stdin.read()))')" > dist/config.js
# A new build id makes returning players refetch the shell instead of keeping the old game.
sed -i "s/haphazard-martians-v1/haphazard-martians-${BUILD_ID}/" dist/sw.js
cp deploy/_headers dist/_headers

size=$(stat -c %s dist/game.wasm)
echo "build ${BUILD_ID}: game.wasm ${size} bytes, backend '${API_URL:-none (demo)}'"
if [ "$size" -gt "$MAX_WASM_BYTES" ]; then
  echo "game.wasm is over the 23 MiB budget" >&2; exit 1
fi
