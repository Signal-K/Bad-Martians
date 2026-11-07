package main

import (
	"fmt"
	"math"
	"time"

	"github.com/signal-k/haphazard-martians/game"
	"github.com/signal-k/haphazard-martians/physics"
)

type Metrics map[string]string

func starter() game.Design { return game.StarterDesign() }

func isBad(bs []Body) bool {
	for _, b := range bs {
		if math.IsNaN(b.X) || math.IsNaN(b.Y) || math.Abs(b.X) > 1e6 || math.Abs(b.Y) > 1e6 {
			return true
		}
	}
	return false
}

// settle: does a rover dropped on flat rock come to rest cleanly?
func settle(f Factory) Metrics {
	t := mkTerrain(-10, 80, game.Bedrock, func(float64) float64 { return 0 })
	s := f.Build(t, starter(), 5)
	const steps = 12 * 60
	settledAt := -1.0
	var e0, e1 float64
	var jitter float64
	var jn int
	for i := 0; i < steps; i++ {
		s.Step(0)
		bs := s.Bodies()
		sp := speedMax(bs)
		now := float64(i+1) * dt
		if sp < 0.05 {
			if settledAt < 0 {
				settledAt = now
			}
		} else {
			settledAt = -1
		}
		if i == steps-3*60 {
			e0 = energy(bs)
		}
		if i >= steps-3*60 {
			jitter += sp
			jn++
		}
		e1 = energy(bs)
	}
	bs := s.Bodies()
	pen := 0.0
	for _, b := range bs {
		pen = math.Max(pen, b.R-b.Y) // wheels/frames: how far below "resting on y=0"
	}
	m := Metrics{}
	if settledAt < 0 {
		m["settles"] = "never"
	} else {
		m["settles"] = fmt.Sprintf("%.1f s", settledAt)
	}
	m["rest jitter"] = fmt.Sprintf("%.1f mm/s", jitter/float64(jn)*1000)
	m["energy drift"] = fmt.Sprintf("%.2f J / 3 s", math.Abs(e1-e0))
	m["sink at rest"] = fmt.Sprintf("%.0f mm", math.Max(pen, 0)*1000)
	return m
}

// slope: steepest soil slope the starter rover climbs under full throttle.
func slope(f Factory) Metrics {
	best := 0
	var t20 string = "no"
	for deg := 10; deg <= 55; deg += 5 {
		t := mkTerrain(-10, 80, game.Soil, ramp(float64(deg)))
		s := f.Build(t, starter(), 2)
		ok := false
		for i := 0; i < 25*60; i++ {
			s.Step(1)
			bs := s.Bodies()
			if isBad(bs) {
				break
			}
			if x, _ := center(bs); x >= 40 {
				ok = true
				if deg == 20 {
					t20 = fmt.Sprintf("%.1f s", float64(i+1)*dt)
				}
				break
			}
		}
		if ok {
			best = deg
		}
	}
	return Metrics{"steepest climb": fmt.Sprintf("%d°", best), "20° climb to 40 m": t20}
}

// highspeed: fall from a plateau at speed; count steps where a body centre is
// below the surface (it tunnelled) and whether the rover ends up on top.
func highspeed(f Factory, height float64) Metrics {
	t := mkTerrain(-10, 1800, game.Bedrock, func(x float64) float64 {
		if x < 8 {
			return height
		}
		return 0
	})
	s := f.Build(t, starter(), 3)
	s.Launch(14, -10)
	through := 0
	maxSp := 0.0
	for i := 0; i < 20*60; i++ {
		s.Step(0)
		bs := s.Bodies()
		if isBad(bs) {
			return Metrics{"tunnelled steps": "blew up"}
		}
		maxSp = math.Max(maxSp, speedMax(bs))
		for _, b := range bs {
			if b.Y < t.HeightAt(b.X)-0.05 {
				through++
				break
			}
		}
	}
	bs := s.Bodies()
	on := true
	for _, b := range bs {
		if b.Y < t.HeightAt(b.X)-0.05 {
			on = false
		}
	}
	return Metrics{
		"tunnelled steps": fmt.Sprint(through),
		"ends on surface": map[bool]string{true: "yes", false: "NO"}[on],
		"peak speed":      fmt.Sprintf("%.0f m/s", maxSp),
	}
}

// reference is the sturdy all-wheel-drive rover every shipped level is
// designed to be beatable with.
func reference() game.Design {
	return game.Design{Cells: []game.Cell{
		{X: 0, Y: 0, K: game.KMotor}, {X: 1, Y: 0, K: game.KFrame}, {X: 2, Y: 0, K: game.KFrame}, {X: 3, Y: 0, K: game.KMotor},
		{X: 0, Y: 1, K: game.KFrame}, {X: 1, Y: 1, K: game.KFrame}, {X: 2, Y: 1, K: game.KMast}, {X: 3, Y: 1, K: game.KFrame},
	}}
}

