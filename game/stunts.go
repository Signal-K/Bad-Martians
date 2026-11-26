package game

import (
	"fmt"
	"math"

	"github.com/signal-k/haphazard-martians/physics"
)

// The stunt engine (docs/STUNTS.md). It watches a drive and awards points for
// flips, wheelies and airtime. It is headless and deterministic, so tests and
// the playtest bots use the same rules as the drive scene.
//
// Stunt points are shown to the player but are NOT part of Score(): the server
// recomputes every leaderboard score from the level, samples and time
// (CheckRun) and cannot replay stunts. Fold them into the score only once runs
// can be verified (MAR-62 follow-up).

// StuntKind names a trick.
type StuntKind string

const (
	StuntBackflip  StuntKind = "backflip"  // a full turn, nose up and over
	StuntFrontflip StuntKind = "frontflip" // a full turn, nose down
	StuntWheelie   StuntKind = "wheelie"   // front wheels up, back wheels driving
	StuntAirtime   StuntKind = "airtime"   // long enough off the ground
)

// Stunt values. The design sketch says 2 points per flip, 2 per wheelie
// second and 2 bonus points for airtime; one sketch point is stuntPointScale
// score points so a trick reads like the rest of the score (a sample is 150).
const (
	stuntPointScale    = 50
	StuntFlipPoints    = 2 * stuntPointScale
	StuntWheeliePoints = 2 * stuntPointScale // per whole second
	StuntAirtimePoints = 2 * stuntPointScale
	StuntAirtimeMin    = 1.5  // seconds off the ground before airtime pays; the sketch left it open
	flipCredit         = 0.85 // fraction of a full turn that counts as a flip
	wheelieMinPitch    = 0.35 // rad (20°) nose up
	wheelieMaxPitch    = 1.2  // rad: beyond this the rover is tipping over, not popping a wheelie
	wheelieGrace       = 0.25 // seconds a wheelie may dip before it ends
	StuntPopupSeconds  = 1.6
)

// Anti-farming. Leaning in the air pumps spin into every bounce, so a rover can
// hop on the spot and spin forever. Jump stunts therefore pay only on landing,
// and only for a jump that gets somewhere.
const (
	StuntMinJump   = 2.5 // metres from take-off to landing
	StuntNewGround = 6.0 // a paid jump must land this far past the last paid one
	StuntMaxFlips  = 3   // flips paid for one jump
)

// Stunt is one awarded trick.
type Stunt struct {
	Kind   StuntKind
	Points int
	Count  int     // flips so far in this jump, or whole wheelie seconds
	Secs   float64 // airtime, seconds
	At     float64 // run time it was awarded
}

// Stunts tracks tricks over one run.
type Stunts struct {
	Awarded []Stunt
	Points  int

	// totals for the debrief and analytics
	Backflips, Frontflips int
	WheelieSecs           float64 // longest wheelie
	BestAir               float64 // longest airtime that paid

	rear, front int // wheel bodies at the back and front of the rover
	ok          bool
	angle       float64 // unwrapped orientation, rad, positive nose up
	primed      bool

	air      float64 // seconds airborne in this jump
	spin     float64 // rotation accumulated this jump
	takeX    float64 // where this jump left the ground
	paidX    float64 // where the last paid jump landed
	paid     bool    // a jump has been paid
	wasAir   bool
	wheelie  float64 // seconds of the current wheelie
	paidSecs int     // whole wheelie seconds already paid
	lapse    float64 // seconds since the wheelie condition last held
	now      float64
	fresh    []Stunt
}

// NewStunts starts tracking a rover. A rover with fewer than two bodies has no
// orientation, so it earns nothing.
func NewStunts(rv *Rover) *Stunts {
	s := &Stunts{}
	s.rear, s.front = -1, -1
	for i, b := range rv.Bodies {
		if !b.Wheel {
			continue
		}
		if s.rear < 0 || b.Pos.X < rv.Bodies[s.rear].Pos.X {
			s.rear = i
		}
		if s.front < 0 || b.Pos.X > rv.Bodies[s.front].Pos.X {
			s.front = i
		}
	}
	s.ok = s.rear >= 0 && s.front >= 0 && s.rear != s.front
	return s
}

// Take returns the stunts awarded since the last call.
func (s *Stunts) Take() []Stunt {
	out := s.fresh
	s.fresh = nil
	return out
}

// WheelieNow is how long the current wheelie has lasted, 0 when there is none.
func (s *Stunts) WheelieNow() float64 { return s.wheelie }

