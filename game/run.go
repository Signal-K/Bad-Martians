package game

import (
	"math"

	"github.com/signal-k/haphazard-martians/physics"
)

type Outcome int

const (
	Running Outcome = iota
	Won
	Fell
	Stuck
	TimedOut
)

func (o Outcome) Message() string {
	return [...]string{"", "Navcam reached the finish!", "Rover lost in a crevasse.", "Rover is stuck.", "Out of time."}[o]
}

const (
	sampleRadius = 0.9
	stuckSeconds = 7.0
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
	Outcome  Outcome
}

func NewRun(l *Level, d Design) *Run {
	w := physics.NewWorld()
	w.Terrain = l.Terrain
	x := l.StartX + 2
	r := &Run{Level: l, World: w, Got: make([]bool, len(l.Samples))}
	r.Rover = SpawnRover(w, d, x, l.Terrain.HeightAt(x))
	r.BestX = r.Rover.Center().X
	return r
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
	} else if throttle != 0 { // only count time the player is trying
		r.stuckFor += dt
		if r.stuckFor > stuckSeconds {
			r.Outcome = Stuck
			return
		}
	}
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
