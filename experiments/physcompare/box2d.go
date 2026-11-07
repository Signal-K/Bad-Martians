package main

import (
	"math"

	box2d "github.com/ByteArena/box2d"

	"github.com/signal-k/haphazard-martians/game"
	"github.com/signal-k/haphazard-martians/physics"
)

// b2 models the rover the Bad Piggies way: one rigid body per part (boxes for
// structure, circles for wheels), welded together, with revolute motor joints
// on the wheels. This is what a rigid-body engine is for, so it is tested in
// its natural form rather than imitating our lattice.
type b2 struct {
	w      *box2d.B2World
	bodies []*box2d.B2Body
	radius []float64
	mass   []float64
	wheel  []bool
	kinds  []game.Kind
	motors []*box2d.B2RevoluteJoint
}

const b2Group = -1 // negative group: rover parts never collide with each other

func newBox2D(t *physics.Terrain, d game.Design, x0 float64) Sim {
	w := box2d.MakeB2World(box2d.MakeB2Vec2(0, -gravity))
	sim := &b2{w: &w}
	// terrain: a chain of edges, friction per segment
	gb := w.CreateBody(&box2d.B2BodyDef{Type: box2d.B2BodyType.B2_staticBody, Active: true, AllowSleep: true, GravityScale: 1})
	for i := 0; i < t.Len()-1; i++ {
		e := box2d.MakeB2EdgeShape()
		e.Set(box2d.MakeB2Vec2(t.X0+float64(i)*t.DX, t.H[i]), box2d.MakeB2Vec2(t.X0+float64(i+1)*t.DX, t.H[i+1]))
		fd := box2d.MakeB2FixtureDef()
		fd.Shape, fd.Friction, fd.Filter = &e, t.Materials[t.Seg[i]].Friction, box2d.MakeB2Filter()
		gb.CreateFixtureFromDef(&fd)
	}
	minBottom := 1e9
	for _, cell := range d.Cells {
		p, _ := game.PartOf(cell.K)
		if v := float64(cell.Y) - p.Radius; v < minBottom {
			minBottom = v
		}
	}
	gy := t.HeightAt(x0)
	idx := map[[2]int]int{}
	for i, cell := range d.Cells {
		p, _ := game.PartOf(cell.K)
		bd := box2d.MakeB2BodyDef()
		bd.Type = box2d.B2BodyType.B2_dynamicBody
		bd.Position = box2d.MakeB2Vec2(x0+float64(cell.X), gy+0.05+float64(cell.Y)-minBottom)
		b := w.CreateBody(&bd)
		fd := box2d.MakeB2FixtureDef()
		fd.Filter = box2d.MakeB2Filter()
		fd.Filter.GroupIndex = b2Group
		fd.Friction = 0.4
		if p.Wheel {
			c := box2d.MakeB2CircleShape()
			c.M_radius = p.Radius
			fd.Shape, fd.Friction, fd.Density = &c, 1, p.Mass/(math.Pi*p.Radius*p.Radius)
		} else {
			half := 0.45
			if p.Kind == game.KMast {
				half = 0.3
			}
			pg := box2d.MakeB2PolygonShape()
			pg.SetAsBox(half, half)
			fd.Shape, fd.Density = &pg, p.Mass/(4*half*half)
		}
		b.CreateFixtureFromDef(&fd)
		sim.bodies = append(sim.bodies, b)
		sim.radius = append(sim.radius, p.Radius)
		sim.mass = append(sim.mass, p.Mass)
		sim.wheel = append(sim.wheel, p.Wheel)
		sim.kinds = append(sim.kinds, cell.K)
		idx[[2]int{cell.X, cell.Y}] = i
	}
	anchor := func(i, j int) box2d.B2Vec2 {
		a, b := sim.bodies[i].GetPosition(), sim.bodies[j].GetPosition()
		return box2d.MakeB2Vec2((a.X+b.X)/2, (a.Y+b.Y)/2)
	}
	// structure: weld every non-wheel pair that touches on the 4-neighbour grid
	for i, cell := range d.Cells {
		if sim.wheel[i] {
			continue
		}
		for _, o := range [][2]int{{1, 0}, {0, 1}} {
			if j, ok := idx[[2]int{cell.X + o[0], cell.Y + o[1]}]; ok && !sim.wheel[j] {
				wd := box2d.MakeB2WeldJointDef()
				wd.Initialize(sim.bodies[i], sim.bodies[j], anchor(i, j))
				w.CreateJoint(&wd)
			}
		}
	}
	// wheels: revolute joint at the wheel centre to the first structural neighbour
	for i, cell := range d.Cells {
		if !sim.wheel[i] {
			continue
		}
		p, _ := game.PartOf(cell.K)
		for _, o := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
			j, ok := idx[[2]int{cell.X + o[0], cell.Y + o[1]}]
			if !ok || sim.wheel[j] {
				continue
			}
			rd := box2d.MakeB2RevoluteJointDef()
			rd.Initialize(sim.bodies[j], sim.bodies[i], sim.bodies[i].GetPosition())
			if p.Motor {
				rd.EnableMotor, rd.MaxMotorTorque = true, 40
			}
			joint := w.CreateJoint(&rd)
			if p.Motor {
				sim.motors = append(sim.motors, joint.(*box2d.B2RevoluteJoint))
			}
			break
		}
	}
	return sim
}

func (s *b2) Step(th float64) {
	for _, m := range s.motors {
		m.SetMotorSpeed(-th * physics.MaxOmega)
	}
	s.w.Step(dt, 8, 3)
}

func (s *b2) Bodies() []Body {
	out := make([]Body, len(s.bodies))
	for i, b := range s.bodies {
		p, v := b.GetPosition(), b.GetLinearVelocity()
		out[i] = Body{p.X, p.Y, v.X, v.Y, s.radius[i], s.mass[i], s.kinds[i]}
	}
	return out
}

func (s *b2) Launch(vx, vy float64) {
	for _, b := range s.bodies {
		b.SetLinearVelocity(box2d.MakeB2Vec2(vx, vy))
	}
}
