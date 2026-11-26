package game

import (
	"strings"
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

// sceneWords is how many words a scene draws on its first frames.
func sceneWords(t *testing.T, h *harness, scene string) int {
	t.Helper()
	if !h.g.OpenStandalone(scene) {
		t.Fatalf("no scene %q", scene)
	}
	h.frames(3, engine.Input{})
	r := &textR{}
	h.g.Draw(r)
	return len(strings.Fields(strings.Join(r.texts, " ")))
}

func TestFirstOpenSkipsToBuild(t *testing.T) {
	h := engine.NewMemHost()
	if !FirstOpen(h) {
		t.Fatal("an empty save is a first open")
	}
	g := engine.NewGame(ScreenW, ScreenH, h)
	Setup(g)
	g.Load(SceneBuild, FirstMissionSession(h))
	if FirstMissionSession(h).Tut == nil {
		t.Fatal("the first open must start the tutorial")
	}
	FinishTutorial(h, nil)
	if FirstOpen(h) {
		t.Fatal("a finished tutorial is a returning player")
	}
}

func TestFirstOpenEndsOnFirstWin(t *testing.T) {
	h := engine.NewMemHost()
	p := LoadProgress(h)
	p.Record(LevelDefs[0].ID, 1, 30, 100)
	p.Save(h)
	if FirstOpen(h) {
		t.Fatal("a player with a win is not on a first open")
	}
}

func TestDisclosureTiers(t *testing.T) {
	p := &Progress{Levels: map[int]LevelProgress{}}
	for want, wins := range map[int]int{TierFirst: 0, TierLearning: tierLearningWins, TierFull: tierFullWins} {
		p.Levels = map[int]LevelProgress{}
		for _, d := range LevelDefs[:wins] {
			p.Levels[d.ID] = LevelProgress{Stars: 1}
		}
		if got := DisclosureTier(p); got != want {
			t.Errorf("%d wins: tier %d, want %d", wins, got, want)
		}
	}
}

func TestMapHidesLaterSystemsUntilEarned(t *testing.T) {
	h := newHarness(t)
	h.g.Load(SceneMap, nil)
	h.frames(2, engine.Input{})
	r := &textR{}
	h.g.Draw(r)
	text := strings.Join(r.texts, "\n")
	for _, hidden := range []string{"Daily mission", "Expedition", "Account", "stars", "Travelled"} {
		if strings.Contains(text, hidden) {
			t.Errorf("a new player's map shows %q", hidden)
		}
	}
	h.unlockAll()
	h.g.Load(SceneMap, nil)
	h.frames(2, engine.Input{})
	r = &textR{}
	h.g.Draw(r)
	text = strings.Join(r.texts, "\n")
	for _, shown := range []string{"Daily mission", "stars", "Account"} {
		if !strings.Contains(text, shown) {
			t.Errorf("an experienced player's map hides %q", shown)
		}
	}
}

// Playtest feedback: complexity can be approximated by the amount of text a
// player has to read, so each screen gets a word budget. Raise one only on
// purpose, and with a reason in docs/playtest/feedback-2026-11-26.md.
func TestSceneWordBudgets(t *testing.T) {
	budgets := map[string]int{
		SceneMap:      45, // a new player's map: title, one chapter, one button
		SceneBuild:    70, // the first bay, tutorial line included
		SceneLaunch:   60,
		SceneBriefing: 80,
	}
	for scene, max := range budgets {
		h := newHarness(t)
		n := sceneWords(t, h, scene)
		t.Logf("%s: %d words (budget %d)", scene, n, max)
		if n > max {
			t.Errorf("%s draws %d words, over its budget of %d", scene, n, max)
		}
	}
}
