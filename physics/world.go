// Package physics is a minimal deterministic 2D position-based-dynamics
// engine: round particles (some are wheels), distance constraints and a
// heightfield terrain with per-segment materials. Vehicles are lattices of
// particles joined by stiff constraints, which is enough for Bad-Piggies-style
// contraptions and trivially portable (pure Go, no allocations in the hot loop).
package physics

import (
	"math"

	"github.com/signal-k/haphazard-martians/engine"
)

type Vec = engine.Vec2

// V builds a Vec.
func V(x, y float64) Vec { return Vec{X: x, Y: y} }

// Material describes a terrain surface.
type Material struct {
	Friction float64 // Coulomb coefficient
	Drag     float64 // 1/s velocity damping while in contact (sand sinking)
}

// Terrain is a uniformly sampled heightfield; segment i spans H[i]..H[i+1].
type Terrain struct {
	X0, DX    float64
	H         []float64
	Seg       []uint8 // material index per segment (len(H)-1)
	Materials []Material
}

func (t *Terrain) Len() int     { return len(t.H) }
func (t *Terrain) pt(i int) Vec { return V(t.X0+float64(i)*t.DX, t.H[i]) }

// HeightAt returns the interpolated surface height at x (clamped to the ends).
func (t *Terrain) HeightAt(x float64) float64 {
	f := (x - t.X0) / t.DX
	if f <= 0 {
		return t.H[0]
	}
	if f >= float64(len(t.H)-1) {
		return t.H[len(t.H)-1]
	}
	i := int(f)
	u := f - float64(i)
	return t.H[i]*(1-u) + t.H[i+1]*u
}

// MaterialAt returns the material under x.
func (t *Terrain) MaterialAt(x float64) Material {
	i := int((x - t.X0) / t.DX)
	if i < 0 {
		i = 0
	}
	if i >= len(t.Seg) {
		i = len(t.Seg) - 1
	}
	return t.Materials[t.Seg[i]]
}

// Body is a round particle.
type Body struct {
	Pos, Prev Vec
	Radius    float64
	InvMass   float64
	Friction  float64 // body's own grip; combined with the surface's
	Tag       int     // free for the gameplay layer

	// Wheel state. Angle is the visual rotation (radians, clockwise-positive
	// matches rolling towards +X).
	Wheel    bool
	Angle    float64
	Omega    float64
	Motor    bool
	Throttle float64 // -1..1, set by the gameplay layer

	OnGround bool
	Ground   Material
}

func (b *Body) Vel(h float64) Vec { return b.Pos.Sub(b.Prev).Mul(1 / h) }

// Constraint keeps two bodies a fixed distance apart.
type Constraint struct {
	A, B      *Body
	Rest      float64
	Stiffness float64 // 0..1 per iteration
}

// Tunables for wheels/motors.
const (
	MaxOmega      = 16.0 // rad/s at full throttle
	MotorAccel    = 60.0 // rad/s² while driving
	BrakeAccel    = 30.0 // rad/s² towards zero when throttle is released
	wheelTraction = 0.5  // share of slip taken by body (rest goes to spin)
)

type World struct {
	Gravity    Vec
	Terrain    *Terrain
	Bodies     []*Body
	Cons       []*Constraint
	Substeps   int
	Iterations int
	Damping    float64 // 1/s air damping
}

func NewWorld() *World {
	return &World{Gravity: V(0, -3.71), Substeps: 10, Iterations: 6, Damping: 0.05}
}

func (w *World) AddBody(b *Body) *Body {
	b.Prev = b.Pos
	w.Bodies = append(w.Bodies, b)
	return b
}

func (w *World) Connect(a, b *Body, stiffness float64) *Constraint {
	c := &Constraint{A: a, B: b, Rest: a.Pos.Sub(b.Pos).Len(), Stiffness: stiffness}
	w.Cons = append(w.Cons, c)
	return c
}

// SubstepDT is the substep length Step(dt) uses.
func (w *World) SubstepDT(dt float64) float64 { return dt / float64(max(1, w.Substeps)) }

// Step advances the world by dt seconds.
func (w *World) Step(dt float64) {
	n := w.Substeps
	if n < 1 {
		n = 1
	}
	h := dt / float64(n)
	for s := 0; s < n; s++ {
		w.integrate(h)
		for it := 0; it < w.Iterations; it++ {
			w.solveConstraints()
		}
		w.collide(h)
	}
}

