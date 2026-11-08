package game

import (
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

func TestExpeditionRules(t *testing.T) {
	e := NewExpedition()
	start := e.Parts
	if start != GetLevel(ExpeditionLevels[0]).Credits {
		t.Fatalf("start parts %d", start)
	}
	// A win keeps the parts and adds the next mission's supplies.
	e.Finish(true, 10, 700)
	if e.Index != 1 || e.Score != 700 || e.Parts != start+GetLevel(ExpeditionLevels[1]).Credits {
		t.Fatalf("after win: %+v", e)
	}
	// A crash loses the parts the rover used.
	before := e.Parts
	e.Finish(false, 8, 0)
	if e.Lost != 8 || e.Parts != before-8 {
		t.Fatalf("after crash: %+v", e)
	}
	// Running out of parts ends the expedition.
	e.Finish(false, e.Parts, 0)
	if !e.Failed || !e.Over() {
		t.Fatalf("should have failed: %+v", e)
	}
}

func TestExpeditionCompletes(t *testing.T) {
	e := NewExpedition()
	for range ExpeditionLevels {
		e.Finish(true, 5, 500)
	}
	if !e.Done || e.Score != 500*len(ExpeditionLevels) {
		t.Fatalf("not complete: %+v", e)
	}
}

func TestExpeditionBudgetIsInventory(t *testing.T) {
	e := NewExpedition()
	s := ExpeditionSession(engine.NewMemHost(), e)
	if s.Budget() != e.Parts {
		t.Fatalf("budget %d, parts %d", s.Budget(), e.Parts)
	}
}

func TestExpeditionOpensFromMenu(t *testing.T) {
	h := newHarness(t)
	h.g.Load(SceneMenu, nil)
	h.frames(2, engine.Input{})
	h.clickButton("Expedition · 3 missions")
	if h.g.Scene().Name != SceneBuild {
		t.Fatalf("expected build, got %s", h.g.Scene().Name)
	}
}

func TestExpeditionResultShowsLossAndEnds(t *testing.T) {
	h := newHarness(t)
	e := NewExpedition()
	e.Parts = MinRover() // one crash will empty the inventory
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
