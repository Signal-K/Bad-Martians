package game

import "testing"

func groundLevel(t *testing.T) *Level {
	t.Helper()
	for _, d := range LevelDefs {
		l := BuildLevel(d)
		for i := 1; i < len(l.Terrain.Seg); i++ {
			if l.Terrain.Seg[i] != l.Terrain.Seg[i-1] {
				return l
			}
		}
	}
	t.Fatal("no level with more than one class")
	return nil
}

func TestGroundCrustFadesBetweenStretches(t *testing.T) {
	lvl := groundLevel(t)
	g := BuildGround(lvl.Terrain)
	seg := lvl.Terrain.Seg
	cuts := 0
	for i := 1; i < len(seg); i++ {
		if seg[i] != seg[i-1] {
			cuts++
		}
	}
	if cuts == 0 {
		t.Fatal("level has a single class; pick one with stretches")
	}
	// The old art jumped the full colour difference in one sample; the fade
	// must keep each step small.
	for i := 1; i < len(g.Crust); i++ {
		a, b := g.Crust[i-1], g.Crust[i]
		for _, d := range []int{int(a.R) - int(b.R), int(a.G) - int(b.G), int(a.B) - int(b.B)} {
			if d > 24 || d < -24 {
				t.Fatalf("crust colour jumps %d at sample %d", d, i)
			}
		}
	}
}

func TestGroundLayersAreOrderedAndDeterministic(t *testing.T) {
	lvl := groundLevel(t)
	a, b := BuildGround(lvl.Terrain), BuildGround(lvl.Terrain)
	for i := range a.CrustD {
		if !(a.CrustD[i] > 0 && a.CrustD[i] <= a.PebD[i] && a.PebD[i] <= a.RockD[i]) {
			t.Fatalf("sample %d: crust %.2f pebbles %.2f rock %.2f out of order", i, a.CrustD[i], a.PebD[i], a.RockD[i])
		}
		if a.CrustD[i] != b.CrustD[i] || a.RockD[i] != b.RockD[i] || a.Crust[i] != b.Crust[i] {
			t.Fatalf("sample %d differs between builds", i)
		}
	}
	if a.LayerAt(0, 0) != LayerCrust || a.LayerAt(0, 50) != LayerRock {
		t.Fatal("LayerAt does not run crust to rock")
	}
}

func TestGroundHidesInterestingRocks(t *testing.T) {
	for _, d := range LevelDefs {
		lvl := BuildLevel(d)
		g := GroundOf(lvl.Terrain)
		if len(g.Finds) < 2 {
			t.Fatalf("%s: %d finds, want at least 2", lvl.Name, len(g.Finds))
		}
		for _, f := range g.Finds {
			if f.X < 0 || f.X > lvl.Length {
				t.Errorf("%s: find at %.1f m is off the course (0–%.0f)", lvl.Name, f.X, lvl.Length)
			}
			i := int((f.X - lvl.Terrain.X0) / lvl.Terrain.DX)
			if got := g.LayerAt(i, f.Depth); got != f.Layer {
				t.Errorf("%s: find says layer %d but sits in %d", lvl.Name, f.Layer, got)
			}
		}
	}
}
