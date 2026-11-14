package game

import (
	"fmt"
	"math"

	"github.com/signal-k/haphazard-martians/engine"
)

// Cause explains, in the player's terms, why a rover stopped or failed: what
// physically happened, why the parts behaved that way, and what to change.
// Culprits are the body indices to mark in the world.
type Cause struct {
	What     string
	Why      string
	Fix      string
	Culprits []int
}

func (c Cause) Empty() bool { return c.What == "" }

// pose is where the rover's parts touch, read off the current physics state.
type pose struct {
	motors, motorsDown int   // motor wheels, and those touching the ground
	freeDown           int   // free wheels touching the ground
	dragging           []int // frames or the mast resting on the ground
	airMotors          []int // motor wheels in the air
	tilt               float64
	slope              float64 // ground slope under the rover, degrees, + uphill
	sandMotors         int     // motor wheels touching sand
	lowAir             bool    // an airborne motor wheel is on the bottom row
	sandParts          int     // other parts touching sand
	slipping           bool    // a motor wheel on the ground spins far faster than it moves
	downMotors         []int   // the slipping motor wheels
}

func joinAnd(s []string) string {
	switch len(s) {
	case 0:
		return ""
	case 1:
		return s[0]
	}
	out := s[0]
	for _, x := range s[1 : len(s)-1] {
		out += ", " + x
	}
	return out + " and " + s[len(s)-1]
}

func (r *Run) pose() pose {
	var p pose
	for i, b := range r.Rover.Bodies {
		switch {
		case b.Motor:
			p.motors++
			if b.OnGround {
				p.motorsDown++
				if r.Level.Segments[r.Level.SegmentAt(b.Pos.X)].Class == Sand {
					p.sandMotors++
				}
			} else {
				p.airMotors = append(p.airMotors, i)
				if r.Rover.Cells[i].Y == r.bottom {
					p.lowAir = true
				}
			}
		case b.Wheel:
			if b.OnGround {
				p.freeDown++
			}
		case b.OnGround:
			p.dragging = append(p.dragging, i)
		}
	}
	p.tilt = r.tilt()
	c := r.Rover.Center()
	t := r.Level.Terrain
	p.slope = math.Atan2(t.HeightAt(c.X+1)-t.HeightAt(c.X-1), 2) * 180 / math.Pi
	h := r.World.SubstepDT(engine.FixedDT * max(1, r.Speed))
	for i, b := range r.Rover.Bodies {
		if !b.OnGround {
			continue
		}
		x := b.Pos.X
		if b.Wheel {
			p.slope = math.Max(p.slope, math.Atan2(t.HeightAt(x+0.3)-t.HeightAt(x-0.3), 0.6)*180/math.Pi)
		}
		if !b.Motor && r.Level.Segments[r.Level.SegmentAt(x)].Class == Sand {
			p.sandParts++
		}
		// a driven wheel whose rim moves much faster than the rover is slipping
		if b.Motor && math.Abs(b.Omega*b.Radius)-math.Abs(b.Vel(h).X) > 2 {
			p.slipping = true
			p.downMotors = append(p.downMotors, i)
		}
	}
	return p
}

// tilt is how far the rover has turned from how it was built, in degrees,
// positive nose up: the best-fit rotation of every part about the centre.
func (r *Run) tilt() float64 {
	c := r.Rover.Center()
	var dot, cross float64
	for i, b := range r.Rover.Bodies {
		o := r.built[i]
		d := b.Pos.Sub(c)
		dot += o.X*d.X + o.Y*d.Y
		cross += o.X*d.Y - o.Y*d.X
	}
	return math.Atan2(cross, dot) * 180 / math.Pi
}

// Diagnose says why the rover is not getting anywhere, or why the run failed.
func (r *Run) Diagnose() Cause {
	if r.cause.What != "" {
		return r.cause
	}
	return r.diagnose()
}

