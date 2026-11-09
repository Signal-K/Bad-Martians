package client

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"math"
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

	touch    engine.Vec2 // last touch point, used for the release frame
	touching bool
	touchID  int // the finger the pointer follows

	fbW, fbH int // framebuffer size from the last Layout, in device pixels
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

// toVirtual maps a framebuffer pixel to the game's virtual screen, undoing the
// letterbox the renderer applies.
func (a *App) toVirtual(x, y int) engine.Vec2 {
	s, ox, oy := fit(float64(a.fbW), float64(a.fbH), game.ScreenW, game.ScreenH)
	if s <= 0 {
		return engine.V(float64(x), float64(y))
	}
	return engine.V((float64(x)-ox)/s, (float64(y)-oy)/s)
}

func (a *App) readInput() engine.Input {
	var in engine.Input
	in.ToggleEdit = inpututil.IsKeyJustPressed(ebiten.KeyF1) || inpututil.IsKeyJustPressed(ebiten.KeyBackquote)
	in.Pause = inpututil.IsKeyJustPressed(ebiten.KeyP)
	in.Step = inpututil.IsKeyJustPressed(ebiten.KeyPeriod)
	in.Delete = inpututil.IsKeyJustPressed(ebiten.KeyDelete) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace)
	in.Left = ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA)
	in.Right = ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD)
	in.Up = ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW)
	in.DownKey = ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS)
	if ids := ebiten.AppendTouchIDs(nil); len(ids) > 0 {
		// The pointer follows one finger until it lifts: the order of ids is
		// not stable, so ids[0] could jump to another finger mid-drag.
		primary := -1
		for _, id := range ids {
			p := a.toVirtual(ebiten.TouchPosition(id))
			in.Touches = append(in.Touches, engine.Touch{ID: int(id), Pos: p})
			if a.touching && int(id) == a.touchID {
				primary = int(id)
				in.Pointer = p
			}
		}
		if primary < 0 {
			a.touchID, in.Pointer = int(ids[0]), in.Touches[0].Pos
		}
		in.Down = true
		a.touch, a.touching = in.Pointer, true
		return in
	}
	if a.touching {
		// The finger just lifted: release where it was, not at the (stale) mouse
		// cursor, or buttons that need press-and-release inside never fire on touch.
		a.touching = false
		in.Pointer = a.touch
		return in
	}
	in.Pointer = a.toVirtual(ebiten.CursorPosition())
	in.Down = ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft)
	if in.Down {
		in.Touches = []engine.Touch{{ID: engine.MouseTouch, Pos: in.Pointer}}
	}
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
	if name := a.G.TakePhotoRequest(); name != "" {
		// draw once without the HUD, keep that, then draw the frame as usual
		a.G.Snapping = true
		a.frame(screen)
		a.G.Snapping = false
		if ph, ok := a.G.Host.(engine.PhotoHost); ok {
			if b, err := a.photo(screen); err == nil {
				ph.Photo(name, b)
			}
		}
	}
	a.frame(screen)
}

func (a *App) frame(screen *ebiten.Image) {
	screen.Fill(color.NRGBA{20, 12, 14, 255}) // letterbox bars
	a.r.begin(screen, game.ScreenW, game.ScreenH)
	a.G.Draw(a.r)
	a.r.end()
}

// photoMaxW caps a photo's width: plenty for sharing, quick to encode.
const photoMaxW = 1920

// photo encodes the game area of the screen (without letterbox bars) as PNG.
func (a *App) photo(screen *ebiten.Image) ([]byte, error) {
	b := screen.Bounds()
	pix := make([]byte, 4*b.Dx()*b.Dy())
	screen.ReadPixels(pix)
	full := &image.RGBA{Pix: pix, Stride: 4 * b.Dx(), Rect: image.Rect(0, 0, b.Dx(), b.Dy())}
	s, ox, oy := fit(float64(b.Dx()), float64(b.Dy()), game.ScreenW, game.ScreenH)
	area := image.Rect(int(ox+0.5), int(oy+0.5), int(ox+game.ScreenW*s), int(oy+game.ScreenH*s))
	src := full.SubImage(area).(*image.RGBA)
	out := image.Image(src)
	if w := area.Dx(); w > photoMaxW { // nearest-neighbour downscale
		k := float64(w) / photoMaxW
		h := int(float64(area.Dy()) / k)
		dst := image.NewRGBA(image.Rect(0, 0, photoMaxW, h))
		for y := 0; y < h; y++ {
			for x := 0; x < photoMaxW; x++ {
				dst.Set(x, y, src.At(area.Min.X+int(float64(x)*k), area.Min.Y+int(float64(y)*k)))
			}
		}
		out = dst
	}
	var buf bytes.Buffer
	err := (&png.Encoder{CompressionLevel: png.BestSpeed}).Encode(&buf, out)
	return buf.Bytes(), err
}

// maxFB caps the framebuffer's long side: past this the extra pixels cost fill
// rate on phones without looking any sharper.
const maxFB = 2400

// Layout draws at the display's real pixel density so shapes, drawn without
// anti-aliasing, stay crisp; the renderer letterboxes the virtual screen.
func (a *App) Layout(w, h int) (int, int) {
	d := ebiten.Monitor().DeviceScaleFactor()
	fw, fh := float64(w)*d, float64(h)*d
	if m := math.Max(fw, fh); m > maxFB {
		fw, fh = fw*maxFB/m, fh*maxFB/m
	}
	a.fbW, a.fbH = max(1, int(fw)), max(1, int(fh))
	return a.fbW, a.fbH
}
