// Package web embeds the PWA shell and, once built, the WASM game for the
// server to serve. Run `make wasm` first so game.wasm and wasm_exec.js exist;
// without them the server still starts but the game 404s.
package web

import "embed"

//go:embed *
var FS embed.FS