func (r *Run) diagnose() Cause {
	if r.Outcome == Fell {
		return r.fellCause()
	}
	p := r.pose()
	motorWord := partNoun(KMotor, p.motors)
	switch {
	case math.Abs(p.tilt) > 55:
		what := "The rover tipped onto its side."
		if math.Abs(p.tilt) > 120 {
			what = "The rover flipped onto its back."
		}
		return Cause{
			What:     what,
			Why:      fmt.Sprintf("It is %.0f m tall on a base %.0f m long, so a bump or a hill tips it. On its side the motor wheels spin in the air.", r.height, r.length),
			Fix:      "Build it lower and longer, with wheels at both ends of the bottom row. Hold the stick against a tip.",
			Culprits: p.dragging,
		}
	case p.motorsDown == 0 && p.lowAir:
		return Cause{
			What:     "The ground dropped away under the motor wheel.",
			Why:      fmt.Sprintf("With %d %s, one bump lifts it off and nothing pushes. Free wheels and frames cannot drive.", p.motors, partNoun(KMotor, p.motors)),
			Fix:      "Add a motor wheel at the other end of the bottom row, so one always touches.",
			Culprits: p.airMotors,
		}
	case p.motorsDown == 0 && len(p.dragging) > 0:
		return Cause{
			What:     "Beached: the frame rests on the ground and the motor wheels hang in the air.",
			Why:      "Motor wheels only push while they touch the ground. A frame slides, it cannot roll.",
			Fix:      "Put the motor wheels on the bottom row, lower than any frame, so they touch first.",
			Culprits: append(append([]int(nil), p.airMotors...), p.dragging...),
		}
	case p.motorsDown == 0:
		return Cause{
			What:     "Only free wheels touch the ground.",
			Why:      "Free wheels roll but have no power. The motor wheel is up in the air, so nothing pushes.",
			Fix:      "Swap a bottom free wheel for a motor wheel, or move the motor wheel down to the bottom row.",
			Culprits: p.airMotors,
		}
	case len(p.dragging) > 0:
		what := "A frame is dragging on the ground."
		for _, i := range p.dragging {
			if r.Rover.Cells[i].K == KMast {
				what = "The Navcam mast is dragging on the ground."
			}
		}
		return Cause{
			What:     what,
			Why:      fmt.Sprintf("Frames and the mast slide instead of rolling. The scrape holds the rover back harder than %d %s can push.", p.motors, motorWord),
			Fix:      "Put a wheel under the part that drags, or move that part up a row.",
			Culprits: p.dragging,
		}
	case p.sandMotors > 0:
		return Cause{
			What: "Bogged down in sand.",
			Why:  fmt.Sprintf("Sand gives wheels half the grip of soil and drags on everything touching it. %d %s cannot push this rover through.", p.motorsDown, partNoun(KMotor, p.motorsDown)),
			Fix:  "Add motor wheels so more of them push, and leave off parts you do not need: lighter rovers sink less.",
		}
	case p.slipping:
		var why []string
		if share := p.motorsDown + p.freeDown; p.freeDown > 0 {
			why = append(why, fmt.Sprintf("only %d of the %d wheels on the ground %s a motor", p.motorsDown, share, map[bool]string{true: "has", false: "have"}[p.motorsDown == 1]))
		}
		if p.sandParts > 0 {
			why = append(why, "parts sunk in sand drag behind it")
		}
		if p.slope > 4 {
			why = append(why, fmt.Sprintf("the ground climbs %.0f°", p.slope))
		}
		reason := "A motor wheel grips with the weight resting on it."
		if len(why) > 0 {
			reason += " Here " + joinAnd(why) + ", so it cannot grip hard enough."
		}
		return Cause{
			What:     fmt.Sprintf("The %s %s in place: no grip.", partNoun(KMotor, p.motorsDown), map[bool]string{true: "spins", false: "spin"}[p.motorsDown == 1]),
			Why:      reason,
			Fix:      "Swap free wheels for motor wheels, so every wheel on the ground pushes. Leave off parts you do not need.",
			Culprits: p.downMotors,
		}
	case r.Outcome == TimedOut:
		dist := r.Progress() * (r.Level.GoalX - r.Level.StartX)
		return Cause{
			What: fmt.Sprintf("Out of time after %.0f m.", dist),
			Why:  fmt.Sprintf("The rover averaged %.1f m/s. Free wheels and frames are weight with no push.", dist/math.Max(1, r.Time)),
			Fix:  "Add a motor wheel or leave off parts you do not need.",
		}
	default:
		return Cause{
			What: "Something in front of the wheels is too tall to roll over.",
			Why:  "A wheel climbs a step lower than its middle. Higher than that, it just pushes against it.",
			Fix:  "Lean back (stick up) to lift the nose over, or put a wheel higher up at the front to meet the step first.",
		}
	}
}

// fellCause explains a drop into a crevasse with the gap and the rover sizes.
func (r *Run) fellCause() Cause {
	c := r.Rover.Center()
	gi := -1
	for i, g := range r.Level.Gaps {
		if c.X > g[0]-2 && c.X < g[1]+2 {
			gi = i
		}
	}
	if gi < 0 {
		return Cause{
			What: "The rover fell off the course.",
			Why:  "It left the ground too fast and came down past the edge.",
			Fix:  "Ease off the stick before big bumps.",
		}
	}
	g := r.Level.Gaps[gi]
	gap := g[1] - g[0]
	v := r.gapSpeed[gi]
	if r.wheelbase < 0.5 {
		return Cause{
			What: fmt.Sprintf("Dropped into a %.1f m crevasse.", gap),
			Why:  fmt.Sprintf("It has one wheel on the ground, so nothing held it up over the gap, and %.1f m/s at the edge was not enough to fly over.", v),
			Fix:  fmt.Sprintf("Put wheels at both ends of the bottom row, at least %.0f cells apart, so it bridges the gap.", math.Ceil(gap+0.5)),
		}
	}
	if r.wheelbase < gap+0.5 {
		return Cause{
			What: fmt.Sprintf("Dropped into a %.1f m crevasse.", gap),
			Why:  fmt.Sprintf("Its outer wheels are %.1f m apart, too close to reach both sides, and %.1f m/s at the edge was not enough to fly over.", r.wheelbase, v),
			Fix:  fmt.Sprintf("Put wheels at least %.0f cells apart on the bottom row so it bridges the gap.", math.Ceil(gap+0.5)),
		}
	}
	return Cause{
		What: fmt.Sprintf("Tipped into a %.1f m crevasse.", gap),
		Why:  fmt.Sprintf("Its wheels are %.1f m apart, wide enough, but its weight hung over the gap and pulled it in.", r.wheelbase),
		Fix:  "Keep heavy parts (motors, the mast) between the wheels, and lean back (stick up) at the edge.",
	}
}

