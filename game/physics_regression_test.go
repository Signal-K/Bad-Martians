package game

import (
	"math"
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
	"github.com/signal-k/haphazard-martians/physics"
)

// These scenarios came out of the solver comparison (experiments/physcompare).
// They guard the behaviours that make the sim feel solid, whatever solver
// sits underneath.

func flat(class Class, x1 float64, h func(x float64) float64) *physics.Terrain {
	const dx = 0.5
	t := &physics.Terrain{X0: -10, DX: dx}
	for _, c := range Classes {
		t.Materials = append(t.Materials, c.Mat)
	}
	n := int((x1+10)/dx) + 1
	for i := 0; i < n; i++ {
		t.H = append(t.H, h(-10+float64(i)*dx))
	}
	t.Seg = make([]uint8, n-1)
	for i := range t.Seg {
		t.Seg[i] = uint8(class)
	}
	return t
}

func rover(t *physics.Terrain, d Design, x float64) (*physics.World, *Rover) {
	w := physics.NewWorld()
	w.Terrain = t
	return w, SpawnRover(w, d, x, t.HeightAt(x))
}

func maxSpeed(w *physics.World) float64 {
	m := 0.0
	h := engine.FixedDT / float64(w.Substeps)
	for _, b := range w.Bodies {
		m = math.Max(m, b.Vel(h).Len())
	}
	return m
}

func TestRoverRestsStillOnFlatRock(t *testing.T) {
	w, _ := rover(flat(Bedrock, 80, func(float64) float64 { return 0 }), StarterDesign(), 5)
	for i := 0; i < 12*60; i++ {
		w.Step(engine.FixedDT)
	}
	if v := maxSpeed(w); v > 0.005 {
		t.Fatalf("rover still moving at %.4f m/s after 12 s", v)
	}
	for _, b := range w.Bodies {
		if b.Pos.Y < b.Radius-0.02 {
			t.Fatalf("body sunk into the rock: y=%.3f r=%.2f", b.Pos.Y, b.Radius)
		}
	}
}

func TestNoTunnellingFromGreatHeight(t *testing.T) {
	// Fall 1000 m (about 86 m/s on impact) onto thin terrain.
	ter := flat(Bedrock, 1800, func(x float64) float64 {
		if x < 8 {
			return 1000
		}
		return 0
	})
	w, r := rover(ter, StarterDesign(), 3)
	h := engine.FixedDT / float64(w.Substeps)
	for _, b := range r.Bodies {
		b.Prev = b.Pos.Sub(physics.V(14, -10).Mul(h))
	}
	for i := 0; i < 20*60; i++ {
		w.Step(engine.FixedDT)
		for _, b := range r.Bodies {
			if b.Pos.Y < ter.HeightAt(b.Pos.X)-0.05 {
				t.Fatalf("step %d: body passed through the ground", i)
			}
		}
	}
}

func TestSolverIsRepeatable(t *testing.T) {
	run := func() []physics.Vec {
		ter := flat(Soil, 200, func(x float64) float64 { return math.Max(0, x-10) * 0.21 })
		w, r := rover(ter, referenceDesign(), 2)
		r.SetThrottle(1)
		for i := 0; i < 20*60; i++ {
			w.Step(engine.FixedDT)
		}
		var p []physics.Vec
		for _, b := range r.Bodies {
			p = append(p, b.Pos)
		}
		return p
	}
	a, b := run(), run()
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("body %d differs between identical runs: %v vs %v", i, a[i], b[i])
		}
	}
}

func TestReferenceRoverClimbs(t *testing.T) {
	for _, deg := range []float64{10, 15, 20} {
		k := math.Tan(deg * math.Pi / 180)
		ter := flat(Soil, 80, func(x float64) float64 {
			switch {
			case x < 10:
				return 0
			case x < 60:
				return (x - 10) * k
			}
			return 50 * k
		})
		w, r := rover(ter, referenceDesign(), 2)
		r.SetThrottle(1)
		ok := false
		for i := 0; i < 30*60 && !ok; i++ {
			w.Step(engine.FixedDT)
			ok = r.Center().X > 40
		}
		t.Logf("%v°: reached 40 m = %v", deg, ok)
		if deg <= 15 && !ok {
			t.Errorf("reference rover should climb a %v° soil slope", deg)
		}
	}
}
