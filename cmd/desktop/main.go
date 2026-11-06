//go:build !js

// Command desktop runs the game in a native window (handy for development).
// Storage is in-memory; progress is not persisted.
package main

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/signal-k/bad-martians/client"
	"github.com/signal-k/bad-martians/engine"
)

func main() {
	ebiten.SetWindowSize(960, 540)
	ebiten.SetWindowTitle("Bad Martians")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(client.New(engine.NewMemHost())); err != nil {
		panic(err)
	}
}
