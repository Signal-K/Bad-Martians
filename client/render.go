// Package client is the only place that knows about Ebitengine. It adapts the
// engine's Renderer and Input to it, so the same game runs in a browser (WASM
// PWA), on desktop, and via ebitenmobile on iOS and Android.
package client

import (
	"bytes"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"

	"github.com/signal-k/haphazard-martians/engine"
)

var fontSrc *text.GoTextFaceSource

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		panic(err)
	}
	fontSrc = s
}

type ebitenRenderer struct {
	dst   *ebiten.Image
	faces map[int]*text.GoTextFace
	path  vector.Path
}

func newRenderer() *ebitenRenderer { return &ebitenRenderer{faces: map[int]*text.GoTextFace{}} }

func rgba(c engine.Color) color.Color { return color.NRGBA{c.R, c.G, c.B, c.A} }

func (e *ebitenRenderer) Line(a, b engine.Vec2, w float64, c engine.Color) {
	vector.StrokeLine(e.dst, float32(a.X), float32(a.Y), float32(b.X), float32(b.Y), float32(w), rgba(c), true)
}

func (e *ebitenRenderer) Circle(c engine.Vec2, r float64, col engine.Color, fill bool) {
	if fill {
		vector.FillCircle(e.dst, float32(c.X), float32(c.Y), float32(r), rgba(col), true)
	} else {
		vector.StrokeCircle(e.dst, float32(c.X), float32(c.Y), float32(r), 2, rgba(col), true)
	}
}

func (e *ebitenRenderer) Rect(x, y, w, h float64, c engine.Color, fill bool) {
	if fill {
		vector.FillRect(e.dst, float32(x), float32(y), float32(w), float32(h), rgba(c), false)
	} else {
		vector.StrokeRect(e.dst, float32(x), float32(y), float32(w), float32(h), 2, rgba(c), false)
	}
}

func (e *ebitenRenderer) Poly(pts []engine.Vec2, c engine.Color) {
	if len(pts) < 3 {
		return
	}
	e.path.Reset()
	e.path.MoveTo(float32(pts[0].X), float32(pts[0].Y))
	for _, p := range pts[1:] {
		e.path.LineTo(float32(p.X), float32(p.Y))
	}
	e.path.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(rgba(c))
	vector.FillPath(e.dst, &e.path, &vector.FillOptions{}, op)
}

func (e *ebitenRenderer) Text(s string, x, y, size float64, c engine.Color, a engine.Align) {
	k := int(size)
	f := e.faces[k]
	if f == nil {
		f = &text.GoTextFace{Source: fontSrc, Size: size}
		e.faces[k] = f
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(rgba(c))
	switch a {
	case engine.AlignCenter:
		op.PrimaryAlign = text.AlignCenter
	case engine.AlignRight:
		op.PrimaryAlign = text.AlignEnd
	}
	text.Draw(e.dst, s, f, op)
}
