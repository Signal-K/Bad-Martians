package game

import (
	"strings"
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
	"github.com/signal-k/haphazard-martians/physics"
)

func settle(d Design, secs float64) Pose {
	l := GetLevel(1)
	w := physics.NewWorld()
	w.Terrain = l.Terrain
	x := l.StartX + 2
	r := SpawnRover(w, d, x, l.Terrain.HeightAt(x))
	for t := 0.0; t < secs; t += engine.FixedDT {
		w.Step(engine.FixedDT)
	}
	return PoseOf(r)
}

func TestTestDropReadsBendAndTilt(t *testing.T) {
	sp := settle(StarterDesign(), 2)
	t.Logf("starter: tilt %.1f°, bend %.3f m", sp.Tilt*57.3, sp.Bend)
	if p := sp; p.Bend > bendLimit {
		t.Fatalf("braced starter rover should hold its shape, bend %.3f", p.Bend)
	}
	// one long unbraced row with a tall mast tower at one end sags
	flat := Design{}
	flat.Set(0, 0, KWheel)
	for x := 1; x < 9; x++ {
		flat.Set(x, 0, KFrame)
	}
	flat.Set(9, 0, KMotor)
	flat.Set(4, 1, KFrame)
	flat.Set(4, 2, KMast)
	p := settle(flat, 2)
	t.Logf("flat span: tilt %.1f°, bend %.3f m", p.Tilt*57.3, p.Bend)
	if p.Bend <= bendLimit {
		t.Fatalf("an unbraced 10 m span should read as bending, bend %.3f", p.Bend)
	}
	// a rover with a big wheel on one end only sits tilted
	lop := Design{}
	lop.Set(0, 0, KFrame)
	lop.Set(1, 0, KFrame)
	lop.Set(0, 1, KFrame)
	lop.Set(1, 1, KMast)
	lop.Set(2, 0, KMotor)
	if p := settle(lop, 2); p.Tilt*57.3 < 2 && p.Tilt*57.3 > -2 {
		t.Fatalf("frames (r 0.3) at the back and a wheel (r 0.5) at the front should tilt, got %.1f°", p.Tilt*57.3)
	}
}

func TestTrayDragSwipeAndQuantity(t *testing.T) {
	h := newHarness(t)
	ses := NewSession(GetLevel(1))
	ses.Design = Design{}
	h.g.Load(SceneBuild, ses)
	h.frames(2, engine.Input{})
	if n := CanAdd(ses.Design, KMotor, ses.Budget()); n != ses.Budget()/4 {
		t.Fatalf("motor quantity %d, want %d", n, ses.Budget()/4)
	}
	// drag the motor tile up onto grid cell 5,0
	m := h.button("Motor wheel")
	from := engine.V(m.X+m.W/2, m.Y+m.H/2)
	to := engine.V(bGX+5*bCell+bCell/2, bGY+float64(GridH-1)*bCell+bCell/2)
	h.frames(1, engine.Input{Pointer: from, Down: true})
	h.frames(1, engine.Input{Pointer: engine.V(from.X, 300), Down: true})
	h.frames(1, engine.Input{Pointer: to, Down: true})
	h.frames(1, engine.Input{Pointer: to})
	if k, ok := ses.Design.At(5, 0); !ok || k != KMotor {
		t.Fatalf("drag should drop a motor at 5,0: %+v", ses.Design)
	}
	if CanAdd(ses.Design, KMast, ses.Budget()) != 1 {
		t.Fatal("mast quantity is 1 before one is placed")
	}
	// swipe left across the tray steps the selection from motor to mast
	f := h.button("Frame")
	y := f.Y + f.H/2
	h.frames(1, engine.Input{Pointer: engine.V(600, y), Down: true})
	h.frames(1, engine.Input{Pointer: engine.V(450, y), Down: true})
	h.frames(1, engine.Input{Pointer: engine.V(450, y)})
	h.click(bGX+bCell/2, bGY+bCell/2) // top-left cell, 0,4
	if k, ok := ses.Design.At(0, GridH-1); !ok || k != KMast {
		t.Fatalf("after swiping, tapping the grid should place a mast: %+v", ses.Design)
	}
	if CanAdd(ses.Design, KMast, ses.Budget()) != 0 {
		t.Fatal("mast quantity is 0 once placed")
	}
}

// dropVerdict settles d on Mission 1's start and reads the bay's verdict.
func dropVerdict(d Design) (string, engine.Color) {
	l := GetLevel(1)
	w := physics.NewWorld()
	w.Terrain = l.Terrain
	x := l.StartX + 2
	r := SpawnRover(w, d, x, l.Terrain.HeightAt(x))
	for t := 0.0; t < 2; t += engine.FixedDT {
		w.Step(engine.FixedDT)
	}
	return testVerdict(r, l.Terrain, 2)
}

// The playtest jumble: motor wheels buried inside the frame. It stalls on the
// course, so the test drop must not call it fine.
func TestTestDropFlagsMotorsOffTheGround(t *testing.T) {
	msg, col := dropVerdict(StarterDesign())
	t.Log("starter:", msg)
	if col != colGood {
		t.Fatalf("starter rover should pass the test drop, got %q", msg)
	}
	jumble := Design{Cells: []Cell{
		{0, 3, KFrame}, {1, 3, KFrame}, {2, 3, KWheel},
		{0, 2, KFrame}, {1, 2, KMotor}, {2, 2, KFrame},
		{0, 1, KFrame}, {1, 1, KMotor}, {2, 1, KFrame}, {3, 1, KMast},
		{0, 0, KWheel}, {1, 0, KFrame}, {2, 0, KFrame},
	}}
	msg, col = dropVerdict(jumble)
	t.Log("jumble:", msg)
	if col != colWarn || !strings.Contains(msg, "motor") {
		t.Fatalf("motors inside the frame should be flagged, got %q", msg)
	}
	// one motor on the ground and one up top: drives, but the top one is waste
	top := StarterDesign()
	top.Set(1, 2, KMotor)
	msg, _ = dropVerdict(top)
	t.Log("top motor:", msg)
	if !strings.Contains(msg, "a motor wheel is off the ground") {
		t.Fatalf("a motor on the roof should be flagged, got %q", msg)
	}
}
