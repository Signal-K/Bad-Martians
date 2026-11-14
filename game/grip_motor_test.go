package game

import (
	"testing"

	"github.com/signal-k/haphazard-martians/physics"
)

func TestGripMotorBuildAndLimits(t *testing.T) {
	d := Design{Cells: []Cell{{0, 0, KGripMotor}, {0, 1, KMast}}}
	if errs := d.Validate(8); len(errs) != 0 {
		t.Fatalf("grip motor should power a valid rover: %v", errs)
	}
	if errs := RuleProblems(LevelDef{MaxMotor: 1}, Design{Cells: append(d.Cells, Cell{1, 0, KMotor})}); len(errs) == 0 {
		t.Fatal("motor limit must count both motor types")
	}
	r := SpawnRover(&physics.World{}, d, 0, 0)
	if !r.Bodies[0].Motor || r.Bodies[0].Friction <= 1 {
		t.Fatal("grip motor must be powered and have higher grip")
	}
}
