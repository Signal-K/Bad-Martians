package game

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

type nullR struct{ calls int }

type textR struct {
	nullR
	texts []string
}

func (r *textR) Text(s string, x, y, sz float64, c engine.Color, a engine.Align) {
	r.texts = append(r.texts, s)
}

func (n *nullR) Line(a, b engine.Vec2, w float64, c engine.Color)          { n.calls++ }
func (n *nullR) Circle(c engine.Vec2, r float64, col engine.Color, f bool) { n.calls++ }
func (n *nullR) Rect(x, y, w, h float64, c engine.Color, f bool)           { n.calls++ }
func (n *nullR) Poly(p []engine.Vec2, c engine.Color)                      { n.calls++ }
func (n *nullR) Text(s string, x, y, sz float64, c engine.Color, a engine.Align) {
	n.calls++
}

type harness struct {
	t *testing.T
	g *engine.Game
	h *engine.MemHost
}

func newHarness(t *testing.T) *harness {
	h := engine.NewMemHost()
	FinishTutorial(h, nil) // these tests drive the returning player's flow
	g := engine.NewGame(ScreenW, ScreenH, h)
	Setup(g)
	hs := &harness{t, g, h}
	hs.frames(2, engine.Input{})
	return hs
}

func (h *harness) frames(n int, in engine.Input) {
	for i := 0; i < n; i++ {
		h.g.Tick(engine.FixedDT, in)
	}
	h.g.Draw(&nullR{}) // every frame must be drawable
}

// click presses and releases at (x,y) on separate frames.
func (h *harness) click(x, y float64) {
	p := engine.V(x, y)
	h.frames(1, engine.Input{Pointer: p, Down: true})
	h.frames(1, engine.Input{Pointer: p})
	h.frames(1, engine.Input{Pointer: p}) // applies pending scene load
}

func (h *harness) count(name string) int {
	n := 0
	for _, e := range h.h.Events {
		if e.Name == name {
			n++
		}
	}
	return n
}

func (h *harness) button(label string) *Button {
	for _, o := range h.g.Scene().Objects {
		if b, ok := engine.Get[*Button](o); ok && b.Label == label {
			return b
		}
	}
	h.t.Fatalf("no button %q in scene %s", label, h.g.Scene().Name)
	return nil
}

func (h *harness) clickButton(label string) {
	b := h.button(label)
	h.click(b.X+b.W/2, b.Y+b.H/2)
}

func (h *harness) menuTile(id int) *Button { return h.button(fmt.Sprintf("Mission %d", id)) }

func TestFullFlow(t *testing.T) {
	h := newHarness(t)
	h.g.Load(SceneMenu, nil)
	h.frames(2, engine.Input{})
	// build first: a mission card goes straight to the rover bay, no labelling gate
	h.clickButton("Mission 1")
	if h.g.Scene().Name != SceneBuild {
		t.Fatalf("expected build, got %s", h.g.Scene().Name)
	}
	// the bay starts empty: nothing to drive until the player builds a rover
	h.clickButton("GO")
	if h.g.Scene().Name != SceneBuild {
		t.Fatalf("GO with an empty bay should stay in build, got %s", h.g.Scene().Name)
	}
	ses := NewSessionFor(h.g.Host, GetLevel(1))
	ses.Design = StarterDesign()
	h.g.Load(SceneBuild, ses)
	h.frames(2, engine.Input{})
	h.clickButton("GO")
	if h.g.Scene().Name != SceneLaunch {
		t.Fatalf("expected launch, got %s", h.g.Scene().Name)
	}
	h.clickButton("Send it out")
	if h.g.Scene().Name != SceneRun {
		t.Fatalf("expected run, got %s", h.g.Scene().Name)
	}
	for i := 0; i < 60*60 && h.g.Scene().Name == SceneRun; i++ {
		h.g.Tick(engine.FixedDT, engine.Input{Right: true})
	}
	h.frames(2, engine.Input{})
	if h.g.Scene().Name != SceneResult {
		t.Fatalf("expected result, got %s", h.g.Scene().Name)
	}
	var ev struct {
		Won   bool `json:"won"`
		Stars int  `json:"stars"`
		Score int  `json:"score"`
	}
	var last engine.HostEvent
	for _, e := range h.h.Events {
		if e.Name == "run" {
			last = e
		}
	}
	if last.Name != "run" {
		t.Fatal("run event should be emitted to the host")
	}
	_ = json.Unmarshal(last.Payload, &ev)
	if !ev.Won || ev.Stars < 1 || ev.Score < 500 {
		t.Fatalf("starter should win level 1 with a score: %s", last.Payload)
	}
	prog := LoadProgress(h.h)
	if !prog.Unlocked(2) || prog.Levels[1].Score != ev.Score {
		t.Fatalf("level 2 should unlock and score be saved: %+v", prog.Levels[1])
	}
	// The unfinished review flow is not part of the normal mission path.
	if h.count("label") != 0 || h.count("review") != 0 {
		t.Fatal("a mission should not submit terrain reviews")
	}
	h.clickButton("Next mission")
	if h.g.Scene().Name != SceneBuild {
		t.Fatalf("expected build for level 2, got %s", h.g.Scene().Name)
	}
}

