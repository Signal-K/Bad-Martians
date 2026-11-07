//go:build !js

// Command desktop runs the game in a native window (handy for development).
// Storage is in-memory; progress is not persisted.
package main

import (
	"flag"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/signal-k/haphazard-martians/client"
	"github.com/signal-k/haphazard-martians/engine"
)

func main() {
	scene := flag.String("scene", "", "open one scene on its own fixture (menu, build, run, result, review, style)")
	flag.Parse()
	ebiten.SetWindowSize(960, 540)
	ebiten.SetWindowTitle("Haphazard Martians")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(client.New(engine.NewMemHost(), *scene)); err != nil {
		panic(err)
	}
}
