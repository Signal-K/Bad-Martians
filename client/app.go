package client

import (
	"encoding/json"
	"sync"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"

	"github.com/signal-k/haphazard-martians/engine"
	"github.com/signal-k/haphazard-martians/game"
)

// App implements ebiten.Game around an engine.Game.
type App struct {
	G *engine.Game
	r *ebitenRenderer

	mu      sync.Mutex
	pending string // scene a host asked to open, applied on the next Update
	edit    int    // editor request from the host: 1 on, 2 off, 3 toggle
}

// New builds the app. If scene names a registered scene it opens that scene on
// its own fixture (standalone); otherwise the game starts at the menu.
func New(host engine.Host, scene string) *App {
	g := engine.NewGame(game.ScreenW, game.ScreenH, host)
	game.Setup(g)
	if scene == "" || !g.OpenStandalone(scene) {
		g.Load(game.SceneMenu, nil)
	}
	return &App{G: g, r: newRenderer()}
}

// Open asks the app to switch to a scene on its fixture. Safe from any goroutine.
func (a *App) Open(scene string) {
	a.mu.Lock()
	a.pending = scene
	a.mu.Unlock()
}

// Edit switches the editor on or off (or toggles it) on the next frame. Safe
// from any goroutine; the web page uses it for ?edit=1 and haphazard.edit().
func (a *App) Edit(mode string) {
	a.mu.Lock()
	a.edit = map[string]int{"on": 1, "off": 2, "toggle": 3}[mode]
	a.mu.Unlock()
}

// SceneList is the registered scenes as JSON, for tooling and the gallery.
func (a *App) SceneList() string {
	type row struct {
		Name, Title, Type, Layout string
		Regions                   map[string]engine.Rect
	}
	var rows []row
	for _, m := range a.G.Scenes() {
		rows = append(rows, row{m.Name, m.Title, string(m.Type), m.Layout.Name, m.Layout.Regions})
	}
	b, _ := json.Marshal(rows)
	return string(b)
}

func (a *App) readInput() engine.Input {
	var in engine.Input
	in.ToggleEdit = inpututil.IsKeyJustPressed(ebiten.KeyF1) || inpututil.IsKeyJustPressed(ebiten.KeyBackquote)
	in.Pause = inpututil.IsKeyJustPressed(ebiten.KeyP)
	in.Step = inpututil.IsKeyJustPressed(ebiten.KeyPeriod)
	in.Delete = inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace)
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
	a.mu.Lock()
	p := a.pending
	a.pending = ""
	ed := a.edit
	a.edit = 0
	a.mu.Unlock()
	if p != "" {
		a.G.OpenStandalone(p)
	}
	in := a.readInput()
	switch ed {
	case 1:
		a.G.SetEditor(true)
	case 2:
		a.G.SetEditor(false)
	case 3:
		in.ToggleEdit = true
	}
	a.G.Tick(1.0/float64(ebiten.TPS()), in)
	return nil
}

func (a *App) Draw(screen *ebiten.Image) {
	a.r.dst = screen
	a.G.Draw(a.r)
}

func (a *App) Layout(int, int) (int, int) { return int(game.ScreenW), int(game.ScreenH) }
