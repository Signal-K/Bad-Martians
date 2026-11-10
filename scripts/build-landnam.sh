#!/usr/bin/env bash
# Build the game for Landnam's site, served at playlandnam.space/martians/.
#   scripts/build-landnam.sh <landnam repo>/web/public/martians
# Same origin as Landnam, so no new domain and Landnam's sign-in carries over:
# host.js adopts Landnam's PocketBase auth (shareAuth) and both use the shared
# Star Sailors backend. The wasm is gzipped (about 5 MB instead of 21 MB) to keep
# the Landnam repo small; index.html unpacks it. The service worker is off: it
# would delete Landnam's caches on the same origin. Landnam's deploy writes
# config.js again with its Clerk key (web/scripts/martians-config.mjs there).
set -euo pipefail
OUT="${1:?usage: scripts/build-landnam.sh <dir>}"
cd "$(dirname "$0")/.."

make wasm >/dev/null
rm -rf "$OUT" && mkdir -p "$OUT"
cp web/index.html web/host.js web/wasm_exec.js web/manifest.webmanifest "$OUT"/
cp -r web/icons "$OUT"/
gzip -9 -n -c web/game.wasm > "$OUT/game.wasm.gz"
cat > "$OUT/config.js" <<'JS'
// Haphazard Martians on playlandnam.space/martians/. Landnam's deploy rewrites
// this file with its Clerk key (web/scripts/martians-config.mjs).
window.HM_CONFIG = { api: "https://signal-k-starsailors.fly.dev", clerk: "", shareAuth: "pocketbase_auth", wasm: "game.wasm.gz", wasmGzip: true, sw: false };
JS
echo "built $(git rev-parse --short HEAD 2>/dev/null): $(du -h "$OUT/game.wasm.gz" | cut -f1) game.wasm.gz in $OUT"
