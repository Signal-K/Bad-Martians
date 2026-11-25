//go:build !js

// Command desktop runs the game in a native window (handy for development).
// Storage is in-memory; progress is not persisted.
package main

import (
	"flag"
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/signal-k/haphazard-martians/client"
	"github.com/signal-k/haphazard-martians/engine"
	"github.com/signal-k/haphazard-martians/game"
)

func main() {
	scene := flag.String("scene", "", "open one scene on its own fixture (menu, build, run, result, review, style)")
	level := flag.Int("level", 1, "level the build, launch and run fixtures open on (21 Phobos, 22 Deimos, 23 Moon)")
	flag.Parse()
	if game.GetLevel(*level) != nil {
		game.FixtureLevel = *level
	}
	ebiten.SetWindowSize(960, 540)
	ebiten.SetWindowTitle("Haphazard Martians")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(client.New(engine.NewMemHost(), *scene)); err != nil {
		panic(err)
	}
}
