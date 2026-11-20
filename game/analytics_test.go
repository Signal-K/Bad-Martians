package game

import (
	"encoding/json"
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

type tracked struct {
	Event string         `json:"event"`
	Props map[string]any `json:"props"`
}

func (h *harness) tracked(event string) []tracked {
	var out []tracked
	for _, e := range h.h.Events {
		if e.Name != "track" {
			continue
		}
		var t tracked
		if err := json.Unmarshal(e.Payload, &t); err != nil {
			h.t.Fatalf("bad track payload %s: %v", e.Payload, err)
		}
		if t.Event == event {
			out = append(out, t)
		}
	}
	return out
}

// The tutorial reports each step once as the player reaches it, and its end.
func TestAnalyticsTutorialSteps(t *testing.T) {
	h := &harness{t: t, h: engine.NewMemHost()}
	h.g = engine.NewGame(ScreenW, ScreenH, h.h)
	Setup(h.g)
	ses := NewSessionFor(h.h, GetLevel(1))
	h.g.Load(SceneBuild, ses)
	h.frames(4, engine.Input{})
	if got := h.tracked(EvTutorialStep); len(got) != 1 || got[0].Props["name"] != "beam" {
		t.Fatalf("an empty bay should report the beam step once, got %+v", got)
	}
	ses.Design = StarterDesign() // a whole rover jumps straight to the test step
	h.frames(4, engine.Input{})
	got := h.tracked(EvTutorialStep)
	if len(got) != 2 || got[1].Props["name"] != "test" {
		t.Fatalf("the starter rover should move the tutorial to the test step, got %+v", got)
	}
	h.clickButton("Skip")
	if d := h.tracked(EvTutorialDone); len(d) != 1 || d[0].Props["skipped"] != true {
		t.Fatalf("skip should report the tutorial done and skipped, got %+v", d)
	}
	h.frames(4, engine.Input{})
	if n := len(h.tracked(EvTutorialStep)); n != 2 {
		t.Fatalf("no steps after the tutorial ends, got %d", n)
	}
}

// A test drop reports its verdict once it settles; GO reports the build;
// the run's result says how far it got; Abort reports an abandoned run.
func TestAnalyticsBuildRunAbort(t *testing.T) {
	h := newHarness(t)
	ses := NewSessionFor(h.g.Host, GetLevel(1))
	ses.Design = StarterDesign()
	h.g.Load(SceneBuild, ses)
	h.frames(2, engine.Input{})
	h.clickButton("Test")
	h.frames(int(testSettle/engine.FixedDT)+10, engine.Input{})
	drops := h.tracked(EvTestDrop)
	if len(drops) != 1 || drops[0].Props["verdict"] == "" || drops[0].Props["level"] != 1.0 {
		t.Fatalf("one settled verdict expected, got %+v", drops)
	}
	h.clickButton("Test") // back to the grid
	h.clickButton("GO")
	b := h.tracked(EvBuildLaunched)
	if len(b) != 1 || b[0].Props["cost"] != float64(ses.Design.Cost()) || b[0].Props["body"] != "mars" {
		t.Fatalf("GO should report the build once with its cost, got %+v", b)
	}
	if parts, ok := b[0].Props["parts"].(map[string]any); !ok || len(parts) == 0 {
		t.Fatalf("the build should list its parts, got %+v", b[0].Props)
	}
	h.clickButton("Send it out")
	h.frames(30, engine.Input{Right: true})
	h.clickButton("Abort")
	a := h.tracked(EvRunAbandoned)
	if len(a) != 1 || a[0].Props["how"] != "abort" {
		t.Fatalf("Abort should report an abandoned run, got %+v", a)
	}
	if p, _ := a[0].Props["progress"].(float64); p < 0 || p > 1 {
		t.Fatalf("progress should be a fraction, got %v", a[0].Props["progress"])
	}
}

// A finished run's event carries how far it got, for the playtest funnel.
func TestAnalyticsRunCarriesProgress(t *testing.T) {
	h := newHarness(t)
	ses := NewSessionFor(h.g.Host, GetLevel(1))
	ses.Design = StarterDesign()
	h.g.Load(SceneRun, ses)
	h.frames(2, engine.Input{})
	for i := 0; i < 60*60 && h.g.Scene().Name == SceneRun; i++ {
		h.g.Tick(engine.FixedDT, engine.Input{Right: true})
	}
	h.frames(2, engine.Input{})
	var p map[string]any
	for _, e := range h.h.Events {
		if e.Name == "run" {
			_ = json.Unmarshal(e.Payload, &p)
		}
	}
	if p == nil || p["won"] != true || p["progress"] != 1.0 || p["body"] != "mars" {
		t.Fatalf("a won run should report full progress, got %v", p)
	}
}
