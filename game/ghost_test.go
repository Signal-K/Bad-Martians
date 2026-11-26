package game

import (
	"math"
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

// A recorded drive survives the host's storage and replays where it was.
func TestGhostRoundTrip(t *testing.T) {
	lvl := BuildLevel(LevelDefs[0])
	run := NewRun(lvl, referenceDesigns()["starter"])
	rec := NewGhostRecorder(run)
	for i := 0; i < 600 && run.Outcome == Running; i++ {
		run.Step(engine.FixedDT, 1)
		rec.Observe(run)
	}
	n := len(rec.g.S)
	rec.Observe(run) // no more samples once the run has ended, or while one is not due
	if run.Outcome != Running && len(rec.g.S) != n {
		t.Fatalf("recorder kept sampling after the run ended")
	}
	h := engine.NewMemHost()
	if LoadGhost(h, 1) != nil {
		t.Fatal("ghost before any was saved")
	}
	SaveGhost(h, 1, rec.Ghost(run.Time))
	g := LoadGhost(h, 1)
	if g == nil {
		t.Fatal("saved ghost did not load")
	}
	c, _ := g.at(run.Time)
	if d := c.Sub(run.Rover.Center()).Len(); d > 0.6 {
		t.Errorf("ghost ends %.2f m from the rover", d)
	}
	mid, _ := g.at(run.Time / 2)
	if start, _ := g.at(0); math.Abs(mid.X-start.X) < 0.1 {
		t.Errorf("ghost did not move between start and midpoint")
	}
	if len(h.Store[ghostKey(1)]) > 40000 {
		t.Errorf("ghost is %d bytes", len(h.Store[ghostKey(1)]))
	}
}
