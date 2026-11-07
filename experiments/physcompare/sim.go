// Command physcompare runs the same rover-and-terrain scenarios against three
// physics solvers and prints a comparison. It is an experiment, not game code.
package main

import (
	"encoding/binary"
	"hash/fnv"
	"math"

	"github.com/signal-k/haphazard-martians/game"
	"github.com/signal-k/haphazard-martians/physics"
)

const (
	dt      = 1.0 / 60.0
	gravity = 3.71
)

// Sim is one solver running one rover on one terrain.
type Sim interface {
	Step(throttle float64)
	Bodies() []Body // every rover body, with mass
	Launch(vx, vy float64)
}

// Body is the solver-independent view of a body.
type Body struct {
	X, Y, VX, VY float64
	R, Mass      float64 // collision radius (for penetration checks) and mass
	Kind         game.Kind
}

// Factory builds a Sim for a design placed at x0 on the terrain.
type Factory struct {
	Name  string
	Build func(t *physics.Terrain, d game.Design, x0 float64) Sim
}

func center(bs []Body) (x, y float64) {
	var m float64
	for _, b := range bs {
		x += b.X * b.Mass
		y += b.Y * b.Mass
		m += b.Mass
	}
	return x / m, y / m
}

func energy(bs []Body) float64 {
	e := 0.0
	for _, b := range bs {
		e += b.Mass * (0.5*(b.VX*b.VX+b.VY*b.VY) + gravity*b.Y)
	}
	return e
}

func speedMax(bs []Body) float64 {
	m := 0.0
	for _, b := range bs {
		m = math.Max(m, math.Hypot(b.VX, b.VY))
	}
	return m
}

// stateHash digests every body position bit for bit.
func stateHash(h *uint64, bs []Body) {
	f := fnv.New64a()
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], *h)
	f.Write(buf[:])
	for _, b := range bs {
		for _, v := range [...]float64{b.X, b.Y} {
			binary.LittleEndian.PutUint64(buf[:], math.Float64bits(v))
			f.Write(buf[:])
		}
	}
	*h = f.Sum64()
}

// Terrain builders -----------------------------------------------------------

func classMats() []physics.Material {
	ms := make([]physics.Material, game.NumClasses)
	for i := range ms {
		ms[i] = game.Classes[i].Mat
	}
	return ms
}

// mkTerrain samples height(x) every 0.5 m from x0, all one class.
func mkTerrain(x0, x1 float64, class game.Class, height func(x float64) float64) *physics.Terrain {
	const dx = 0.5
	n := int((x1-x0)/dx) + 1
	t := &physics.Terrain{X0: x0, DX: dx, Materials: classMats()}
	for i := 0; i < n; i++ {
		t.H = append(t.H, height(x0+float64(i)*dx))
	}
	t.Seg = make([]uint8, n-1)
	for i := range t.Seg {
		t.Seg[i] = uint8(class)
	}
	return t
}

func ramp(deg float64) func(x float64) float64 {
	k := math.Tan(deg * math.Pi / 180)
	return func(x float64) float64 {
		switch {
		case x < 10:
			return 0
		case x < 60:
			return (x - 10) * k
		}
		return 50 * k
	}
}
