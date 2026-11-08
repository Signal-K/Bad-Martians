package game

import "testing"

// reference designs span the vocabulary the builder offers today.
func referenceDesigns() map[string]Design {
	return map[string]Design{
		"starter": StarterDesign(),
		"ref":     referenceDesign(),
		"six":     {Cells: []Cell{{0, 0, KMotor}, {1, 0, KMotor}, {2, 0, KMotor}, {3, 0, KMotor}, {0, 1, KFrame}, {1, 1, KFrame}, {2, 1, KMast}, {3, 1, KFrame}}},
		"tri":     {Cells: []Cell{{0, 0, KMotor}, {1, 0, KFrame}, {2, 0, KMotor}, {3, 0, KFrame}, {4, 0, KMotor}, {1, 1, KFrame}, {2, 1, KMast}, {3, 1, KFrame}}},
		"low":     {Cells: []Cell{{0, 0, KMotor}, {1, 0, KFrame}, {2, 0, KMotor}, {1, 1, KMast}}},
		"wide":    {Cells: []Cell{{0, 0, KMotor}, {1, 0, KFrame}, {2, 0, KFrame}, {3, 0, KFrame}, {4, 0, KMotor}, {1, 1, KFrame}, {2, 1, KMast}}},
	}
}

// clears reports whether a design is affordable on the level's base budget and
// wins it.
func clears(def LevelDef, d Design) bool {
	if d.Cost() > def.Credits {
		return false
	}
	return drive(BuildLevel(def), d, 120).Outcome == Won
}

// The design rule from the brief: levels must be built so that one design
// cannot clear them all, and every level must be clearable by something.
func TestNoSingleDesignClearsEveryLevel(t *testing.T) {
	ds := referenceDesigns()
	won := map[string]int{}
	for _, def := range LevelDefs {
		any := false
		for name, d := range ds {
			if clears(def, d) {
				won[name]++
				any = true
			}
		}
		if !any {
			t.Errorf("level %d (%s): no reference design clears it", def.ID, def.Name)
		}
	}
	for name, n := range won {
		t.Logf("%-8s clears %d/%d levels", name, n, len(LevelDefs))
		if n == len(LevelDefs) {
			t.Errorf("design %q clears every level; levels need to demand different rovers", name)
		}
	}
}
