package game

import "github.com/signal-k/haphazard-martians/engine"

// textW estimates the pixel width of s at the given size (the UI font is wide).
func textW(s string, size float64) float64 { return float64(len([]rune(s))) * size * 0.6 }

// outline strokes a rectangle at the theme's line weight.
func outline(r engine.Renderer, x, y, w, h float64, c engine.Color) {
	r.Rect(x, y, w, h, c, false)
	r.Rect(x+1, y+1, w-2, h-2, c, false)
}

func chamfer(x, y, w, h, c float64) []engine.Vec2 {
	return []engine.Vec2{
		engine.V(x+c, y), engine.V(x+w, y), engine.V(x+w, y+h-c),
		engine.V(x+w-c, y+h), engine.V(x, y+h), engine.V(x, y+c),
	}
}

// drawTab draws a neon label that sits on a panel's top edge. It returns the
// tab's width.
func drawTab(r engine.Renderer, x, y float64, label string) float64 {
	w := textW(label, sizeTab) + 28
	r.Poly([]engine.Vec2{engine.V(x, y), engine.V(x+w, y), engine.V(x+w-10, y+24), engine.V(x, y+24)}, colSpace)
	r.Text(label, x+12, y+3, sizeTab, colNeon, engine.AlignLeft)
	return w
}

// drawPanel is the standard container: black, chamfered, white outline, with
// an optional neon tab label on the top edge.
func drawPanel(r engine.Renderer, x, y, w, h float64, label string) {
	pts := chamfer(x, y, w, h, cutC)
	r.Poly(pts, colTile.Alpha(225))
	for i := range pts {
		r.Line(pts[i], pts[(i+1)%len(pts)], lineW, colInk)
	}
	if label != "" {
		drawTab(r, x+20, y-16, label)
	}
}

// TileState is how a tile looks.
type TileState int

const (
	TileNormal TileState = iota
	TileSelected
	TileHeld
	TileDisabled
)

// drawTile is the black icon tile with a white glyph.
func drawTile(r engine.Renderer, x, y, s float64, ic Icon, st TileState) {
	fill, line, glyph := colTile, colInk, colInk
	switch st {
	case TileSelected:
		line = colNeon
	case TileHeld:
		fill, glyph = colInk, colTile
	case TileDisabled:
		line, glyph = colMuted.Alpha(110), colMuted.Alpha(110)
	}
	r.Rect(x, y, s, s, fill, true)
	outline(r, x, y, s, s, line)
	if st == TileSelected {
		outline(r, x-3, y-3, s+6, s+6, colNeon.Alpha(120))
	}
	drawIcon(r, ic, x+s/2, y+s/2, s*0.62, glyph)
}

// drawChip is a small resource readout: icon, then value. Returns its width.
func drawChip(r engine.Renderer, x, y float64, ic Icon, txt string, fill engine.Color) float64 {
	w := textW(txt, sizeBody) + 20
	if ic != IconNone {
		w += 26
	}
	r.Rect(x, y, w, 30, fill, true)
	outline(r, x, y, w, 30, colInk)
	tx := x + 10
	if ic != IconNone {
		drawIcon(r, ic, x+20, y+15, 20, colInk)
		tx += 26
	}
	r.Text(txt, tx, y+5, sizeBody, colInk, engine.AlignLeft)
	return w
}

// drawMeter is a segmented bar.
func drawMeter(r engine.Renderer, x, y, w float64, segs, filled int, col engine.Color) {
	const h, gap = 14.0, 3.0
	sw := (w - gap*float64(segs-1)) / float64(segs)
	for i := 0; i < segs; i++ {
		c := colPanel2
		if i < filled {
			c = col
		}
		r.Rect(x+float64(i)*(sw+gap), y, sw, h, c, true)
	}
}

// drawBackdrop paints the dark nebula behind menu-style scenes.
func drawBackdrop(r engine.Renderer) {
	r.Rect(0, 0, ScreenW, ScreenH, colSpace, true)
	blobs := []struct {
		x, y, rad float64
		c         engine.Color
	}{
		{190, 150, 270, engine.RGB(170, 36, 40)},
		{720, 120, 210, engine.RGB(36, 160, 100)},
		{780, 430, 290, engine.RGB(220, 116, 44)},
		{70, 470, 250, engine.RGB(46, 76, 190)},
	}
	for _, b := range blobs {
		for i := 0; i < 5; i++ {
			f := 1 - float64(i)*0.2
			r.Circle(engine.V(b.x, b.y), b.rad*f, b.c.Alpha(14), true)
		}
	}
	rg := newRng(7)
	for i := 0; i < 90; i++ {
		a := uint8(70 + rg.intn(185))
		r.Circle(engine.V(rg.float()*ScreenW, rg.float()*ScreenH), 0.6+rg.float()*1.1, colInk.Alpha(a), true)
	}
}
