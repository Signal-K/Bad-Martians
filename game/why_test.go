package game

import (
	"strings"
	"testing"
)

// driveUntilStopped drives at full throttle until the run ends or the Restart
// offer appears.
func driveUntilStopped(l *Level, d Design) *Run {
	r := NewRun(l, d)
	for r.Time < 120 && r.Outcome == Running && !r.IsStuck() {
		r.Step(1.0/60, 1)
	}
	return r
}

// Every way a rover stops is named, with the reason and what to change, so a
// failure teaches the player something about the parts.
func TestFailuresExplainWhy(t *testing.T) {
	for _, c := range []struct {
		name  string
		level int
		d     Design
		want  string // in Cause.What
		blame Kind   // a culprit is this kind ("" for none required)
	}{
		{"motor above the frame", 1, Design{Cells: []Cell{{0, 0, KFrame}, {1, 0, KFrame}, {2, 0, KFrame}, {1, 1, KMotor}, {2, 1, KMast}}}, "Beached", KMotor},
		{"motor above free wheels", 1, Design{Cells: []Cell{{0, 0, KWheel}, {1, 0, KFrame}, {2, 0, KWheel}, {1, 1, KMotor}, {2, 1, KMast}}}, "Only free wheels", KMotor},
		{"frame on the ground", 1, Design{Cells: []Cell{{0, 0, KFrame}, {1, 0, KFrame}, {2, 0, KMotor}, {1, 1, KMast}}}, "frame is dragging", KFrame},
		{"tower", 1, Design{Cells: []Cell{{0, 0, KMotor}, {0, 1, KFrame}, {0, 2, KFrame}, {0, 3, KMast}}}, "tipped", ""},
		{"short over a crevasse", 5, StarterDesign(), "crevasse", ""},
		{"one motor in sand", 8, StarterDesign(), "sand", ""},
		{"one motor up a sandy hill", 16, StarterDesign(), "no grip", KMotor},
	} {
		r := driveUntilStopped(GetLevel(c.level), c.d)
		if r.Outcome == Won {
			t.Fatalf("%s: expected a failure on mission %d", c.name, c.level)
		}
		why := r.Diagnose()
		if !strings.Contains(why.What, c.want) || why.Why == "" || why.Fix == "" {
			t.Errorf("%s: got %+v, want What containing %q and a reason and a fix", c.name, why, c.want)
		}
		if c.blame != "" {
			found := false
			for _, i := range why.Culprits {
				found = found || r.Rover.Cells[i].K == c.blame
			}
			if !found {
				t.Errorf("%s: the %s should be marked to blame, culprits %v", c.name, c.blame, why.Culprits)
			}
		}
		// the bay keeps the cause, with the cells to mark, until the rover changes
		ses := NewSession(GetLevel(c.level))
		ses.Design = c.d
		ses.NoteFailure(r)
		if got, cells := ses.LastFailure(); got.What != why.What || len(cells) != len(why.Culprits) {
			t.Errorf("%s: bay lost the cause: %+v %v", c.name, got, cells)
		}
		ses.Design.Set(5, 4, KFrame)
		if got, _ := ses.LastFailure(); !got.Empty() {
			t.Errorf("%s: cause should clear once the rover is changed", c.name)
		}
	}
}

// Scrapping freezes the cause from before the blast, not the flying debris.
func TestScrapKeepsTheCause(t *testing.T) {
	r := driveUntilStopped(GetLevel(1), Design{Cells: []Cell{{0, 0, KFrame}, {1, 0, KFrame}, {2, 0, KFrame}, {1, 1, KMotor}, {2, 1, KMast}}})
	before := r.Diagnose().What
	r.Scrap()
	for i := 0; i < 60; i++ {
		r.Debris(1.0 / 60)
	}
	if got := r.Diagnose().What; got != before {
		t.Fatalf("cause changed after the scrap: %q -> %q", before, got)
	}
}
