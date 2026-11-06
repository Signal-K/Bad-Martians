package client

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/signal-k/bad-martians/engine"
	"github.com/signal-k/bad-martians/game"
)

// App implements ebiten.Game around an engine.Game.
type App struct {
	G *engine.Game
	r *ebitenRenderer
}

func New(host engine.Host) *App {
	g := engine.NewGame(game.ScreenW, game.ScreenH, host)
	game.Setup(g)
	return &App{G: g, r: newRenderer()}
}

func (a *App) readInput() engine.Input {
	var in engine.Input
	in.Left = ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA)
	in.Right = ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD)
	if ids := ebiten.AppendTouchIDs(nil); len(ids) > 0 {
		x, y := ebiten.TouchPosition(ids[0])
		in.Pointer, in.Down = engine.V(float64(x), float64(y)), true
		return in
	}
	x, y := ebiten.CursorPosition()
	in.Pointer = engine.V(float64(x), float64(y))
	in.Down = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	return in
}

func (a *App) Update() error {
	a.G.Tick(1.0/float64(ebiten.TPS()), a.readInput())
	return nil
}

func (a *App) Draw(screen *ebiten.Image) {
	a.r.dst = screen
	a.G.Draw(a.r)
}

func (a *App) Layout(int, int) (int, int) { return int(game.ScreenW), int(game.ScreenH) }