// levelRun drives a design over a shipped level with throttle held, using the
// game's own win/stuck/fall rules, and reports the outcome.
func levelRun(f Factory, d game.Design, id int) string {
	l := game.GetLevel(id)
	x0 := l.StartX + 2
	s := f.Build(l.Terrain, d, x0)
	bestX, stuck := x0, 0.0
	for i := 0; i < 120*60; i++ {
		s.Step(1)
		bs := s.Bodies()
		if isBad(bs) {
			return "blew up"
		}
		cx, cy := center(bs)
		if cy < l.Terrain.HeightAt(cx)-12 {
			return "fell"
		}
		for _, b := range bs {
			if b.Kind == game.KMast && b.X >= l.GoalX {
				return fmt.Sprintf("%.1f s (par %.0f)", float64(i+1)*dt, l.Par)
			}
		}
		if cx > bestX+0.5 {
			bestX, stuck = cx, 0
		} else if stuck += dt; stuck > 7 {
			return fmt.Sprintf("stuck at %.0f m", cx-l.StartX)
		}
	}
	return "timeout"
}

func levels(f Factory) Metrics {
	m := Metrics{"L1 starter rover": levelRun(f, starter(), 1)}
	for id := 1; id <= 5; id++ {
		m[fmt.Sprintf("L%d reference rover", id)] = levelRun(f, reference(), id)
	}
	return m
}

// determinism: same inputs twice, then the digest for cross-platform comparison.
func determinism(f Factory) (Metrics, uint64) {
	run := func() uint64 {
		// A synthetic ramp, not a shipped level: this isolates the solver from level generation.
		t := mkTerrain(-10, 200, game.Soil, ramp(12))
		s := f.Build(t, reference(), 2)
		var h uint64 = 14695981039346656037
		for i := 0; i < 20*60; i++ {
			s.Step(1)
			stateHash(&h, s.Bodies())
		}
		return h
	}
	a, b := run(), run()
	return Metrics{"repeatable": map[bool]string{true: "bit-identical", false: "DIFFERS"}[a == b], "digest": fmt.Sprintf("%016x", a)}, a
}

// rigidity: a long beam with a wheel at each end. How much does it sag?
func rigidity(f Factory) Metrics {
	var d game.Design
	d.Cells = append(d.Cells, game.Cell{X: 0, Y: 0, K: game.KWheel}, game.Cell{X: 9, Y: 0, K: game.KMotor})
	for x := 1; x <= 8; x++ {
		d.Cells = append(d.Cells, game.Cell{X: x, Y: 0, K: game.KFrame})
	}
	d.Cells = append(d.Cells, game.Cell{X: 4, Y: 1, K: game.KMast})
	t := mkTerrain(-10, 80, game.Bedrock, func(float64) float64 { return 0 })
	s := f.Build(t, d, 5)
	for i := 0; i < 10*60; i++ {
		s.Step(0)
	}
	bs := s.Bodies()
	// find end wheels (first two cells) and the beam frames
	ends := [2]Body{}
	var wi int
	var beam []Body
	for i, c := range d.Cells {
		_ = i
		_ = c
	}
	// bodies align with d.Cells except ours (sorted); classify by Wheel flag + x order.
	for _, b := range bs {
		if b.Kind == game.KWheel || b.Kind == game.KMotor {
			ends[wi%2] = b
			wi++
		}
	}
	lo, hi := ends[0], ends[1]
	if lo.X > hi.X {
		lo, hi = hi, lo
	}
	sag := 0.0
	for _, b := range bs {
		if b.Kind != game.KFrame || b.X <= lo.X || b.X >= hi.X {
			continue
		}
		beam = append(beam, b)
		u := (b.X - lo.X) / (hi.X - lo.X)
		line := lo.Y + u*(hi.Y-lo.Y)
		sag = math.Max(sag, math.Abs(b.Y-line))
	}
	return Metrics{"beam sag": fmt.Sprintf("%.0f mm", sag*1000), "beam parts": fmt.Sprint(len(beam))}
}

// perf: cost per step for a small and a full-grid rover; plus stability of the big one.
func perf(f Factory) Metrics {
	big := game.Design{}
	for y := 0; y < game.GridH; y++ {
		for x := 0; x < game.GridW; x++ {
			k := game.KFrame
			switch {
			case y == 0 && (x == 0 || x == game.GridW-1):
				k = game.KMotor
			case y == 0 && (x == 4 || x == 8):
				k = game.KWheel
			case y == game.GridH-1 && x == 6:
				k = game.KMast
			}
			big.Cells = append(big.Cells, game.Cell{X: x, Y: y, K: k})
		}
	}
	t := mkTerrain(-10, 200, game.Soil, func(float64) float64 { return 0 })
	time1 := func(d game.Design) (float64, bool, float64) {
		s := f.Build(t, d, 2)
		for i := 0; i < 60; i++ {
			s.Step(1)
		}
		start := time.Now()
		const n = 600
		for i := 0; i < n; i++ {
			s.Step(1)
		}
		el := time.Since(start)
		bs := s.Bodies()
		cx, _ := center(bs)
		return float64(el.Microseconds()) / n, !isBad(bs) && speedMax(bs) < 15, cx - 2
	}
	us6, _, _ := time1(starter())
	us84, ok, dx := time1(big)
	return Metrics{
		"6-body step":  fmt.Sprintf("%.0f µs", us6),
		"84-body step": fmt.Sprintf("%.0f µs", us84),
		"84-body sane": map[bool]string{true: "yes", false: "NO"}[ok] + fmt.Sprintf(" (moved %.0f m)", dx),
	}
}

var _ = physics.V
