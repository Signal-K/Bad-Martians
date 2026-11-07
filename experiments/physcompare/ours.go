package main

import (
	"github.com/signal-k/haphazard-martians/game"
	"github.com/signal-k/haphazard-martians/physics"
)

type ours struct {
	w *physics.World
	r *game.Rover
}

func newOurs(t *physics.Terrain, d game.Design, x0 float64) Sim {
	w := physics.NewWorld()
	w.Terrain = t
	return &ours{w, game.SpawnRover(w, d, x0, t.HeightAt(x0))}
}

func (o *ours) Step(th float64) { o.r.SetThrottle(th); o.w.Step(dt) }

func (o *ours) Bodies() []Body {
	h := dt / float64(o.w.Substeps)
	out := make([]Body, len(o.r.Bodies))
	for i, b := range o.r.Bodies {
		v := b.Vel(h)
		out[i] = Body{b.Pos.X, b.Pos.Y, v.X, v.Y, b.Radius, 1 / b.InvMass, o.r.Cells[i].K}
	}
	return out
}

func (o *ours) Launch(vx, vy float64) {
	h := dt / float64(o.w.Substeps)
	for _, b := range o.r.Bodies {
		b.Prev = b.Pos.Sub(physics.V(vx, vy).Mul(h))
	}
}

// newOursTuned is the same solver with more constraint iterations: it shows
// how much stiffness is available just by spending CPU.
func newOursTuned(t *physics.Terrain, d game.Design, x0 float64) Sim {
	o := newOurs(t, d, x0).(*ours)
	o.w.Iterations *= 4
	return o
}
