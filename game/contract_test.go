package game

import (
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

func newGame() (*engine.Game, *engine.MemHost) {
	h := engine.NewMemHost()
	g := engine.NewGame(ScreenW, ScreenH, h)
	Setup(g)
	return g, h
}

// runStandalone boots one scene on its own and runs it for n frames, returning
// the draw digest. With nothing pressed, every scene must stay up.
func runStandalone(t *testing.T, name string, n int) (*engine.Game, *engine.Recorder, uint64) {
	g, _ := newGame()
	if !g.OpenStandalone(name) {
		t.Fatalf("%s: cannot open standalone", name)
	}
	rec := engine.NewRecorder()
	sum := engine.RunFrames(g, rec, n, nil)
	return g, rec, sum
}

func TestEverySceneHonoursTheContract(t *testing.T) {
	g, _ := newGame()
	metas := g.Scenes()
	if len(metas) < 6 {
		t.Fatalf("expected the full scene set, got %d", len(metas))
	}
	for _, m := range metas {
		m := m
		t.Run(m.Name, func(t *testing.T) {
			if m.Type == "" || m.Layout.Name == "" || m.Fixture == nil || m.Title == "" {
				t.Fatalf("incomplete meta: %+v", m)
			}
			// 1. boots alone, runs 10 s, draws something every frame
			g1, rec1, sum1 := runStandalone(t, m.Name, 600)
			if rec1.Ops == 0 || rec1.Texts == 0 {
				t.Fatal("scene drew nothing")
			}
			if g1.Scene().Meta.Name != m.Name {
				t.Fatalf("scene left itself on its own: now %s", g1.Scene().Name)
			}
			// 2. deterministic: a second run draws exactly the same pixels
			if _, _, sum2 := runStandalone(t, m.Name, 600); sum1 != sum2 {
				t.Fatalf("not deterministic: %x vs %x", sum1, sum2)
			}
			// 3. every button is on screen, inside the layout, and big enough to tap
			screen := engine.Rect{X: 0, Y: 0, W: ScreenW, H: ScreenH}
			for _, o := range g1.Scene().Objects {
				b, ok := engine.Get[*Button](o)
				if !ok {
					continue
				}
				br := engine.Rect{X: b.X, Y: b.Y, W: b.W, H: b.H}
				if !screen.Contains(br) {
					t.Errorf("button %q is off screen", b.Label)
				}
				in := false
				for _, r := range m.Layout.Regions {
					in = in || r.Contains(br)
				}
				if !in {
					t.Errorf("button %q (%+v) is outside every region of layout %q", b.Label, br, m.Layout.Name)
				}
				if b.W < touchMin || b.H < touchMin {
					t.Errorf("button %q is %.0fx%.0f, below the %.0f px touch minimum", b.Label, b.W, b.H, touchMin)
				}
			}
		})
	}
}

func TestSceneTypesAndLayoutsAreDistinctAndUsed(t *testing.T) {
	g, _ := newGame()
	want := map[string]engine.SceneType{
		SceneMenu: engine.TypeMenu, SceneBuild: engine.TypeBuild, SceneRun: engine.TypePlay,
		SceneResult: engine.TypeResults, SceneReview: engine.TypeActivity, SceneStyle: engine.TypeDebug,
	}
	for _, m := range g.Scenes() {
		if want[m.Name] != m.Type {
			t.Errorf("%s: type %q, want %q", m.Name, m.Type, want[m.Name])
		}
	}
}

func TestScoreRewardsSpeed(t *testing.T) {
	l := GetLevel(1)
	fast := ScoreFor(l, true, 2, l.Par*0.5)
	atPar := ScoreFor(l, true, 2, l.Par)
	slow := ScoreFor(l, true, 2, l.Par*3)
	if !(fast.Total() > atPar.Total() && atPar.Total() > slow.Total()) {
		t.Fatalf("faster must score higher: %d %d %d", fast.Total(), atPar.Total(), slow.Total())
	}
	if fast.Time != scoreTimeMax || slow.Time != 0 {
		t.Fatalf("time bonus should span 0..%d: %d %d", scoreTimeMax, fast.Time, slow.Time)
	}
	if ScoreFor(l, false, 2, 1).Total() != 0 {
		t.Fatal("a failed run scores nothing")
	}
	if ScoreFor(l, true, 2, l.Par).Total() <= ScoreFor(l, true, 0, l.Par).Total() {
		t.Fatal("samples must add to the score")
	}
}

func TestEveryIconDraws(t *testing.T) {
	for _, ic := range AllIcons {
		rec := engine.NewRecorder()
		drawIcon(rec, ic, 50, 50, 40, colInk)
		if rec.Ops == 0 {
			t.Errorf("icon %q draws nothing", ic)
		}
	}
}
