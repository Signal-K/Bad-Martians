// Package client is the only place that knows about Ebitengine. It adapts the
// engine's Renderer and Input to it, so the same game runs in a browser (WASM
// PWA), on desktop, and via ebitenmobile on iOS and Android.
package client

import (
	"bytes"
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gobold"

	"github.com/signal-k/haphazard-martians/engine"
)

var fontSrc *text.GoTextFaceSource

// whitePx is the source texture for every shape: one opaque texel taken from
// the middle of a 3x3 image so filtering never samples a transparent edge.
var whitePx *ebiten.Image

func init() {
	s, err := text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		panic(err)
	}
	fontSrc = s
	w := ebiten.NewImage(3, 3)
	w.Fill(color.White)
	whitePx = w.SubImage(image.Rect(1, 1, 2, 2)).(*ebiten.Image)
}

// ebitenRenderer draws the game's virtual 960x540 screen onto a framebuffer
// at the display's real pixel density. Shapes are tessellated here into one
// triangle batch (a single draw call between text runs) instead of going
// through ebiten/vector: its anti-aliased paths render through stencil passes
// on an offscreen image and flush whenever anything else touches the screen,
// which cost hundreds of passes a frame and made phones crawl. Drawing at
// native resolution keeps edges crisp without anti-aliasing.
type ebitenRenderer struct {
	dst    *ebiten.Image
	faces  map[int]*text.GoTextFace
	scale  float64 // framebuffer pixels per virtual pixel
	ox, oy float64 // letterbox offset in framebuffer pixels

	vs  []ebiten.Vertex
	is  []uint16
	tri []int // scratch for ear clipping
}

func newRenderer() *ebitenRenderer {
	return &ebitenRenderer{faces: map[int]*text.GoTextFace{}, scale: 1}
}

// begin starts a frame on dst, fitting the virtual screen inside it.
func (e *ebitenRenderer) begin(dst *ebiten.Image, vw, vh float64) {
	e.dst = dst
	b := dst.Bounds()
	e.scale, e.ox, e.oy = fit(float64(b.Dx()), float64(b.Dy()), vw, vh)
	e.vs, e.is = e.vs[:0], e.is[:0]
}

// end flushes whatever is still batched.
func (e *ebitenRenderer) end() { e.flush() }

// fit is the scale and centring offset that letterboxes vw x vh into w x h.
func fit(w, h, vw, vh float64) (s, ox, oy float64) {
	s = math.Min(w/vw, h/vh)
	return s, (w - vw*s) / 2, (h - vh*s) / 2
}

func (e *ebitenRenderer) flush() {
	if len(e.is) == 0 {
		return
	}
	e.dst.DrawTriangles(e.vs, e.is, whitePx, &ebiten.DrawTrianglesOptions{})
	e.vs, e.is = e.vs[:0], e.is[:0]
}

// room flushes first if n more vertices would overflow 16-bit indices.
func (e *ebitenRenderer) room(n int) {
	if len(e.vs)+n > math.MaxUint16 {
		e.flush()
	}
}

// vert adds one vertex in virtual coordinates and returns its index.
func (e *ebitenRenderer) vert(x, y float64, c engine.Color) uint16 {
	a := float32(c.A) / 255 // premultiplied
	e.vs = append(e.vs, ebiten.Vertex{
		DstX: float32(x*e.scale + e.ox), DstY: float32(y*e.scale + e.oy),
		SrcX: 1, SrcY: 1,
		ColorR: float32(c.R) / 255 * a, ColorG: float32(c.G) / 255 * a, ColorB: float32(c.B) / 255 * a, ColorA: a,
	})
	return uint16(len(e.vs) - 1)
}

func (e *ebitenRenderer) quad(x0, y0, x1, y1, x2, y2, x3, y3 float64, c engine.Color) {
	e.room(4)
	a, b, cc, d := e.vert(x0, y0, c), e.vert(x1, y1, c), e.vert(x2, y2, c), e.vert(x3, y3, c)
	e.is = append(e.is, a, b, cc, a, cc, d)
}

func (e *ebitenRenderer) Line(a, b engine.Vec2, w float64, c engine.Color) {
	d := b.Sub(a)
	l := d.Len()
	if l < 1e-6 {
		return
	}
	n := engine.V(-d.Y/l*w/2, d.X/l*w/2)
	e.quad(a.X+n.X, a.Y+n.Y, b.X+n.X, b.Y+n.Y, b.X-n.X, b.Y-n.Y, a.X-n.X, a.Y-n.Y, c)
}

// segs is how many sides a circle of radius r (virtual px) needs to look round.
func (e *ebitenRenderer) segs(r float64) int {
	n := int(math.Ceil(2 * math.Pi * r * e.scale / 6))
	return max(10, min(64, n))
}