func (s *Stunts) award(st Stunt) {
	st.At = s.now
	s.Points += st.Points
	s.Awarded = append(s.Awarded, st)
	s.fresh = append(s.fresh, st)
}

// Update advances the tracker by dt seconds of simulated time.
func (s *Stunts) Update(rv *Rover, dt float64) {
	s.now += dt
	if !s.ok || dt <= 0 {
		return
	}
	a, f := rv.Bodies[s.rear], rv.Bodies[s.front]
	ang := math.Atan2(f.Pos.Y-a.Pos.Y, f.Pos.X-a.Pos.X)
	if !s.primed {
		s.angle, s.primed = ang, true
		return
	}
	d := ang - math.Mod(s.angle, 2*math.Pi)
	for d > math.Pi {
		d -= 2 * math.Pi
	}
	for d < -math.Pi {
		d += 2 * math.Pi
	}
	s.angle += d

	grounded := false
	for _, b := range rv.Bodies {
		if b.OnGround {
			grounded = true
			break
		}
	}
	if grounded {
		s.land(rv.Center().X)
		s.updateWheelie(a, f, dt)
		return
	}
	// airborne: wheelies end, rotation and time accumulate; nothing pays until
	// the rover lands
	s.endWheelie()
	if !s.wasAir {
		s.wasAir, s.takeX = true, rv.Center().X
	}
	s.air += dt
	s.spin += d
}

// land closes a jump. Flips and long airtime pay now, if the jump covered real
// ground and landed past the last paid one; either way the jump starts over.
func (s *Stunts) land(x float64) {
	if !s.wasAir {
		return
	}
	spin, air, from := s.spin, s.air, s.takeX
	s.wasAir, s.air, s.spin = false, 0, 0
	if x-from < StuntMinJump || (s.paid && x < s.paidX+StuntNewGround) {
		return
	}
	paid := false
	owed := min(StuntMaxFlips, int(math.Abs(spin)/(2*math.Pi)+(1-flipCredit)))
	for n := 1; n <= owed; n++ {
		kind := StuntBackflip
		if spin < 0 {
			kind = StuntFrontflip
			s.Frontflips++
		} else {
			s.Backflips++
		}
		s.award(Stunt{Kind: kind, Points: StuntFlipPoints, Count: n})
		paid = true
	}
	if air >= StuntAirtimeMin {
		s.award(Stunt{Kind: StuntAirtime, Points: StuntAirtimePoints, Secs: air})
		s.BestAir = math.Max(s.BestAir, air)
		paid = true
	}
	if paid {
		s.paid, s.paidX = true, x
	}
}

func (s *Stunts) endWheelie() {
	s.WheelieSecs = math.Max(s.WheelieSecs, s.wheelie)
	s.wheelie, s.paidSecs, s.lapse = 0, 0, 0
}

// updateWheelie pays 1 wheelie second at a time while the nose is up, the back
// wheel is down and the front wheel is not.
func (s *Stunts) updateWheelie(a, f *physics.Body, dt float64) {
	pitch := s.angle - 2*math.Pi*math.Round(s.angle/(2*math.Pi))
	if a.OnGround && !f.OnGround && pitch >= wheelieMinPitch && pitch <= wheelieMaxPitch {
		s.wheelie += dt + s.lapse
		s.lapse = 0
		s.WheelieSecs = math.Max(s.WheelieSecs, s.wheelie)
		for s.paidSecs < int(s.wheelie) {
			s.paidSecs++
			s.award(Stunt{Kind: StuntWheelie, Points: StuntWheeliePoints, Count: s.paidSecs})
		}
		return
	}
	if s.wheelie > 0 {
		if s.lapse += dt; s.lapse > wheelieGrace {
			s.endWheelie()
		}
	}
}

// stuntPopup is a callout on screen.
type stuntPopup struct {
	text string
	age  float64
}

// Label is the callout the player reads: the trick and what it paid.
func (s Stunt) Label() string {
	switch s.Kind {
	case StuntBackflip:
		return fmt.Sprintf("Backflip! +%d", s.Points)
	case StuntFrontflip:
		return fmt.Sprintf("Frontflip! +%d", s.Points)
	case StuntWheelie:
		return fmt.Sprintf("Wheelie! +%d", s.Points)
	}
	return fmt.Sprintf("Air time %.1fs +%d", s.Secs, s.Points)
}
