//go:build js && wasm

// Command wasm is the browser build: Ebitengine on a canvas, with storage and
// events bridged to web/host.js (localStorage + PocketBase).
package main

import (
	"syscall/js"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/signal-k/haphazard-martians/client"
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

// Photo hands a PNG to the page, which shares or downloads it.
func (h jsHost) Photo(name string, png []byte) {
	u8 := js.Global().Get("Uint8Array").New(len(png))
	js.CopyBytesToJS(u8, png)
	h.bridge.Call("photo", name, u8)
}

func main() {
	bridge := js.Global().Get("haphazard")
	ebiten.SetWindowTitle("Haphazard Martians")
	// ?scene=build opens one scene on its own fixture.
	scene := js.Global().Get("URLSearchParams").New(js.Global().Get("location").Get("search")).Call("get", "scene")
	name := ""
	if !scene.IsNull() {
		name = scene.String()
	}
	app := client.New(jsHost{bridge}, name)
	if q := js.Global().Get("URLSearchParams").New(js.Global().Get("location").Get("search")).Call("get", "edit"); !q.IsNull() {
		app.Edit("on")
	}
	bridge.Set("edit", js.FuncOf(func(_ js.Value, a []js.Value) any {
		mode := "toggle"
		if len(a) > 0 {
			mode = a[0].String()
		}
		app.Edit(mode)
		return nil
	}))
	bridge.Set("scenes", js.FuncOf(func(js.Value, []js.Value) any { return app.SceneList() }))
	bridge.Set("open", js.FuncOf(func(_ js.Value, a []js.Value) any { app.Open(a[0].String()); return nil }))
	if err := ebiten.RunGame(app); err != nil {
		panic(err)
	}
}
