package game

import (
	"math"

	"github.com/signal-k/haphazard-martians/engine"
	"github.com/signal-k/haphazard-martians/physics"
)

type Outcome int

const (
	Running Outcome = iota
	Won
	Fell
	Stuck
	TimedOut
	Scrapped
)

func (o Outcome) Message() string {
	return [...]string{"", "Navcam reached the finish!", "Rover lost in a crevasse.", "Rover is stuck.", "Out of time.", "Rover scrapped. Its parts are gone."}[o]
}

const (
	sampleRadius = 0.9
	// StuckSeconds without progress, while trying to drive, before the player is
	// offered the Restart button. Nothing ends the run for being stuck: the
	// player decides, and may still wriggle free.
	StuckSeconds = 4.0
	maxSeconds   = 120.0
)

// Run is the headless rules of one attempt: it owns the physics world and
// decides win/lose, so scenes and tests share it.
type Run struct {
	Level    *Level
	World    *physics.World
	Rover    *Rover
	Time     float64
	Got      []bool
	BestX    float64
	stuckFor float64
	lastC    engine.Vec2 // rover centre on the previous step
	Outcome  Outcome
}

func NewRun(l *Level, d Design) *Run {
	w := physics.NewWorld()
	w.Terrain = l.Terrain
	x := l.StartX + 2
	r := &Run{Level: l, World: w, Got: make([]bool, len(l.Samples))}
	r.Rover = SpawnRover(w, d, x, l.Terrain.HeightAt(x))
	r.lastC = r.Rover.Center()
	r.BestX = r.lastC.X
	return r
}

// IsStuck reports whether the rover has made no progress for StuckSeconds of
// trying, which is when the Restart button appears.
func (r *Run) IsStuck() bool { return r.Outcome == Running && r.stuckFor >= StuckSeconds }

// Scrap blows the rover up: every joint breaks and the parts fly apart. The
// attempt fails, so an expedition loses the parts the rover was built from.
func (r *Run) Scrap() {
	if r.Outcome != Running {
		return
	}
	r.Outcome = Scrapped
	c := r.Rover.Center()
	r.World.Cons = r.World.Cons[:0]
	r.Rover.Links = nil
	rg := newRng(uint64(len(r.Rover.Bodies))*977 + uint64(r.Time*60))
	h := 1.0 / 60 / float64(max(1, r.World.Substeps))
	for _, b := range r.Rover.Bodies {
		d := b.Pos.Sub(c)
		if l := d.Len(); l > 1e-6 {
			d = d.Mul(1 / l)
		} else {
			d = physics.V(0, 1)
		}
		v := d.Mul(rg.rangef(5, 9)).Add(physics.V(rg.rangef(-1.5, 1.5), rg.rangef(4, 7)))
		b.Prev = b.Pos.Sub(v.Mul(h))
		if b.Wheel {
			b.Omega = rg.rangef(-20, 20)
		}
	}
}

// Debris keeps the wreck flying after the rover is scrapped.
func (r *Run) Debris(dt float64) {
	if r.Outcome == Scrapped {
		r.World.Step(dt)
	}
}

func (r *Run) SamplesGot() int {
	n := 0
	for _, g := range r.Got {
		if g {
			n++
		}
	}
	return n
}

// Step advances one fixed step with throttle in [-1,1].
func (r *Run) Step(dt, throttle float64) {
	if r.Outcome != Running {
		return
	}
	r.Time += dt
	r.Rover.SetThrottle(throttle)
	r.World.Step(dt)

	for i, s := range r.Level.Samples {
		if r.Got[i] {
			continue
		}
		for _, b := range r.Rover.Bodies {
			if b.Pos.Sub(s).Len() < sampleRadius+b.Radius {
				r.Got[i] = true
				break
			}
		}
	}

	c := r.Rover.Center()
	if m := r.Rover.Mast(); m != nil && m.Pos.X >= r.Level.GoalX {
		r.Outcome = Won
		return
	}
	if c.Y < r.Level.Terrain.HeightAt(c.X)-12 {
		r.Outcome = Fell
		return
	}
	if c.X > r.BestX+0.5 {
		r.BestX, r.stuckFor = c.X, 0
	} else if throttle != 0 && c.Sub(r.lastC).Len() < 1.5*dt { // trying, but barely moving
		r.stuckFor += dt
	}
	r.lastC = c
	if r.Time > maxSeconds {
		r.Outcome = TimedOut
	}
}

// Stars: 1 finish, 2 all samples, 3 under par time (needs all prior).
func (r *Run) Stars() int {
	if r.Outcome != Won {
		return 0
	}
	s := 1
	if r.SamplesGot() == len(r.Got) {
		s++
		if r.Time <= r.Level.Par {
			s++
		}
	}
	return s
}

func (r *Run) Progress() float64 {
	span := r.Level.GoalX - r.Level.StartX
	return math.Max(0, math.Min(1, (r.Rover.Center().X-r.Level.StartX)/span))
}
