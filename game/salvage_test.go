package game

import (
	"encoding/json"
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

func TestSalvagePicksPriciestPartButNotTheMast(t *testing.T) {
	if k, ok := SalvagePart(StarterDesign()); !ok || k != KMotor {
		t.Fatalf("starter salvage %q, want motor wheel", k)
	}
	d := Design{Cells: []Cell{{0, 0, KWheel}, {1, 0, KFrame}, {1, 1, KMast}}}
	if k, _ := SalvagePart(d); k != KWheel {
		t.Fatalf("salvage %q, want wheel", k)
	}
	if _, ok := SalvagePart(Design{Cells: []Cell{{0, 0, KMast}}}); ok {
		t.Fatal("a mast is never salvaged")
	}
}

// The scrape counter decides salvage: the starter rover glides through level
// 1 untouched but drags its frame on the sand trap.
func TestUndamagedRunsAreDetected(t *testing.T) {
	if r := drive(GetLevel(1), StarterDesign(), 120); r.Outcome != Won || r.Damaged() {
		t.Fatalf("level 1 starter: outcome %v scrapes %d", r.Outcome, r.Scrapes)
	}
	if r := drive(GetLevel(2), StarterDesign(), 120); !r.Damaged() {
		t.Fatalf("level 2 starter should scrape, scrapes %d", r.Scrapes)
	}
}

func TestExpeditionSalvageGoesInTheKit(t *testing.T) {
	e := NewExpedition()
	motors := e.Inv[KMotor]
	e.Finish(true, StarterDesign(), 600)
	if k := e.Salvage(StarterDesign()); k != KMotor || e.Inv[KMotor] != motors+1 || e.Salvaged != KMotor {
		t.Fatalf("salvage should add a motor wheel to the kit: %+v", e)
	}
	e.Finish(false, StarterDesign(), 0)
	if e.Salvaged != "" {
		t.Fatalf("a crash salvages nothing: %+v", e)
	}
}

// A single mission's salvage is worth its credits on the next mission.
func TestSpareAddsCreditsNextMission(t *testing.T) {
	s := NewSession(GetLevel(2))
	base := s.Budget()
	s.Spare = KMotor
	if p, _ := PartOf(KMotor); s.Budget() != base+p.Cost {
		t.Fatalf("budget %d, want %d", s.Budget(), base+p.Cost)
	}
}

func TestNames(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < len(nameFirst)*len(nameLast); i++ {
		n := SuggestName(i)
		if seen[n] || len(n) > MaxNameLen {
			t.Fatalf("name %d %q repeats or is too long", i, n)
		}
		seen[n] = true
	}
	for in, want := range map[string]string{
		"  Plucky   Mule ": "Plucky Mule", "Rövér": "Rvr", "a very very long rover name": "a very very long r", "": "",
	} {
		if got := CleanName(in); got != want {
			t.Errorf("CleanName(%q) = %q, want %q", in, got, want)
		}
	}
	for in, want := range map[string]string{
		"Plucky Mule": "Plucky Mule II", "Plucky Mule II": "Plucky Mule III", "Stubborn Tortoise": "Stubborn Tortoi II",
	} {
		if got := SuccessorName(in); got != want || len(got) > MaxNameLen {
			t.Errorf("SuccessorName(%q) = %q, want %q", in, got, want)
		}
	}
}

// The launch step names the rover: typing replaces the suggestion, a host
// prompt replaces the whole name, and the name reaches the run log.
func TestLaunchNamesTheRover(t *testing.T) {
	h := newHarness(t)
	ses := NewSessionFor(h.g.Host, GetLevel(1))
	ses.Tut, ses.Design = nil, StarterDesign()
	h.g.Load(SceneLaunch, ses)
	h.frames(2, engine.Input{})
	if ses.Name != SuggestName(0) {
		t.Fatalf("suggested %q", ses.Name)
	}
	h.frames(1, engine.Input{Typed: []rune("Dusty")})
	h.frames(1, engine.Input{Backspace: true})
	h.clickButton("Rover name") // a phone asks the host for a text box
	if h.count("prompt") != 1 {
		t.Fatal("tapping the name should ask the host for a prompt")
	}
	h.frames(1, engine.Input{Replace: "  Wren  ", Replacing: true})
	h.frames(1, engine.Input{Typed: []rune(" 2")})
	h.frames(1, engine.Input{Enter: true})
	h.frames(1, engine.Input{})
	if h.g.Scene().Name != SceneRun || ses.Name != "Wren 2" {
		t.Fatalf("scene %s name %q", h.g.Scene().Name, ses.Name)
	}
	if LoadProgress(h.h).Launched != 1 {
		t.Fatal("launch should advance the name generator")
	}
	for i := 0; i < 60*60 && h.g.Scene().Name == SceneRun; i++ {
		h.g.Tick(engine.FixedDT, engine.Input{Right: true})
	}
	h.frames(2, engine.Input{})
	if h.g.Scene().Name != SceneResult {
		t.Fatalf("expected result, got %s", h.g.Scene().Name)
	}
	var ev struct {
		Rover string `json:"rover"`
		Won   bool   `json:"won"`
	}
	for _, e := range h.h.Events {
		if e.Name == "run" {
			_ = json.Unmarshal(e.Payload, &ev)
		}
	}
	if ev.Rover != "Wren 2" || !ev.Won {
		t.Fatalf("run event %+v", ev)
	}
}
