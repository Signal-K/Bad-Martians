package game

import "testing"

// referenceDesigns is a library of rovers a player might build: tiny and
// cheap, compact and motor-heavy, tall, long and braced, wheel-and-motor mixes.
func referenceDesigns() map[string]Design {
	M, F, W, A := KMotor, KFrame, KWheel, KMast
	row := func(ks ...Kind) []Cell {
		var c []Cell
		for i, k := range ks {
			c = append(c, Cell{i, 0, k})
		}
		return c
	}
	with := func(c []Cell, extra ...Cell) Design { return Design{Cells: append(c, extra...)} }
	return map[string]Design{
		"mini":    with(row(M, M), Cell{0, 1, A}),
		"low":     with(row(M, F, M), Cell{1, 1, A}),
		"starter": StarterDesign(),
		"quad":    with(row(M, M, M), Cell{1, 1, A}),
		"tall":    with(row(M, F, M), Cell{1, 1, F}, Cell{1, 2, F}, Cell{1, 3, A}),
		"ref":     referenceDesign(),
		"wide":    with(row(M, F, F, F, M), Cell{1, 1, F}, Cell{2, 1, A}),
		"tri":     with(row(M, F, M, F, M), Cell{1, 1, F}, Cell{2, 1, A}, Cell{3, 1, F}),
		"six":     with(row(M, M, M, M), Cell{0, 1, F}, Cell{1, 1, F}, Cell{2, 1, A}, Cell{3, 1, F}),
		"m4":      with(row(M, M, M, M), Cell{1, 1, A}),
		"long":    with(row(M, F, F, F, F, F, M), Cell{3, 1, A}),
		"longw":   with(row(W, F, F, F, F, F, M), Cell{3, 1, A}),
		"long3":   with(row(M, F, F, M, F, F, M), Cell{3, 1, A}),
		"long4":   with(row(M, M, F, F, F, M, M), Cell{3, 1, A}),
		"truss6":  with(row(M, F, F, F, F, M), Cell{1, 1, F}, Cell{2, 1, F}, Cell{3, 1, A}, Cell{4, 1, F}),
		"truss7":  with(row(M, F, F, F, F, F, M), Cell{1, 1, F}, Cell{2, 1, F}, Cell{3, 1, A}, Cell{4, 1, F}, Cell{5, 1, F}),
		"truss7w": with(row(W, F, F, F, F, F, M), Cell{1, 1, F}, Cell{2, 1, F}, Cell{3, 1, A}, Cell{4, 1, F}, Cell{5, 1, F}),
		"truss73": with(row(M, F, F, M, F, F, M), Cell{1, 1, F}, Cell{2, 1, F}, Cell{3, 1, A}, Cell{4, 1, F}, Cell{5, 1, F}),
		"truss8":  with(row(M, M, F, F, F, F, M, M), Cell{1, 1, F}, Cell{2, 1, F}, Cell{3, 1, F}, Cell{4, 1, A}, Cell{5, 1, F}, Cell{6, 1, F}),
		"w2m":     with(row(W, M, M, W), Cell{1, 1, F}, Cell{2, 1, A}),
		"mw":      with(row(M, F, W), Cell{1, 1, A}),
		"mfw":     with(row(M, F, F, W), Cell{1, 1, F}, Cell{2, 1, A}),
		"wmw":     with(row(W, M, W), Cell{1, 1, A}),
		"wm":      with(row(W, M), Cell{1, 1, A}),
		"mm":      with(row(M, F, F, M), Cell{1, 1, A}),
		"long9":   with(row(M, F, F, F, M, F, F, F, M), Cell{4, 1, A}),
	}
}

// clears reports whether a design is affordable on the level's base budget and
// wins it.
func clears(def LevelDef, d Design) bool {
	if d.Cost() > def.Credits || len(RuleProblems(def, d)) > 0 {
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
		if 2*n > len(LevelDefs) {
			t.Errorf("design %q clears %d of %d levels; no rover may win more than half", name, n, len(LevelDefs))
		}
	}
}

// Lander rules: a rover that breaks one cannot launch, and the tray stops
// offering parts at the cap.
func TestLanderRules(t *testing.T) {
	def := LevelDef{MaxParts: 4, MaxMotor: 1, MaxWidth: 3}
	d := Design{Cells: []Cell{{0, 0, KMotor}, {1, 0, KMotor}, {2, 0, KFrame}, {3, 0, KWheel}, {1, 1, KMast}}}
	if got := len(RuleProblems(def, d)); got != 3 {
		t.Fatalf("want 3 rule problems, got %v", RuleProblems(def, d))
	}
	ok := Design{Cells: []Cell{{0, 0, KMotor}, {1, 0, KFrame}, {2, 0, KWheel}, {1, 1, KMast}}}
	if p := RuleProblems(def, ok); len(p) != 0 {
		t.Fatal(p)
	}
	ses := NewSession(&Level{LevelDef: def})
	ses.Design = ok
	if ses.CanAdd(KFrame) != 0 || ses.CanAdd(KMotor) != 0 {
		t.Fatalf("tray should be capped: frame %d motor %d", ses.CanAdd(KFrame), ses.CanAdd(KMotor))
	}
	if RuleLine(def) == "" || RuleLine(LevelDef{}) != "" {
		t.Fatal("rule line")
	}
}

// A short rover drops into a crevasse and is lost; a long braced one with
// motors spread along it bridges it.
func TestCrevasseNeedsALongRover(t *testing.T) {
	l := GetLevel(9)
	if len(l.Gaps) != 2 {
		t.Fatalf("want 2 crevasses, got %v", l.Gaps)
	}
	ds := referenceDesigns()
	if r := drive(l, ds["low"], 120); r.Outcome != Fell {
		t.Errorf("low rover: %v, want it lost in a crevasse", r.Outcome.Message())
	}
	if r := drive(l, ds["tri"], 120); r.Outcome != Won {
		t.Errorf("tri rover: %v", r.Outcome.Message())
	}
}
