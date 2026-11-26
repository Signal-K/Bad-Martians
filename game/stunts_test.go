package game

import (
	"math"
	"testing"

	"github.com/signal-k/haphazard-martians/physics"
)

// stuntRover is a two-wheel rover pose we can pitch by hand.
func stuntRover() (*Rover, *physics.Body, *physics.Body) {
	a := &physics.Body{Wheel: true, Pos: physics.V(0, 1)}
	f := &physics.Body{Wheel: true, Pos: physics.V(2, 1)}
	return &Rover{Bodies: []*physics.Body{a, f}}, a, f
}

// pitch puts the rover at an angle (rad, positive nose up) about its middle.
func pitch(a, f *physics.Body, ang float64) {
	c := physics.V(1, 1)
	a.Pos = physics.V(c.X-math.Cos(ang), c.Y-math.Sin(ang))
	f.Pos = physics.V(c.X+math.Cos(ang), c.Y+math.Sin(ang))
}

// spin turns the airborne rover through turns full rotations over secs seconds.
func spinFor(st *Stunts, rv *Rover, a, f *physics.Body, turns, secs float64) {
	const dt = 1.0 / 60
	a.OnGround, f.OnGround = false, false
	n := int(secs / dt)
	for i := 1; i <= n; i++ {
		pitch(a, f, 2*math.Pi*turns*float64(i)/float64(n))
		st.Update(rv, dt)
	}
}

func land(st *Stunts, rv *Rover, a, f *physics.Body) {
	a.OnGround, f.OnGround = true, true
	pitch(a, f, 0)
	st.Update(rv, 1.0/60)
}

func count(st *Stunts, k StuntKind) int {
	n := 0
	for _, s := range st.Awarded {
		if s.Kind == k {
			n++
		}
	}
	return n
}

func TestBackflipAndFrontflip(t *testing.T) {
	for _, tc := range []struct {
		turns float64
		kind  StuntKind
		n     int
	}{{1, StuntBackflip, 1}, {-1, StuntFrontflip, 1}, {2, StuntBackflip, 2}, {0.5, StuntBackflip, 0}} {
		rv, a, f := stuntRover()
		st := NewStunts(rv)
		land(st, rv, a, f)
		spinFor(st, rv, a, f, tc.turns, 0.9)
		if got := count(st, tc.kind); got != tc.n {
			t.Errorf("%.1f turns: %d %s, want %d", tc.turns, got, tc.kind, tc.n)
		}
		if st.Points != tc.n*StuntFlipPoints {
			t.Errorf("%.1f turns: %d points, want %d", tc.turns, st.Points, tc.n*StuntFlipPoints)
		}
	}
}

func TestFlipOnTheGroundIsNotAStunt(t *testing.T) {
	rv, a, f := stuntRover()
	st := NewStunts(rv)
	a.OnGround, f.OnGround = true, true
	for i := 1; i <= 120; i++ {
		pitch(a, f, 2*math.Pi*float64(i)/120)
		st.Update(rv, 1.0/60)
	}
	if st.Points != 0 {
		t.Fatalf("rolling over on the ground paid %d", st.Points)
	}
}

func TestAirtimePaysOnlyWhenLongEnough(t *testing.T) {
	for _, tc := range []struct {
		secs float64
		paid bool
	}{{0.5, false}, {StuntAirtimeMin + 0.2, true}} {
		rv, a, f := stuntRover()
		st := NewStunts(rv)
		land(st, rv, a, f)
		a.OnGround, f.OnGround = false, false
		for i := 0; i < int(tc.secs*60); i++ {
			st.Update(rv, 1.0/60)
		}
		land(st, rv, a, f)
		if got := count(st, StuntAirtime) == 1; got != tc.paid {
			t.Errorf("%.1fs in the air: paid=%v, want %v", tc.secs, got, tc.paid)
		}
	}
}

func TestWheeliePaysPerSecond(t *testing.T) {
	rv, a, f := stuntRover()
	st := NewStunts(rv)
	land(st, rv, a, f)
	pitch(a, f, 0.6)
	a.OnGround, f.OnGround = true, false
	for i := 0; i < int(2.5*60); i++ {
		st.Update(rv, 1.0/60)
	}
	if got := count(st, StuntWheelie); got != 2 {
		t.Fatalf("2.5s wheelie paid %d seconds, want 2", got)
	}
	// front wheel down again: it stops, and no further seconds are paid
	f.OnGround = true
	pitch(a, f, 0)
	for i := 0; i < 120; i++ {
		st.Update(rv, 1.0/60)
	}
	if got := count(st, StuntWheelie); got != 2 {
		t.Fatalf("a finished wheelie kept paying: %d", got)
	}
}

func TestNoFlatWheelieOrTipover(t *testing.T) {
	for _, ang := range []float64{0.1, 1.5} { // level, and tipped over
		rv, a, f := stuntRover()
		st := NewStunts(rv)
		land(st, rv, a, f)
		pitch(a, f, ang)
		a.OnGround, f.OnGround = true, false
		for i := 0; i < 240; i++ {
			st.Update(rv, 1.0/60)
		}
		if st.Points != 0 {
			t.Errorf("pitch %.1f paid %d", ang, st.Points)
		}
	}
}

func TestTakeReturnsEachStuntOnce(t *testing.T) {
	rv, a, f := stuntRover()
	st := NewStunts(rv)
	land(st, rv, a, f)
	spinFor(st, rv, a, f, 1, 0.9)
	if n := len(st.Take()); n != 1 {
		t.Fatalf("first take: %d, want 1", n)
	}
	if n := len(st.Take()); n != 0 {
		t.Fatalf("second take: %d, want 0", n)
	}
}

// The real physics must not hand out stunts for ordinary driving: the
// starter rover on mission 1 only drives.
func TestStarterDriveOnMissionOneIsNotAStuntShow(t *testing.T) {
	r := NewRun(GetLevel(1), StarterDesign())
	for i := 0; i < 60*20 && r.Outcome == Running; i++ {
		r.Step(1.0/60, 1)
	}
	t.Logf("outcome %v, stunts %d points, awarded %v", r.Outcome, r.Stunts.Points, r.Stunts.Awarded)
	if r.Stunts.Backflips+r.Stunts.Frontflips > 0 {
		t.Fatalf("flat-out drive flipped the rover: %+v", r.Stunts.Awarded)
	}
}
