package game

import (
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

func TestExpeditionRules(t *testing.T) {
	e := NewExpedition()
	if e.Inv[KWheel] != ExpeditionKit[KWheel] || e.Inv.Total() != ExpeditionKit.Total() {
		t.Fatalf("start kit %v", e.Inv)
	}
	rover := StarterDesign()
	// A win keeps the parts; there are no supply drops.
	e.Finish(true, rover, 700)
	if e.Index != 1 || e.Score != 700 || e.Inv.Total() != ExpeditionKit.Total() {
		t.Fatalf("after win: %+v", e)
	}
	// A crash loses exactly the parts on the rover, kind by kind.
	before := e.Inv[KFrame]
	e.Finish(false, rover, 0)
	if e.Lost[KFrame] != 3 || e.Inv[KFrame] != before-3 || e.Lost.Total() != len(rover.Cells) {
		t.Fatalf("after crash: %+v", e)
	}
	// No mast left ends the expedition.
	for !e.Over() {
		e.Finish(false, rover, 0)
	}
	if !e.Failed || e.Inv[KMast] != 0 {
		t.Fatalf("should have failed on masts: %+v", e)
	}
}

func TestExpeditionCompletes(t *testing.T) {
	e := NewExpedition()
	for range ExpeditionLevels {
		e.Finish(true, StarterDesign(), 500)
	}
	if !e.Done || e.Score != 500*len(ExpeditionLevels) {
		t.Fatalf("not complete: %+v", e)
	}
}

// The bay counts what is left in the kit, part by part.
func TestExpeditionBayLimitsByKit(t *testing.T) {
	e := NewExpedition()
	e.Inv[KWheel] = 2
	s := ExpeditionSession(engine.NewMemHost(), e)
	s.Design = StarterDesign() // uses 1 wheel
	if n := s.CanAdd(KWheel); n != 1 {
		t.Fatalf("can add %d wheels, want 1", n)
	}
	s.Design.Set(5, 0, KWheel)
	s.Design.Set(4, 0, KFrame)
	if !s.Fits() || len(s.Problems()) != 0 {
		t.Fatalf("2 of 2 wheels should fit: %v", s.Problems())
	}
	s.Design.Set(6, 0, KWheel)
	if s.Fits() || len(s.Problems()) == 0 {
		t.Fatal("a third wheel must not fit a kit of 2")
	}
}

func TestExpeditionOpensFromMenuThroughBriefing(t *testing.T) {
	h := newHarness(t)
	h.g.Load(SceneMenu, nil)
	h.frames(2, engine.Input{})
	h.clickButton("Expedition · 3 missions")
	if h.g.Scene().Name != SceneBriefing {
		t.Fatalf("expected the briefing, got %s", h.g.Scene().Name)
	}
	h.frames(60, engine.Input{})
	h.clickButton("Start Mission 1")
	if h.g.Scene().Name != SceneBuild {
		t.Fatalf("expected build, got %s", h.g.Scene().Name)
	}
}

func TestExpeditionResultShowsLossAndEnds(t *testing.T) {
	h := newHarness(t)
	e := NewExpedition()
	e.Inv[KMast] = 1 // one crash will lose the last mast
	ses := ExpeditionSession(h.g.Host, e)
	ses.Design = StarterDesign()
	h.g.Load(SceneResult, &ResultArg{Session: ses, Outcome: Fell})
	h.frames(3, engine.Input{})
	if !e.Failed {
		t.Fatalf("expedition should be over: %+v", e)
	}
	h.clickButton("Finish")
	if h.g.Scene().Name != SceneMenu {
		t.Fatalf("expected menu, got %s", h.g.Scene().Name)
	}
}
