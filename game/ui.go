package game

import (
	"fmt"

	"github.com/signal-k/haphazard-martians/engine"
)

// DrawFunc is a Component that draws via a closure; handy for HUD/backdrops.
type DrawFunc func(cam engine.Camera, r engine.Renderer)

func (d DrawFunc) Draw(_ *engine.GameObject, cam engine.Camera, r engine.Renderer) { d(cam, r) }

// Button is a clickable (or holdable) rectangle in screen space.
type Button struct {
	X, Y, W, H float64
	Label      string
	Icon       Icon
	HideLabel  bool // icon-only look; Label stays as the accessible name
	Primary    bool // the scene's main action: neon fill
	Disabled   bool
	OnClick    func()
	// Custom replaces the default look.
	Custom func(b *Button, r engine.Renderer)

	Held  bool // pointer is down inside
	armed bool
}

func (b *Button) inside(p engine.Vec2) bool {
	return p.X >= b.X && p.X <= b.X+b.W && p.Y >= b.Y && p.Y <= b.Y+b.H
}

func (b *Button) Update(o *engine.GameObject, _ float64) {
	in := o.Scene.Game.In
	in_ := b.inside(in.Pointer)
	b.Held = in.Down && in_ && !b.Disabled
	if in.Pressed && in_ {
		b.armed = true
	}
	if in.Released {
		if b.armed && in_ && !b.Disabled && b.OnClick != nil {
			b.OnClick()
		}
		b.armed = false
	}
}

func (b *Button) Draw(_ *engine.GameObject, _ engine.Camera, r engine.Renderer) {
	if b.Custom != nil {
		b.Custom(b, r)
		return
	}
	fill, line, ink := colTile.Alpha(235), colInk, colInk
	switch {
	case b.Disabled:
		line, ink = colMuted.Alpha(90), colMuted.Alpha(120)
	case b.Held:
		fill, ink = colInk, colTile
	case b.Primary:
		fill, ink = colNeon, colTile
	}
	r.Rect(b.X, b.Y, b.W, b.H, fill, true)
	outline(r, b.X, b.Y, b.W, b.H, line)
	cx, cy := b.X+b.W/2, b.Y+b.H/2
	if b.Icon != IconNone {
		is := b.H * 0.52
		if b.HideLabel || b.Label == "" {
			drawIcon(r, b.Icon, cx, cy, is*1.1, ink)
			return
		}
		drawIcon(r, b.Icon, b.X+b.H*0.5, cy, is, ink)
		r.Text(b.Label, (b.X+b.H+b.X+b.W)/2, cy-sizeBody/2-2, sizeBody+1, ink, engine.AlignCenter)
		return
	}
	r.Text(b.Label, cx, cy-sizeBody/2-2, sizeBody+1, ink, engine.AlignCenter)
}

// AddButton spawns a Button object on layer 20.
// Bounds and MoveBy let the engine editor pick and drag a button.
func (b *Button) Bounds(*engine.GameObject, engine.Camera) (x, y, w, h float64) {
	return b.X, b.Y, b.W, b.H
}
func (b *Button) MoveBy(_ *engine.GameObject, _ engine.Camera, dx, dy float64) { b.X += dx; b.Y += dy }

func AddButton(s *engine.Scene, x, y, w, h float64, label string, onClick func()) *Button {
	b := &Button{X: x, Y: y, W: w, H: h, Label: label, OnClick: onClick}
	o := s.Spawn("btn:" + label)
	o.Layer = 20
	o.Add(b)
	return b
}

func AddDraw(s *engine.Scene, layer int, f DrawFunc) {
	o := s.Spawn("draw")
	o.Layer = layer
	o.Add(f)
}

func lerpColor(a, b engine.Color, t float64) engine.Color {
	l := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return engine.Color{R: l(a.R, b.R), G: l(a.G, b.G), B: l(a.B, b.B), A: 255}
}

// drawSky paints a dusty Martian sky with two parallax dune layers.
func drawSky(r engine.Renderer, camX float64) {
	top, hor := engine.RGB(120, 76, 78), engine.RGB(236, 184, 140)
	const bands = 14
	for i := 0; i < bands; i++ {
		t := float64(i) / (bands - 1)
		r.Rect(0, ScreenH*float64(i)/bands, ScreenW, ScreenH/bands+1, lerpColor(top, hor, t*t), true)
	}
	layers := []struct {
		par, amp, base, freq float64
		col                  engine.Color
	}{
		{0.15, 26, 330, 0.011, engine.RGB(190, 128, 100)},
		{0.35, 20, 380, 0.017, engine.RGB(160, 100, 84)},
	}
	for _, L := range layers {
		var pts []engine.Vec2
		pts = append(pts, engine.V(0, ScreenH))
		for x := 0.0; x <= ScreenW+16; x += 16 {
			wx := x + camX*L.par*40
			pts = append(pts, engine.V(x, L.base+L.amp*sinf(wx*L.freq)+L.amp*0.5*sinf(wx*L.freq*2.3+1)))
		}
		pts = append(pts, engine.V(ScreenW+16, ScreenH))
		r.Poly(pts, L.col)
	}
}

func drawStar(r engine.Renderer, cx, cy, rad float64, c engine.Color) {
	pts := make([]engine.Vec2, 0, 10)
	for i := 0; i < 10; i++ {
		a := -1.5708 + float64(i)*0.62832
		rr := rad
		if i%2 == 1 {
			rr = rad * 0.45
		}
		pts = append(pts, engine.V(cx+rr*cosf(a), cy+rr*sinf(a)))
	}
	r.Poly(pts, c)
}

func drawStars(r engine.Renderer, cx, cy, size float64, n int) {
	for i := 0; i < 3; i++ {
		c := engine.RGB(104, 80, 78)
		if i < n {
			c = colStar
		}
		drawStar(r, cx+float64(i-1)*size*2.3, cy, size, c)
	}
}

func fmtTime(t float64) string { return fmt.Sprintf("%.1fs", t) }
