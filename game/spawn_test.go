package game

import (
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

// A rover must start resting on the ground, not inside it. A wide rover on a
// sloped start used to spawn with its downhill wheels buried, and the
// collision solver launched it at up to 39 m/s on the first step.
func TestRoversAreNotFlungAtTheStart(t *testing.T) {
	for _, def := range LevelDefs {
		l := BuildLevel(def)
		for name, d := range referenceDesigns() {
			r := NewRun(l, d)
			r.Speed = DriveSpeed
			for r.Time < 1.5 && r.Outcome == Running {
				c := r.Rover.Center()
				r.Step(engine.FixedDT, 1)
				if vy := (r.Rover.Center().Y - c.Y) / (engine.FixedDT * DriveSpeed); vy > 4 {
					t.Errorf("level %d, %s: rover leaves the start at %.1f m/s upwards", def.ID, name, vy)
					break
				}
			}
		}
	}
}
