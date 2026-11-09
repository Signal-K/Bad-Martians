package game

import (
	"math"

	"github.com/signal-k/haphazard-martians/engine"
)

// Stick is an on-screen joystick fixed at Home. A finger that lands within
// Grab px of Home takes it, and the stick follows that one finger until it
// lifts, whatever other fingers do: a second tap on a button neither moves nor
// drops it. Dragging sets X (drive) and Y (lean, up is +), each in [-1,1].
type Stick struct {
	Home  engine.Vec2
	R     float64 // how far the knob travels, px
	Grab  float64 // how far from Home a press still takes the stick, px
	Block func(p engine.Vec2) bool

	X, Y   float64
	active bool
	id     int          // the finger holding the stick
	seen   map[int]bool // fingers down on the last update, to spot new presses
	knob   engine.Vec2  // offset from Home
}

const stickDead = 0.12

func (s *Stick) Update(o *engine.GameObject, _ float64) {
	in := o.Scene.Game.In
	touches := in.Touches
	if len(touches) == 0 && in.Down { // hosts and tests that only send the pointer
		touches = []engine.Touch{{ID: engine.MouseTouch, Pos: in.Pointer}}
	}
	now := make(map[int]bool, len(touches))
	for _, t := range touches {
		now[t.ID] = true
		if !s.active && !s.seen[t.ID] && t.Pos.Sub(s.Home).Len() <= s.Grab && (s.Block == nil || !s.Block(t.Pos)) {
			s.active, s.id = true, t.ID
		}
	}
	s.seen = now
	var p engine.Vec2
	if s.active {
		var ok bool
		p, ok = (engine.Input{Touches: touches}).TouchByID(s.id)
		s.active = ok
	}
	if !s.active {
		s.X, s.Y, s.knob = 0, 0, engine.Vec2{}
		return
	}
	d := p.Sub(s.Home)
	if l := d.Len(); l > s.R {
		d = d.Mul(s.R / l) // the base stays put; the knob stops at the rim
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
	return engine.Rect{X: c.X - s.R - 14, Y: c.Y - s.R - 14, W: 2*s.R + 28, H: 2*s.R + 28}
}

func (s *Stick) Draw(_ *engine.GameObject, _ engine.Camera, r engine.Renderer) {
	c := s.Home
	a := uint8(150)
	if s.active {
		a = 230
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
