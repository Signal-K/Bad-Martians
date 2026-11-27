package game

import (
	"math"
	"testing"

	"github.com/signal-k/haphazard-martians/physics"
)

// stuntRover is a two-wheel rover pose we can pitch by hand.
func stuntRover() (*Rover, *physics.Body, *physics.Body) {
	testX = 0
	a := &physics.Body{Wheel: true, Pos: physics.V(0, 1), InvMass: 1}
	f := &physics.Body{Wheel: true, Pos: physics.V(2, 1), InvMass: 1}
	return &Rover{Bodies: []*physics.Body{a, f}}, a, f
}

// testX is how far along the course the posed rover is; jumps pay only when
// the rover travels, so the helpers advance it.
var testX float64

// pitch puts the rover at an angle (rad, positive nose up) about its middle.
func pitch(a, f *physics.Body, ang float64) {
	c := physics.V(1+testX, 1)
	a.Pos = physics.V(c.X-math.Cos(ang), c.Y-math.Sin(ang))
	f.Pos = physics.V(c.X+math.Cos(ang), c.Y+math.Sin(ang))
}

// spin turns the airborne rover through turns full rotations over secs seconds.
func spinFor(st *Stunts, rv *Rover, a, f *physics.Body, turns, secs float64) {
	const dt = 1.0 / 60
	a.OnGround, f.OnGround = false, false
	n := int(secs / dt)
	for i := 1; i <= n; i++ {
		testX += 6 * dt // a real jump travels
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
		land(st, rv, a, f) // flips pay on landing
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
			testX += 6.0 / 60
			pitch(a, f, 0)
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
	land(st, rv, a, f)
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

// throw lifts the starter rover clear of the ground and spins it about its
// centre at omega rad/s (positive = nose up, a backflip), then lets the real
// physics bring it down. It reports the stunts the run awarded.
func throw(t *testing.T, lift, omega float64) *Run {
	t.Helper()
	r := NewRun(GetLevel(1), StarterDesign())
	r.Speed = DriveSpeed
	c := r.Rover.Center()
	for _, b := range r.Rover.Bodies {
		d := b.Pos.Sub(c)
		b.Pos.Y += lift
		b.Prev = b.Pos.Sub(physics.V(-d.Y*omega+6, d.X*omega).Mul(physH)) // spinning, and travelling at 6 m/s
	}
	airborne := false
	for i := 0; i < 60*12 && r.Outcome == Running; i++ {
		r.Step(1.0/60, 0)
		grounded := false
		for _, b := range r.Rover.Bodies {
			grounded = grounded || b.OnGround
		}
		airborne = airborne || !grounded
		if airborne && grounded {
			break
		}
	}
	if !airborne {
		t.Fatal("the rover never left the ground")
	}
	return r
}

func TestRealPhysicsBackflip(t *testing.T) {
	r := throw(t, 6, 5)
	t.Logf("backflip throw: %+v points=%d", r.Stunts.Awarded, r.Stunts.Points)
	if r.Stunts.Backflips < 1 || r.Stunts.Frontflips != 0 {
		t.Fatalf("a nose-up spin should be a backflip: back %d, front %d", r.Stunts.Backflips, r.Stunts.Frontflips)
	}
	if r.Stunts.Points < StuntFlipPoints {
		t.Fatalf("the flip paid %d, want at least %d", r.Stunts.Points, StuntFlipPoints)
	}
}

func TestRealPhysicsFrontflip(t *testing.T) {
	r := throw(t, 6, -5)
	t.Logf("frontflip throw: %+v points=%d", r.Stunts.Awarded, r.Stunts.Points)
	if r.Stunts.Frontflips < 1 || r.Stunts.Backflips != 0 {
		t.Fatalf("a nose-down spin should be a frontflip: back %d, front %d", r.Stunts.Backflips, r.Stunts.Frontflips)
	}
}

func TestRealPhysicsStraightDropFlipsNothing(t *testing.T) {
	r := throw(t, 6, 0)
	t.Logf("straight drop: %+v points=%d", r.Stunts.Awarded, r.Stunts.Points)
	if r.Stunts.Backflips+r.Stunts.Frontflips != 0 {
		t.Fatalf("a straight drop flipped: %+v", r.Stunts.Awarded)
	}
}

// Stunts are shown but never scored: the verified leaderboard score must be
// identical with or without them.
func TestStuntsDoNotChangeScore(t *testing.T) {
	r := throw(t, 6, 5)
	if r.Stunts.Points == 0 {
		t.Skip("no stunt points to compare")
	}
	if got, want := r.Score(), ScoreFor(r.Level, r.Outcome == Won, r.SamplesGot(), r.Time); got != want {
		t.Fatalf("stunts leaked into the score: %+v vs %+v", got, want)
	}
}

// Spinning in the air without going anywhere is not a stunt: lean pumps spin
// into bounces, so hopping on the spot would otherwise pay forever.
func TestHoppingOnTheSpotPaysNothing(t *testing.T) {
	rv, a, f := stuntRover()
	st := NewStunts(rv)
	land(st, rv, a, f)
	for hop := 0; hop < 5; hop++ {
		a.OnGround, f.OnGround = false, false
		for i := 1; i <= 120; i++ { // two seconds up, a full turn, straight up and down
			pitch(a, f, 2*math.Pi*float64(i)/120)
			st.Update(rv, 1.0/60)
		}
		land(st, rv, a, f)
	}
	if st.Points != 0 {
		t.Fatalf("hopping on the spot paid %d: %+v", st.Points, st.Awarded)
	}
}

// A second jump that lands where the last paid one did is not new ground.
func TestRepeatJumpOnTheSameGroundPaysOnce(t *testing.T) {
	rv, a, f := stuntRover()
	st := NewStunts(rv)
	land(st, rv, a, f)
	spinFor(st, rv, a, f, 1, 0.9)
	land(st, rv, a, f)
	first := st.Points
	if first == 0 {
		t.Fatal("the first jump should pay")
	}
	back := testX
	for i := 0; i < 2; i++ { // jump again, landing short of new ground
		testX = back - 4
		land(st, rv, a, f)
		spinFor(st, rv, a, f, 1, 0.9)
		testX = back // lands level with the paid jump, not past it
		land(st, rv, a, f)
	}
	if st.Points != first {
		t.Fatalf("repeats on paid ground paid %d more", st.Points-first)
	}
}

// A flip the rover does not survive to land never pays.
func TestUnlandedFlipPaysNothing(t *testing.T) {
	rv, a, f := stuntRover()
	st := NewStunts(rv)
	land(st, rv, a, f)
	spinFor(st, rv, a, f, 2, 1.2) // spins twice, then the run ends in the air
	if st.Points != 0 {
		t.Fatalf("an unlanded jump paid %d", st.Points)
	}
}

// Holding lean down for a whole drive used to pump spin into every landing,
// so the rover hopped along flat ground farming flips. It must now behave like
// a rover with a stuck button, not a stunt machine.
func TestHoldingLeanDoesNotFarmStunts(t *testing.T) {
	for _, dir := range []float64{1, -1} {
		m, _ := MissionFor("2026-12-21")
		r := NewRun(BuildLevel(m.Level), StarterDesign())
		r.Speed = DriveSpeed
		for i := 0; i < 60*60 && r.Outcome == Running; i++ {
			r.Lean = dir
			r.Step(1.0/60, 1)
		}
		t.Logf("lean %+.0f held: outcome %v, %d stunt points (%d back, %d front)", dir, r.Outcome, r.Stunts.Points, r.Stunts.Backflips, r.Stunts.Frontflips)
		if r.Stunts.Points > 600 {
			t.Errorf("holding lean %+.0f farmed %d stunt points", dir, r.Stunts.Points)
		}
	}
}
