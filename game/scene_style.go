package game

import (
	"fmt"

	"github.com/signal-k/haphazard-martians/engine"
)

// styleScene is the living style guide: every token and widget in one place.
// If a widget looks wrong anywhere in the game, it looks wrong here first.
func styleScene(g *engine.Game, _ any) *engine.Scene {
	s := newScene()
	addBackdrop(s)
	addTitle(s, "Style sheet", "Tokens and widgets shared by every scene.")
	swatches := []struct {
		n string
		c engine.Color
	}{{"ink", colInk}, {"muted", colMuted}, {"neon", colNeon}, {"chip", colChip}, {"chip light", colChipLt},
		{"good", colGood}, {"warn", colWarn}, {"star", colStar}, {"mars", colMars}, {"tile", colTile}}
	AddDraw(s, 2, func(_ engine.Camera, r engine.Renderer) {
		drawPanel(r, 28, 124, 300, 160, "Colour")
		for i, sw := range swatches {
			x, y := 44+float64(i%5)*54, 148+float64(i/5)*62
			r.Rect(x, y, 44, 30, sw.c, true)
			outline(r, x, y, 44, 30, colInk)
			r.Text(sw.n, x+22, y+34, 10, colMuted, engine.AlignCenter)
		}
		drawPanel(r, 348, 124, 584, 160, "Icons")
		for i, ic := range AllIcons {
			x, y := 380+float64(i%8)*66, 150+float64(i/8)*62
			drawIcon(r, ic, x, y+16, 36, colInk)
			r.Text(string(ic), x, y+38, 10, colMuted, engine.AlignCenter)
		}
		drawPanel(r, 28, 312, 300, 144, "Tiles")
		for i, st := range []TileState{TileNormal, TileSelected, TileHeld, TileDisabled} {
			drawTile(r, 44+float64(i)*68, 342, 56, IconWheel, st)
		}
		r.Text("normal  selected  held  disabled", 44, 414, 10, colMuted, engine.AlignLeft)
		drawPanel(r, 348, 312, 300, 144, "Chips and meters")
		x := 366.0
		x += drawChip(r, x, 336, IconCredit, "12/14", colChip) + 8
		x += drawChip(r, x, 336, IconReview, "3/5", colChipLt) + 8
		drawChip(r, x, 336, IconStarIco, "900", colWarn)
		for i := 0; i < 3; i++ {
			drawMeter(r, 366, 388+float64(i)*20, 262, 10, (i+1)*3, colNeon)
		}
		drawPanel(r, 668, 312, 264, 144, "Buttons")
		r.Text(fmt.Sprintf("line %.0fpx · chamfer %.0f · touch ≥ %.0f", lineW, cutC, touchMin), 684, 438, 11, colMuted, engine.AlignLeft)
	})
	addBack(s, func() { g.Load(SceneMenu, nil) })
	AddButton(s, 684, 336, 112, 44, "Default", nil)
	p := AddButton(s, 804, 336, 112, 44, "Primary", nil)
	p.Primary = true
	d := AddButton(s, 684, 388, 112, 44, "Disabled", nil)
	d.Disabled = true
	AddButton(s, 804, 388, 112, 44, "Icon", nil).Icon = IconRover
	return s
}
