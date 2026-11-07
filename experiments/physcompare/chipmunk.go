package main

import (
	"github.com/jakecoffman/cp/v2"

	"github.com/signal-k/haphazard-martians/game"
	"github.com/signal-k/haphazard-martians/physics"
)

// chipmunk models the rover the way our own solver does: round bodies joined
// by rigid distance links on the 8-neighbour lattice, so the only difference is
// the solver (sequential impulses instead of position projection).
type chipmunk struct {
	sp     *cp.Space
	bodies []*cp.Body
	radius []float64
	wheel  []bool
	kinds  []game.Kind
	motors []*cp.Constraint
}

func newChipmunk(t *physics.Terrain, d game.Design, x0 float64) Sim {
	sp := cp.NewSpace()
	sp.SetGravity(cp.Vector{X: 0, Y: -gravity})
	sp.Iterations = 20
	sp.SetCollisionSlop(0.01)
	ground := sp.StaticBody
	for i := 0; i < t.Len()-1; i++ {
		a := cp.Vector{X: t.X0 + float64(i)*t.DX, Y: t.H[i]}
		b := cp.Vector{X: t.X0 + float64(i+1)*t.DX, Y: t.H[i+1]}
		s := sp.AddShape(cp.NewSegment(ground, a, b, 0.02))
		s.SetFriction(t.Materials[t.Seg[i]].Friction)
	}
	c := &chipmunk{sp: sp}
	idx := map[[2]int]int{}
	minBottom := 1e9
	for _, cell := range d.Cells {
		p, _ := game.PartOf(cell.K)
		if v := float64(cell.Y) - p.Radius; v < minBottom {
			minBottom = v
		}
	}
	gy := t.HeightAt(x0)
	filter := cp.NewShapeFilter(1, cp.ALL_CATEGORIES, cp.ALL_CATEGORIES)
	for i, cell := range d.Cells {
		p, _ := game.PartOf(cell.K)
		moment := cp.MomentForCircle(p.Mass, 0, p.Radius, cp.Vector{})
		if !p.Wheel {
			moment = cp.INFINITY // structure doesn't spin, so a wheel motor reacts against the chassis
		}
		b := sp.AddBody(cp.NewBody(p.Mass, moment))
		b.SetPosition(cp.Vector{X: x0 + float64(cell.X), Y: gy + 0.05 + float64(cell.Y) - minBottom})
		s := sp.AddShape(cp.NewCircle(b, p.Radius, cp.Vector{}))
		s.SetFilter(filter)
		s.SetFriction(0.4)
		if p.Wheel {
			s.SetFriction(1)
		}
		c.bodies, c.radius, c.wheel, c.kinds = append(c.bodies, b), append(c.radius, p.Radius), append(c.wheel, p.Wheel), append(c.kinds, cell.K)
		idx[[2]int{cell.X, cell.Y}] = i
	}
	for i, cell := range d.Cells {
		for _, o := range [][2]int{{1, 0}, {0, 1}, {1, 1}, {-1, 1}} {
			if j, ok := idx[[2]int{cell.X + o[0], cell.Y + o[1]}]; ok {
				sp.AddConstraint(cp.NewPinJoint(c.bodies[i], c.bodies[j], cp.Vector{}, cp.Vector{}))
			}
		}
	}
	// motors: spin each motor wheel relative to a non-wheel neighbour
	for i, cell := range d.Cells {
		if p, _ := game.PartOf(cell.K); !p.Motor {
			continue
		}
		for _, o := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {1, 1}, {-1, 1}, {0, -1}} {
			if j, ok := idx[[2]int{cell.X + o[0], cell.Y + o[1]}]; ok && !c.wheel[j] {
				m := sp.AddConstraint(cp.NewSimpleMotor(c.bodies[j], c.bodies[i], 0))
				m.SetMaxForce(40)
				c.motors = append(c.motors, m)
				break
			}
		}
	}
	return c
}

func (c *chipmunk) Step(th float64) {
	for _, m := range c.motors {
		m.Class.(*cp.SimpleMotor).Rate = th * physics.MaxOmega
	}
	c.sp.Step(dt)
}

func (c *chipmunk) Bodies() []Body {
	out := make([]Body, len(c.bodies))
	for i, b := range c.bodies {
		p, v := b.Position(), b.Velocity()
		out[i] = Body{p.X, p.Y, v.X, v.Y, c.radius[i], b.Mass(), c.kinds[i]}
	}
	return out
}

func (c *chipmunk) Launch(vx, vy float64) {
	for _, b := range c.bodies {
		b.SetVelocity(vx, vy)
	}
}
