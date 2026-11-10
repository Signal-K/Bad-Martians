package engine

import "testing"

type counter struct{ starts, updates int }

func (c *counter) Start(*GameObject)           { c.starts++ }
func (c *counter) Update(*GameObject, float64) { c.updates++ }

func testMeta(name string) SceneMeta {
	return SceneMeta{Name: name, Type: TypeDebug,
		Layout:  Layout{Name: "full", Regions: map[string]Rect{"main": {0, 0, 100, 100}}},
		Fixture: func(*Game) any { return nil }}
}

func newTestGame() (*Game, *MemHost) {
	h := NewMemHost()
	g := NewGame(100, 100, h)
	g.RegisterScene(testMeta("a"), func(g *Game, arg any) *Scene {
		s := &Scene{}
		o := s.Spawn("o")
		o.Add(&counter{})
		return s
	})
	g.RegisterScene(testMeta("b"), func(g *Game, arg any) *Scene { return &Scene{} })
	return g, h
}

func TestFixedStepAndLifecycle(t *testing.T) {
	g, _ := newTestGame()
	g.Load("a", nil)
	g.Tick(0, Input{})
	for i := 0; i < 10; i++ {
		g.Tick(FixedDT*2.5, Input{}) // 25 fixed steps total, however frames fall
	}
	c, _ := Get[*counter](g.Scene().Find("o"))
	if c.starts != 1 || c.updates < 24 || c.updates > 25 { // float accumulator may hold the last step back
		t.Fatalf("starts=%d updates=%d", c.starts, c.updates)
	}
}

func TestPressEdgeSurvivesFramesWithoutSteps(t *testing.T) {
	g, _ := newTestGame()
	g.Load("a", nil)
	var seen int
	g.Tick(0, Input{})
	g.Scene().OnUpdate = func(float64) {
		if g.In.Pressed {
			seen++
		}
	}
	g.Tick(FixedDT/4, Input{Down: true}) // frame too short for a step
	g.Tick(FixedDT/4, Input{Down: true})
	g.Tick(FixedDT, Input{Down: true})
	g.Tick(FixedDT, Input{Down: true})
	if seen != 1 {
		t.Fatalf("press delivered %d times, want exactly 1", seen)
	}
}

func TestTypedTextReachesExactlyOneStep(t *testing.T) {
	g, _ := newTestGame()
	g.Load("a", nil)
	g.Tick(0, Input{})
	got := ""
	enters := 0
	g.Scene().OnUpdate = func(float64) {
		if g.In.Replacing {
			got = g.In.Replace
		}
		got += string(g.In.Typed)
		if g.In.Enter {
			enters++
		}
	}
	g.Tick(FixedDT/4, Input{Typed: []rune("Ro")}) // no step this frame
	g.Tick(FixedDT/4, Input{Typed: []rune("ver")})
	g.Tick(FixedDT*3, Input{Enter: true}) // several steps in one frame
	if got != "Rover" || enters != 1 {
		t.Fatalf("got %q enters=%d, want \"Rover\" once", got, enters)
	}
	g.Tick(FixedDT, Input{Replace: "Mule", Replacing: true, Typed: []rune("!")})
	if got != "Mule!" {
		t.Fatalf("replace then type: got %q, want \"Mule!\"", got)
	}
}

func TestSceneSwitchAndDestroy(t *testing.T) {
	g, h := newTestGame()
	g.Load("a", nil)
	g.Tick(FixedDT, Input{})
	g.Scene().Find("o").Destroy()
	g.Tick(FixedDT, Input{})
	if g.Scene().Find("o") != nil {
		t.Fatal("destroyed object should be gone")
	}
	g.Load("b", nil)
	g.Tick(FixedDT, Input{})
	if g.Scene().Name != "b" {
		t.Fatal("scene should have switched")
	}
	if len(h.Events) != 2 || h.Events[1].Name != "scene" {
		t.Fatalf("scene events: %+v", h.Events)
	}
}

func TestCameraRoundTrip(t *testing.T) {
	c := Camera{Pos: V(5, 2), Zoom: 40, Screen: V(960, 540)}
	p := V(7.5, 1.25)
	q := c.ToWorld(c.ToScreen(p))
	if q.Sub(p).Len() > 1e-9 {
		t.Fatalf("round trip %v -> %v", p, q)
	}
}

func TestRegisterSceneEnforcesContract(t *testing.T) {
	bad := map[string]SceneMeta{
		"no type":    {Name: "x", Layout: testMeta("x").Layout, Fixture: testMeta("x").Fixture},
		"no layout":  {Name: "x", Type: TypeMenu, Fixture: testMeta("x").Fixture},
		"no fixture": {Name: "x", Type: TypeMenu, Layout: testMeta("x").Layout},
		"offscreen":  {Name: "x", Type: TypeMenu, Fixture: testMeta("x").Fixture, Layout: Layout{Name: "l", Regions: map[string]Rect{"main": {0, 0, 500, 5}}}},
	}
	for name, m := range bad {
		func() {
			defer func() {
				if recover() == nil {
					t.Errorf("%s: expected RegisterScene to panic", name)
				}
			}()
			NewGame(100, 100, nil).RegisterScene(m, func(*Game, any) *Scene { return &Scene{} })
		}()
	}
}

func TestOpenStandaloneUsesFixture(t *testing.T) {
	g, _ := newTestGame()
	if !g.OpenStandalone("a") || g.OpenStandalone("nope") {
		t.Fatal("OpenStandalone result wrong")
	}
	g.Tick(0, Input{})
	if g.Scene().Meta.Type != TypeDebug {
		t.Fatal("scene did not receive its meta")
	}
}
