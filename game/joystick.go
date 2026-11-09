package game

import (
	"math"

	"github.com/signal-k/haphazard-martians/engine"
)

// Stick is a floating on-screen joystick. Press anywhere in Zone and the stick
// centres under the finger; dragging sets X (drive) and Y (lean, up is +), each
// in [-1,1]. It rests at Home when nothing holds it.
type Stick struct {
	Home  engine.Vec2
	R     float64     // how far the knob travels, px
	Zone  engine.Rect // where a press grabs the stick
	Block func(p engine.Vec2) bool

	X, Y   float64
	active bool
	base   engine.Vec2
	knob   engine.Vec2 // offset from base
}

const stickDead = 0.12

func (s *Stick) Update(o *engine.GameObject, _ float64) {
	in := o.Scene.Game.In
	if in.Pressed && s.Zone.Contains(engine.Rect{X: in.Pointer.X, Y: in.Pointer.Y}) && (s.Block == nil || !s.Block(in.Pointer)) {
		s.active, s.base = true, in.Pointer
	}
	if !in.Down {
		s.active = false
	}
	if !s.active {
		s.X, s.Y, s.knob = 0, 0, engine.Vec2{}
		return
	}
	d := in.Pointer.Sub(s.base)
	if l := d.Len(); l > s.R {
		// past the rim the base follows the finger, so the stick never sticks
		s.base = s.base.Add(d.Mul((l - s.R) / l))
		d = d.Mul(s.R / l)
	}
	s.knob = d
	ax := func(v float64) float64 {
		v = math.Max(-1, math.Min(1, v/s.R))
		if math.Abs(v) < stickDead {
			return 0
		}
		return v
	}
	s.X, s.Y = ax(d.X), ax(-d.Y)
}

// Held reports whether a finger or the mouse holds the stick.
func (s *Stick) Held() bool { return s.active }

// Box is where the stick is drawn now, for the tutorial pointer.
func (s *Stick) Box() engine.Rect {
	c := s.Home
	if s.active {
		c = s.base
	}
	return engine.Rect{X: c.X - s.R - 14, Y: c.Y - s.R - 14, W: 2*s.R + 28, H: 2*s.R + 28}
}

func (s *Stick) Draw(_ *engine.GameObject, _ engine.Camera, r engine.Renderer) {
	c := s.Home
	a := uint8(150)
	if s.active {
		c, a = s.base, 230
	}
	rr := s.R + 14
	r.Circle(c, rr, colTile.Alpha(a-40), true)
	r.Circle(c, rr, colInk.Alpha(a), false)
	r.Circle(c, rr-1, colInk.Alpha(a/2), false)
	// direction marks: drive left/right, lean up/down
	tri := func(dx, dy float64, col engine.Color) {
		p := c.Add(engine.V(dx, dy).Mul(rr - 12))
		n := engine.V(-dy, dx)
		r.Poly([]engine.Vec2{p.Add(engine.V(dx, dy).Mul(7)), p.Add(n.Mul(7)).Sub(engine.V(dx, dy).Mul(4)), p.Sub(n.Mul(7)).Sub(engine.V(dx, dy).Mul(4))}, col)
	}
	tri(1, 0, colNeon.Alpha(a))
	tri(-1, 0, colInk.Alpha(a))
	tri(0, -1, colChipLt.Alpha(a))
	tri(0, 1, colChipLt.Alpha(a))
	k := c.Add(s.knob)
	r.Circle(k, 30, colInk.Alpha(a), true)
	r.Circle(k, 30, colNeon.Alpha(a), false)
	r.Circle(k, 18, colPanel2.Alpha(a), true)
}

// Bounds and MoveBy let the engine editor pick and drag the stick.
func (s *Stick) Bounds(*engine.GameObject, engine.Camera) (x, y, w, h float64) {
	b := s.Box()
	return b.X, b.Y, b.W, b.H
}
func (s *Stick) MoveBy(_ *engine.GameObject, _ engine.Camera, dx, dy float64) {
	s.Home = s.Home.Add(engine.V(dx, dy))
}
