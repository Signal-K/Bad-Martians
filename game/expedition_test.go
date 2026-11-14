package game

import (
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

func TestExpeditionBudgetAndCarryover(t *testing.T) {
	e := NewExpedition()
	rover := StarterDesign()
	e.Finish(true, rover, 700, 10, 14)
	if e.Index != 1 || e.Score != 700 || e.Credits != ExpeditionStartingCredits+UnderParCredits {
		t.Fatalf("after win: %+v", e)
	}
	ses := ExpeditionSession(engine.NewMemHost(), e)
	if ses.Design.Cost() != rover.Cost() || !ses.Fits() {
		t.Fatalf("surviving rover missing: %+v", ses.Design)
	}
	before := e.Credits
	e.Finish(false, rover, 0, 20, 14)
	if e.Lost != rover.Cost() || e.Credits != before-rover.Cost() || len(e.Rover.Cells) != 0 {
		t.Fatalf("after crash: %+v", e)
	}
	e.Credits = MinRover() - 1 + rover.Cost()
	e.Finish(false, rover, 0, 20, 14)
	if !e.Failed {
		t.Fatal("insufficient funds should end expedition")
	}
}

func TestExpeditionParRewards(t *testing.T) {
	for _, tc := range []struct {
		elapsed float64
		want    int
	}{{13, UnderParCredits}, {14, 0}, {15, 0}, {0, 0}} {
		e := NewExpedition()
		e.Finish(true, StarterDesign(), 500, tc.elapsed, 14)
		if e.ParReward != tc.want || e.Credits != ExpeditionStartingCredits+tc.want {
			t.Fatalf("time %v: %+v", tc.elapsed, e)
		}
	}
	e := NewExpedition()
	for range ExpeditionLevels {
		e.Finish(true, StarterDesign(), 500, 10, 14)
	}
	if !e.Done || e.Score != 1500 || e.Credits != ExpeditionStartingCredits+2*UnderParCredits || e.ParReward != 0 {
		t.Fatalf("completed: %+v", e)
	}
}

func TestExpeditionBayLimitsByCredits(t *testing.T) {
	e := NewExpedition()
	rover := StarterDesign()
	e.Credits = rover.Cost()
	s := ExpeditionSession(engine.NewMemHost(), e)
	s.Design = StarterDesign()
	if !s.Fits() || len(s.Problems()) != 0 || s.CanAdd(KWheel) != 0 {
		t.Fatalf("budget: %v", s.Problems())
	}
	s.Design.Set(6, 0, KWheel)
	if s.Fits() || len(s.Problems()) == 0 {
		t.Fatal("over-budget rover must not fit")
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
	rover := StarterDesign()
	e.Credits = rover.Cost() // one crash exhausts the budget
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

func TestExpeditionResultAwardsOnce(t *testing.T) {
	h := newHarness(t)
	e := NewExpedition()
	ses := ExpeditionSession(h.h, e)
	ses.Design = StarterDesign()
	result := &ResultArg{Session: ses, Outcome: Won, Time: 10, Score: Score{Finish: 500}, Damaged: true}
	h.g.Load(SceneResult, result)
	h.frames(2, engine.Input{})
	credits := e.Credits
	if credits != ExpeditionStartingCredits+UnderParCredits {
		t.Fatalf("credits %d", credits)
	}
	h.g.Load(SceneResult, result)
	h.frames(2, engine.Input{})
	if e.Credits != credits || e.Index != 1 {
		t.Fatalf("result paid twice: %+v", e)
	}
	h.clickButton("Next mission")
	if h.g.Scene().Name != SceneBuild || len(e.Rover.Cells) == 0 {
		t.Fatal("surviving rover should reach the next build")
	}
}

func TestExpeditionRetryStartsWithEmptyBay(t *testing.T) {
	h := newHarness(t)
	e := NewExpedition()
	ses := ExpeditionSession(h.h, e)
	ses.Design = StarterDesign()
	h.g.Load(SceneResult, &ResultArg{Session: ses, Outcome: Fell})
	h.frames(2, engine.Input{})
	h.clickButton("Retry")
	if h.g.Scene().Name != SceneBuild || len(ses.Design.Cells) != 0 {
		t.Fatal("lost rover must not return on retry")
	}
}