func (e *ebitenRenderer) Circle(c engine.Vec2, r float64, col engine.Color, fill bool) {
	if r <= 0 {
		return
	}
	n := e.segs(r)
	if fill {
		e.room(n + 1)
		m := e.vert(c.X, c.Y, col)
		first := uint16(len(e.vs))
		for i := 0; i < n; i++ {
			t := float64(i) / float64(n) * 2 * math.Pi
			e.vert(c.X+r*math.Cos(t), c.Y+r*math.Sin(t), col)
		}
		for i := 0; i < n; i++ {
			e.is = append(e.is, m, first+uint16(i), first+uint16((i+1)%n))
		}
		return
	}
	// a 2px ring centred on the radius, as vector.StrokeCircle drew it
	e.room(2 * n)
	ri, ro := math.Max(0, r-1), r+1
	first := uint16(len(e.vs))
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n) * 2 * math.Pi
		cs, sn := math.Cos(t), math.Sin(t)
		e.vert(c.X+ri*cs, c.Y+ri*sn, col)
		e.vert(c.X+ro*cs, c.Y+ro*sn, col)
	}
	for i := 0; i < n; i++ {
		a, b := first+uint16(2*i), first+uint16(2*((i+1)%n))
		e.is = append(e.is, a, a+1, b+1, a, b+1, b)
	}
}

func (e *ebitenRenderer) Rect(x, y, w, h float64, c engine.Color, fill bool) {
	if fill {
		e.quad(x, y, x+w, y, x+w, y+h, x, y+h, c)
		return
	}
	// 2px border centred on the edge, as vector.StrokeRect drew it
	e.Rect(x-1, y-1, w+2, 2, c, true)
	e.Rect(x-1, y+h-1, w+2, 2, c, true)
	e.Rect(x-1, y+1, 2, h-2, c, true)
	e.Rect(x+w-1, y+1, 2, h-2, c, true)
}

// Poly fills a simple polygon (convex or not) by ear clipping.
func (e *ebitenRenderer) Poly(pts []engine.Vec2, c engine.Color) {
	n := len(pts)
	if n < 3 {
		return
	}
	var area float64 // orientation from the signed area
	for i := range pts {
		a, b := pts[i], pts[(i+1)%n]
		area += a.X*b.Y - b.X*a.Y
	}
	if area == 0 {
		return
	}
	sign := 1.0
	if area < 0 {
		sign = -1
	}
	e.room(n)
	base := uint16(len(e.vs))
	for _, p := range pts {
		e.vert(p.X, p.Y, c)
	}
	idx := e.tri[:0]
	for i := 0; i < n; i++ {
		idx = append(idx, i)
	}
	cross := func(a, b, c engine.Vec2) float64 { return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X) }
	for len(idx) > 3 {
		m := len(idx)
		clipped := false
		for k := 0; k < m; k++ {
			i0, i1, i2 := idx[(k+m-1)%m], idx[k], idx[(k+1)%m]
			a, b, cc := pts[i0], pts[i1], pts[i2]
			if cross(a, b, cc)*sign <= 0 { // reflex or flat corner
				continue
			}
			ear := true
			for _, j := range idx {
				if j == i0 || j == i1 || j == i2 {
					continue
				}
				p := pts[j]
				if cross(a, b, p)*sign >= 0 && cross(b, cc, p)*sign >= 0 && cross(cc, a, p)*sign >= 0 {
					ear = false
					break
				}
			}
			if !ear {
				continue
			}
			e.is = append(e.is, base+uint16(i0), base+uint16(i1), base+uint16(i2))
			idx = append(idx[:k], idx[k+1:]...)
			clipped = true
			break
		}
		if !clipped { // self-intersecting or degenerate: fan what is left
			break
		}
	}
	for k := 1; k+1 < len(idx); k++ {
		e.is = append(e.is, base+uint16(idx[0]), base+uint16(idx[k]), base+uint16(idx[k+1]))
	}
	e.tri = idx
}

func (e *ebitenRenderer) Text(s string, x, y, size float64, c engine.Color, a engine.Align) {
	e.flush()                      // painter's order: shapes queued before this text stay underneath
	k := int(size*e.scale*4 + 0.5) // quarter-pixel face sizes, cached
	f := e.faces[k]
	if f == nil {
		f = &text.GoTextFace{Source: fontSrc, Size: float64(k) / 4}
		e.faces[k] = f
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(x*e.scale+e.ox, y*e.scale+e.oy)
	op.ColorScale.ScaleWithColor(color.NRGBA{c.R, c.G, c.B, c.A})
	switch a {
	case engine.AlignCenter:
		op.PrimaryAlign = text.AlignCenter
	case engine.AlignRight:
		op.PrimaryAlign = text.AlignEnd
	}
	text.Draw(e.dst, s, f, op)
}