func (w *World) integrate(h float64) {
	for _, b := range w.Bodies {
		if b.InvMass == 0 {
			continue
		}
		if b.Wheel && b.Motor {
			target := b.Throttle * MaxOmega
			rate := MotorAccel
			if b.Throttle == 0 {
				rate = BrakeAccel
			}
			d := target - b.Omega
			m := rate * h
			b.Omega += math.Max(-m, math.Min(m, d))
		}
		vel := b.Pos.Sub(b.Prev).Mul(1 - w.Damping*h)
		vel = vel.Add(w.Gravity.Mul(h * h))
		b.Prev = b.Pos
		b.Pos = b.Pos.Add(vel)
		if b.Wheel {
			b.Angle += b.Omega * h
		}
		b.OnGround = false
	}
}

func (w *World) solveConstraints() {
	for _, c := range w.Cons {
		wsum := c.A.InvMass + c.B.InvMass
		if wsum == 0 {
			continue
		}
		d := c.B.Pos.Sub(c.A.Pos)
		l := d.Len()
		if l < 1e-9 {
			continue
		}
		k := (l - c.Rest) / l / wsum * c.Stiffness
		c.A.Pos = c.A.Pos.Add(d.Mul(k * c.A.InvMass))
		c.B.Pos = c.B.Pos.Sub(d.Mul(k * c.B.InvMass))
	}
}

// contact finds the deepest contact between a circle and the terrain.
func (t *Terrain) contact(p Vec, r float64) (n Vec, depth float64, seg int, ok bool) {
	i0 := int(math.Floor((p.X-r-t.X0)/t.DX)) - 0
	i1 := int(math.Floor((p.X+r-t.X0)/t.DX)) + 0
	if i0 < 0 {
		i0 = 0
	}
	if i1 > len(t.H)-2 {
		i1 = len(t.H) - 2
	}
	for i := i0; i <= i1; i++ {
		a, b := t.pt(i), t.pt(i+1)
		e := b.Sub(a)
		el := e.Len()
		tn := e.Mul(1 / el)
		nrm := V(-tn.Y, tn.X) // up-facing normal
		u := p.Sub(a).Dot(tn)
		var cn Vec
		var d float64
		if u >= 0 && u <= el {
			sd := p.Sub(a).Dot(nrm)
			if sd >= r || sd < -3*r {
				continue
			}
			cn, d = nrm, r-sd
		} else {
			q := a
			if u > el {
				q = b
			}
			dv := p.Sub(q)
			dl := dv.Len()
			if dl >= r || dl < 1e-9 {
				continue
			}
			cn, d = dv.Mul(1/dl), r-dl
		}
		if d > depth {
			n, depth, seg, ok = cn, d, i, true
		}
	}
	return
}

func (w *World) collide(h float64) {
	t := w.Terrain
	if t == nil {
		return
	}
	for _, b := range w.Bodies {
		if b.InvMass == 0 {
			continue
		}
		n, d, seg, ok := t.contact(b.Pos, b.Radius)
		if !ok {
			continue
		}
		mat := t.Materials[t.Seg[seg]]
		b.OnGround, b.Ground = true, mat
		b.Pos = b.Pos.Add(n.Mul(d))

		// Friction in displacement space (Coulomb: |tangential| <= mu * normal).
		tan := V(n.Y, -n.X)
		disp := b.Pos.Sub(b.Prev)
		// Remove inward normal velocity (inelastic).
		if vn := disp.Dot(n); vn < 0 {
			b.Prev = b.Prev.Add(n.Mul(vn))
			disp = b.Pos.Sub(b.Prev)
		}
		vt := disp.Dot(tan)
		mu := mat.Friction * b.Friction
		maxCorr := mu * d
		if b.Wheel {
			roll := b.Omega * h * b.Radius
			slip := vt - roll
			corr := math.Max(-maxCorr, math.Min(maxCorr, -slip))
			b.Prev = b.Prev.Sub(tan.Mul(corr * wheelTraction))
			b.Omega -= corr * (1 - wheelTraction) / (h * b.Radius)
		} else {
			corr := math.Max(-maxCorr, math.Min(maxCorr, -vt))
			b.Prev = b.Prev.Sub(tan.Mul(corr))
		}
		if mat.Drag > 0 {
			b.Prev = b.Prev.Add(b.Pos.Sub(b.Prev).Mul(mat.Drag * h))
		}
	}
}

// Center returns the mass-weighted centre of the given bodies.
func Center(bs []*Body) Vec {
	var c Vec
	var m float64
	for _, b := range bs {
		if b.InvMass == 0 {
			continue
		}
		w := 1 / b.InvMass
		c = c.Add(b.Pos.Mul(w))
		m += w
	}
	if m == 0 {
		return Vec{}
	}
	return c.Mul(1 / m)
}