func TestBuildPainting(t *testing.T) {
	h := newHarness(t)
	ses := NewSession(GetLevel(1))
	ses.Design = Design{}
	h.g.Load(SceneBuild, ses)
	h.frames(2, engine.Input{})
	cell := func(cx, cy int) engine.Vec2 {
		return engine.V(bGX+float64(cx)*bCell+bCell/2, bGY+float64(GridH-1-cy)*bCell+bCell/2)
	}
	h.click(cell(0, 0).X, cell(0, 0).Y) // default tool is wheel
	if k, ok := ses.Design.At(0, 0); !ok || k != KWheel {
		t.Fatalf("expected a wheel at 0,0: %+v", ses.Design)
	}
	h.frames(1, engine.Input{Pointer: cell(2, 3), Down: true})
	h.frames(1, engine.Input{Pointer: cell(3, 3), Down: true})
	h.frames(1, engine.Input{Pointer: cell(4, 3), Down: true})
	h.frames(1, engine.Input{Pointer: cell(4, 3)})
	for x := 2; x <= 4; x++ {
		if _, ok := ses.Design.At(x, 3); !ok {
			t.Fatalf("drag should paint %d,3", x)
		}
	}
	if h.button("GO").Disabled == false {
		t.Fatal("GO must be disabled for an invalid rover")
	}
}

func TestMenuLocksLevels(t *testing.T) {
	h := newHarness(t)
	h.g.Load(SceneMenu, nil)
	h.frames(2, engine.Input{})
	b := h.menuTile(2)
	h.click(b.X+b.W/2, b.Y+b.H/2) // level 2 is locked
	if h.g.Scene().Name != SceneMenu {
		t.Fatal("locked level should not open")
	}
}

func TestProgressKeepsBest(t *testing.T) {
	p := &Progress{Levels: map[int]LevelProgress{}}
	p.Record(1, 2, 20, 900)
	p.Record(1, 1, 15, 700)
	if p.Levels[1].Stars != 2 || p.Levels[1].Best != 15 || p.Levels[1].Score != 900 {
		t.Fatalf("%+v", p.Levels[1])
	}
}

func TestLabelBonus(t *testing.T) {
	l := GetLevel(1)
	s := NewSession(l)
	gold := -1
	for i, sg := range l.Segments {
		if sg.Gold {
			gold = i
			break
		}
	}
	s.Labels[gold] = l.Segments[gold].Class
	if s.Bonus() != 3 {
		t.Fatalf("correct gold label = +3, got %d", s.Bonus())
	}
	s.Labels[gold] = (l.Segments[gold].Class + 1) % NumClasses
	if s.Bonus() != 1 {
		t.Fatalf("wrong gold label = +1, got %d", s.Bonus())
	}
}

func TestLegacyReviewCreditsSurviveChangedTerrain(t *testing.T) {
	h := engine.NewMemHost()
	l := GetLevel(1)
	p := LoadProgress(h)
	p.Labels[l.ID] = map[int]int{len(l.Segments) + 1: 0}
	p.Save(h)
	s := NewSessionFor(h, l)
	if s.Bonus() != 1 || s.Budget() < l.Credits+1 {
		t.Fatalf("legacy review credit should survive terrain changes: bonus=%d budget=%d", s.Bonus(), s.Budget())
	}
	if len(LoadProgress(h).Labels[l.ID]) != 1 {
		t.Fatal("loading a mission must preserve its saved reviews")
	}
}

func TestDailyMissionOpensWithReward(t *testing.T) {
	h := newHarness(t)
	h.g.Load(SceneMenu, nil)
	h.frames(2, engine.Input{})
	h.clickButton("Daily mission") // not locked behind the campaign
	if h.g.Scene().Name != SceneBuild {
		t.Fatalf("expected build, got %s", h.g.Scene().Name)
	}
	m, _, _ := CurrentMission()
	ses := NewSessionFor(h.g.Host, BuildLevel(m.Level))
	if want := m.Level.Credits + m.Reward.Credits; ses.Budget() != want {
		t.Fatalf("budget %d, want %d", ses.Budget(), want)
	}
}

func TestDailyMissionOnlyAdvertisesAvailableRewards(t *testing.T) {
	h := newHarness(t)
	h.g.Load(SceneMenu, nil)
	h.frames(2, engine.Input{})
	r := &textR{}
	h.g.Draw(r)
	text := strings.Join(r.texts, "\n")
	if strings.Contains(text, "free retry") || !strings.Contains(text, "build credits") {
		t.Fatalf("daily card advertises unavailable reward or hides build credits: %s", text)
	}
}

func TestCurrentMissionByDate(t *testing.T) {
	old := Today
	defer func() { Today = old }()
	Today = func() string { return "2026-01-01" }
	if _, ok, next := CurrentMission(); ok || next == nil {
		t.Fatalf("before the first mission: ok=%v next=%v", ok, next)
	}
	Today = func() string { return "2030-01-01" }
	if m, ok, _ := CurrentMission(); !ok || m.Date != "2026-11-08" {
		t.Fatalf("later date should show the latest mission, got %v %v", m.Date, ok)
	}
}
