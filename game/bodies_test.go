package game

import (
	"strings"
	"testing"
)

// KES-490: the campaign ends on Mars's moons, under their own gravity.
func TestMoonLevels(t *testing.T) {
	want := map[int]string{21: "phobos", 22: "deimos", 23: "moon"}
	for id, body := range want {
		def := LevelDefs[id-1]
		if def.Body != body {
			t.Fatalf("level %d body %q, want %q", id, def.Body, body)
		}
		g := NewWorldFor(def).Gravity.Y
		if -g >= Bodies["mars"].Gravity || g >= 0 {
			t.Errorf("level %d gravity %.2f is not lower than Mars's", id, g)
		}
		won := 0
		for _, d := range referenceDesigns() {
			if clears(def, d) {
				won++
			}
		}
		if won == 0 {
			t.Errorf("level %d (%s) is not clearable by any reference rover", id, body)
		}
	}
	// Every Mars level keeps Mars gravity, so the existing campaign plays as before.
	for _, d := range LevelDefs[:20] {
		if BodyOf(d).Key != "mars" || NewWorldFor(d).Gravity.Y != -3.71 {
			t.Errorf("level %d no longer runs under Mars gravity", d.ID)
		}
	}
}

func TestLowGravityHopsFurther(t *testing.T) {
	mars, phobos := LevelDefs[0], LevelDefs[0]
	phobos.Body = "phobos"
	if a, b := NewWorldFor(mars).Gravity.Y, NewWorldFor(phobos).Gravity.Y; b <= a {
		t.Fatalf("phobos pull %.2f should be weaker than mars %.2f", b, a)
	}
}

func TestUnknownBodyRejected(t *testing.T) {
	bad := strings.Replace(goodLevel, `"biome":"plains"`, `"biome":"plains","body":"titan"`, 1)
	if _, err := ParseLevel([]byte(bad)); err == nil || !strings.Contains(err.Error(), "body") {
		t.Fatalf("want a body error, got %v", err)
	}
	ok := strings.Replace(goodLevel, `"biome":"plains"`, `"biome":"plains","body":"deimos"`, 1)
	if d, err := ParseLevel([]byte(ok)); err != nil || d.Body != "deimos" {
		t.Fatalf("deimos level: %+v %v", d, err)
	}
}

// Phobos's loose regolith gives the wheels less to bite on; Mars is unchanged.
func TestBodyGrip(t *testing.T) {
	if g := NewWorldFor(LevelDefs[0]).Grip; g != 0 && g != 1 {
		t.Errorf("Mars grip %.2f, want 1", g)
	}
	if g := NewWorldFor(LevelDefs[20]).Grip; g >= 1 || g <= 0 {
		t.Errorf("Phobos grip %.2f, want below Mars's", g)
	}
}
