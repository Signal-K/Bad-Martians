//go:build js && wasm

// Command wasm is the browser build: Ebitengine on a canvas, with storage and
// events bridged to web/host.js (localStorage + PocketBase).
package main

import (
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/signal-k/bad-martians/client"
)

type jsHost struct{ bridge js.Value }

func (h jsHost) Load(key string) []byte {
	v := h.bridge.Call("load", key)
	if v.IsNull() || v.IsUndefined() {
		return nil
	}
	return []byte(v.String())
}
func (h jsHost) Save(key string, b []byte)   { h.bridge.Call("save", key, string(b)) }
func (h jsHost) Emit(event string, p []byte) { h.bridge.Call("emit", event, string(p)) }

func main() {
	bridge := js.Global().Get("badMartians")
	ebiten.SetWindowTitle("Bad Martians")
	if err := ebiten.RunGame(client.New(jsHost{bridge})); err != nil {
		panic(err)
	}
}
