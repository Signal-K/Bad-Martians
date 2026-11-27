package game

import (
	"math"
	"strings"
	"testing"
)

// plains is a flat, firm level to put ramps on.
func plains() LevelDef {
	d := LevelDefs[1]
	d.Hill, d.Length = 0.6, 140
	return d
}

// A ramp changes the ground only between its foot and the end of its landing,
// and the lip stands exactly Rise above the ground that was there.
func TestRampOnlyChangesItsOwnStretch(t *testing.T) {
	d := plains()
	base := BuildLevel(d)
	rp := Ramp{At: 0.4, Rise: 2, Run: 6, Fall: 3}
	d.Ramps = []Ramp{rp}
	got := BuildLevel(d)
	foot, lip, end := rp.Foot(d.Length), rp.Lip(d.Length), rp.End(d.Length)
	if len(got.Terrain.H) != len(base.Terrain.H) {
		t.Fatal("a ramp changed the terrain length")
	}
	for i := range got.Terrain.H {
		x := got.Terrain.X0 + float64(i)*got.Terrain.DX
		delta := got.Terrain.H[i] - base.Terrain.H[i]
		switch {
		case x <= foot || x >= end:
			if delta != 0 {
				t.Fatalf("x=%.1f is outside the ramp but moved by %.4f", x, delta)
			}
		case x <= lip && delta < -1e-9:
			t.Fatalf("x=%.1f on the way up dipped by %.4f", x, delta)
		}
		if math.Abs(x-lip) < 1e-9 && math.Abs(delta-rp.Rise) > 1e-9 {
			t.Fatalf("lip rises %.4f, want %.4f", delta, rp.Rise)
		}
	}
	// the way up never goes down
	prev := 0.0
	for i := range got.Terrain.H {
		x := got.Terrain.X0 + float64(i)*got.Terrain.DX
		if x > foot && x <= lip {
			delta := got.Terrain.H[i] - base.Terrain.H[i]
			if delta < prev-1e-9 {
				t.Fatalf("ramp dips on the way up at x=%.1f", x)
			}
			prev = delta
		}
	}
}

// A level with no ramps is exactly the level it always was (the golden hashes
// check the campaign; this checks the mechanism on a fresh definition).
func TestNoRampsChangesNothing(t *testing.T) {
	d := plains()
	a := BuildLevel(d)
	d.Ramps = []Ramp{}
	b := BuildLevel(d)
	for i := range a.Terrain.H {
		if a.Terrain.H[i] != b.Terrain.H[i] {
			t.Fatalf("sample %d differs", i)
		}
	}
}

func TestRampFileValidation(t *testing.T) {
	good := `{"id":1,"name":"x","blurb":"","biome":"plains","length":140,"seed":1,"credits":30,"par":30,"hill":1,"traction":1,"mix":{"soil":1},"ramps":[RAMPS]}`
	cases := map[string]struct{ ramps, want string }{
		"fine":            {`{"at":0.4,"rise":2,"run":6}`, ""},
		"two apart":       {`{"at":0.2,"rise":1.5,"run":5},{"at":0.6,"rise":2,"run":6}`, ""},
		"too early":       {`{"at":0.02,"rise":2,"run":6}`, "between 0.1 and 0.8"},
		"too late":        {`{"at":0.95,"rise":2,"run":6}`, "between 0.1 and 0.8"},
		"too tall":        {`{"at":0.4,"rise":9,"run":24}`, "between 0.3 and 6"},
		"too short":       {`{"at":0.4,"rise":2,"run":1}`, "between 2 and 24"},
		"too steep":       {`{"at":0.4,"rise":5,"run":3}`, "steeper than 45"},
		"hits the finish": {`{"at":0.8,"rise":2,"run":24,"fall":10}`, "too close to the finish"},
		"overlap":         {`{"at":0.4,"rise":2,"run":6},{"at":0.42,"rise":2,"run":6}`, "overlaps"},
		"unknown field":   {`{"at":0.4,"rise":2,"run":6,"jump":1}`, "unknown field"},
		"too many":        {strings.Repeat(`{"at":0.2,"rise":2,"run":2},`, 5) + `{"at":0.3,"rise":2,"run":2}`, "at most 4"},
	}
	for name, c := range cases {
		_, err := ParseLevel([]byte(strings.Replace(good, "RAMPS", c.ramps, 1)))
		switch {
		case c.want == "" && err != nil:
			t.Errorf("%s: unexpected error %v", name, err)
		case c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)):
			t.Errorf("%s: error %v, want it to mention %q", name, err, c.want)
		}
	}
}

// Every authored ramp stays clear of crevasses and the start and finish, so a
// ramp never lands a rover in a gap.
func TestAuthoredRampsAvoidCrevasses(t *testing.T) {
	check := func(d LevelDef) {
		l := BuildLevel(d)
		for _, rp := range d.Ramps {
			for _, g := range l.Gaps {
				if rp.Foot(d.Length) < g[1]+4 && g[0]-4 < rp.End(d.Length) {
					t.Errorf("level %d: ramp %.0f to %.0f m is within 4 m of the crevasse at %.0f", d.ID, rp.Foot(d.Length), rp.End(d.Length), g[0])
				}
			}
		}
	}
	for _, d := range LevelDefs {
		check(d)
	}
	for _, m := range Missions {
		check(m.Level)
	}
}

// Reachable, but not free: the starter rover at full throttle gets enough air
// off the ramps to earn airtime and still finishes, but does not flip unless
// the player leans. Leaning once for a turn earns a backflip.
func TestRampsMakeStuntsReachableButNotFree(t *testing.T) {
	m, ok := MissionFor("2026-12-21")
	if !ok || len(m.Level.Ramps) == 0 {
		t.Fatal("the stunt mission is missing")
	}
	drive := func(lean bool) *Run {
		r := NewRun(BuildLevel(m.Level), StarterDesign())
		r.Speed = DriveSpeed
		for i := 0; i < 60*90 && r.Outcome == Running; i++ {
			air := true
			for _, b := range r.Rover.Bodies {
				air = air && !b.OnGround
			}
			r.Lean = 0
			if lean && air && math.Abs(r.Stunts.spin) < 2*math.Pi*0.97 {
				r.Lean = 1
			}
			r.Step(1.0/60, 1)
		}
		return r
	}
	free := drive(false)
	if free.Outcome != Won {
		t.Fatalf("the starter rover should finish the stunt mission, got %v", free.Outcome)
	}
	if free.Stunts.Backflips+free.Stunts.Frontflips != 0 {
		t.Fatalf("flat-out driving flipped without leaning: %+v", free.Stunts.Awarded)
	}
	if free.Stunts.BestAir < StuntAirtimeMin {
		t.Fatalf("the ramps gave only %.2fs of paid air, want at least %.1fs", free.Stunts.BestAir, StuntAirtimeMin)
	}
	flip := drive(true)
	if flip.Stunts.Backflips < 1 {
		t.Fatalf("leaning in the air should earn a backflip: %+v", flip.Stunts.Awarded)
	}
	t.Logf("no lean: %d points, best air %.2fs; one-flip lean: %d points, %d backflips",
		free.Stunts.Points, free.Stunts.BestAir, flip.Stunts.Points, flip.Stunts.Backflips)
}
