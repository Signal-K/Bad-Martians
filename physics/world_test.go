package physics

import (
	"math"
	"testing"
)

func flat(n int, y float64, mu float64) *Terrain {
	h := make([]float64, n)
	for i := range h {
		h[i] = y
	}
	return &Terrain{X0: -10, DX: 1, H: h, Seg: make([]uint8, n-1), Materials: []Material{{Friction: mu}}}
}

func run(w *World, secs float64) {
	for i := 0; i < int(secs*60); i++ {
		w.Step(1.0 / 60)
	}
}

func TestBallRestsOnGround(t *testing.T) {
	w := NewWorld()
	w.Terrain = flat(200, 0, 0.8)
	b := w.AddBody(&Body{Pos: V(0, 5), Radius: 0.5, InvMass: 1, Friction: 1})
	run(w, 4)
	if math.Abs(b.Pos.Y-0.5) > 0.02 {
		t.Fatalf("ball should rest at y=0.5, got %.3f", b.Pos.Y)
	}
}

func TestSlopeSlidesOnlyWhenSteepEnough(t *testing.T) {
	slope := func(angleDeg, mu float64) float64 {
		w := NewWorld()
		tr := flat(200, 0, mu)
		for i := range tr.H {
			tr.H[i] = -math.Tan(angleDeg*math.Pi/180) * (tr.X0 + float64(i)*tr.DX)
		}
		w.Terrain = tr
		x := 0.0
		b := w.AddBody(&Body{Pos: V(x, tr.HeightAt(x)+0.6), Radius: 0.5, InvMass: 1, Friction: 1})
		run(w, 3)
		return b.Pos.X - x
	}
	if d := slope(10, 0.8); math.Abs(d) > 0.2 { // tan10=.18 < mu: sticks
		t.Fatalf("box should hold on shallow slope, moved %.2f", d)
	}
	if d := slope(45, 0.3); d < 3 { // tan45=1 > mu: slides
		t.Fatalf("ball should slide on steep slope, moved %.2f", d)
	}
}

func car(w *World, x float64, motor bool) (*Body, *Body) {
	// two wheels + one frame node on top, triangulated
	a := w.AddBody(&Body{Pos: V(x, 0.6), Radius: 0.5, InvMass: 1, Friction: 1, Wheel: true, Motor: motor})
	b := w.AddBody(&Body{Pos: V(x+1, 0.6), Radius: 0.5, InvMass: 1, Friction: 1, Wheel: true, Motor: motor})
	c := w.AddBody(&Body{Pos: V(x+0.5, 1.5), Radius: 0.3, InvMass: 1, Friction: .4})
	w.Connect(a, b, 1)
	w.Connect(a, c, 1)
	w.Connect(b, c, 1)
	a.Throttle, b.Throttle = 1, 1
	return a, c
}

func TestMotorDrivesCar(t *testing.T) {
	w := NewWorld()
	w.Terrain = flat(400, 0, 0.9)
	a, _ := car(w, 0, true)
	run(w, 5)
	if a.Pos.X < 8 {
		t.Fatalf("powered car should travel, x=%.2f", a.Pos.X)
	}
}

func TestUnpoweredCarStays(t *testing.T) {
	w := NewWorld()
	w.Terrain = flat(400, 0, 0.9)
	a, _ := car(w, 0, false)
	run(w, 5)
	if math.Abs(a.Pos.X) > 0.3 {
		t.Fatalf("unpowered car drifted to %.2f", a.Pos.X)
	}
}

func TestSandIsSlowerThanBedrock(t *testing.T) {
	dist := func(mu, drag float64) float64 {
		w := NewWorld()
		tr := flat(400, 0, mu)
		tr.Materials[0].Drag = drag
		w.Terrain = tr
		a, _ := car(w, 0, true)
		run(w, 5)
		return a.Pos.X
	}
	if dist(0.35, 0.8) >= dist(1.1, 0) {
		t.Fatal("sand should be slower than bedrock")
	}
}

func TestDeterministic(t *testing.T) {
	mk := func() float64 {
		w := NewWorld()
		w.Terrain = flat(400, 0, 0.9)
		a, c := car(w, 0, true)
		run(w, 3)
		return a.Pos.X*1000 + c.Pos.Y
	}
	if mk() != mk() {
		t.Fatal("simulation not deterministic")
	}
}