// measure records the rover's built shape and size when the run starts.
func (r *Run) measure() {
	c := r.Rover.Center()
	lo, hi := math.Inf(1), math.Inf(-1)
	r.built = make([]engine.Vec2, len(r.Rover.Bodies))
	for i, b := range r.Rover.Bodies {
		r.built[i] = b.Pos.Sub(c)
		lo, hi = math.Min(lo, b.Pos.X-b.Radius), math.Max(hi, b.Pos.X+b.Radius)
	}
	r.length = hi - lo
	wlo, whi := math.Inf(1), math.Inf(-1)
	ylo, yhi := math.Inf(1), math.Inf(-1)
	r.bottom = GridH
	for i, b := range r.Rover.Bodies {
		if b.Wheel {
			wlo, whi = math.Min(wlo, b.Pos.X), math.Max(whi, b.Pos.X)
		}
		ylo, yhi = math.Min(ylo, b.Pos.Y-b.Radius), math.Max(yhi, b.Pos.Y+b.Radius)
		r.bottom = min(r.bottom, r.Rover.Cells[i].Y)
	}
	r.wheelbase = math.Max(0, whi-wlo)
	r.height = yhi - ylo
	r.gapSpeed = make([]float64, len(r.Level.Gaps))
	for i := range r.gapSpeed {
		r.gapSpeed[i] = -1
	}
}

// watchGaps notes the rover's speed as its front reaches each crevasse.
func (r *Run) watchGaps(speed float64) {
	front := math.Inf(-1)
	for _, b := range r.Rover.Bodies {
		front = math.Max(front, b.Pos.X)
	}
	for i, g := range r.Level.Gaps {
		if r.gapSpeed[i] < 0 && front >= g[0] {
			r.gapSpeed[i] = speed
		}
	}
}

// drawCause is the "why" panel: what happened, the reason, and what to try.
// It returns the panel's height.
func drawCause(r engine.Renderer, c Cause, x, y, w float64) float64 {
	what := wrapText(c.What, sizeBody, w-36)
	why := wrapText(c.Why, sizeSmall, w-36)
	fix := wrapText("Try: "+c.Fix, sizeSmall, w-36)
	h := 26 + 24*float64(len(what)) + 8 + 20*float64(len(why)+len(fix)) + 8
	drawPanel(r, x, y, w, h, "Why it stopped")
	ty := y + 18
	for _, ln := range what {
		r.Text(ln, x+18, ty, sizeBody, colWarn, engine.AlignLeft)
		ty += 24
	}
	ty += 6
	for _, ln := range why {
		r.Text(ln, x+18, ty, sizeSmall, colInk, engine.AlignLeft)
		ty += 20
	}
	ty += 2
	for _, ln := range fix {
		r.Text(ln, x+18, ty, sizeSmall, colNeon, engine.AlignLeft)
		ty += 20
	}
	return h
}

// drawFailure fills the debrief panel of a failed mission with the cause, the
// reason and what to change in the bay.
func drawFailure(r engine.Renderer, c Cause, x, y, w, h float64) {
	drawPanel(r, x, y, w, h, "What went wrong")
	ty := y + 24
	for _, ln := range wrapText(c.What, sizeBody+1, w-48) {
		r.Text(ln, x+24, ty, sizeBody+1, colWarn, engine.AlignLeft)
		ty += 26
	}
	ty += 8
	for _, ln := range wrapText(c.Why, sizeSmall+1, w-48) {
		r.Text(ln, x+24, ty, sizeSmall+1, colInk, engine.AlignLeft)
		ty += 21
	}
	ty += 12
	r.Text("Change in the bay", x+24, ty, sizeSmall, colMuted, engine.AlignLeft)
	ty += 22
	for _, ln := range wrapText(c.Fix, sizeSmall+1, w-48) {
		r.Text(ln, x+24, ty, sizeSmall+1, colNeon, engine.AlignLeft)
		ty += 21
	}
}
