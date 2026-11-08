package engine

import "testing"

type box struct{ x, y, w, h float64 }

func (b *box) Bounds(*GameObject, Camera) (float64, float64, float64, float64) {
	return b.x, b.y, b.w, b.h
}
func (b *box) MoveBy(_ *GameObject, _ Camera, dx, dy float64) { b.x += dx; b.y += dy }

type stepCounter struct{ n int }

func (c *stepCounter) Update(*GameObject, float64) { c.n++ }

func editorGame(t *testing.T) (*Game, *stepCounter, *box) {
	g := NewGame(960, 540, nil)
	cnt, bx := &stepCounter{}, &box{300, 200, 80, 40}
	g.RegisterScene(SceneMeta{Name: "t", Title: "t", Type: TypePlay, Layout: Layout{Name: "full", Regions: map[string]Rect{"main": {0, 0, 100, 100}}},
		Fixture: func(*Game) any { return nil }}, func(g *Game, _ any) *Scene {
		s := &Scene{}
		s.Cam = Camera{Zoom: 10, Pos: V(0, 0)}
		a := s.Spawn("ticker")
		a.Add(cnt)
		b := s.Spawn("panel")
		b.Layer = 5
		b.Add(bx)
		w := s.Spawn("worldthing")
		w.Pos = V(5, 3)
		return s
	})
	g.Load("t", nil)
	g.Tick(FixedDT, Input{})
	return g, cnt, bx
}

func click(g *Game, p Vec2) {
	g.Tick(FixedDT, Input{Pointer: p, Down: true})
	g.Tick(FixedDT, Input{Pointer: p})
}

func TestEditorPauseStepAndPlay(t *testing.T) {
	g, cnt, _ := editorGame(t)
	g.Tick(FixedDT, Input{})
	run := cnt.n
	g.Tick(FixedDT, Input{ToggleEdit: true})
	if !g.Ed.On || !g.Ed.Paused {
		t.Fatal("editor should open paused")
	}
	for i := 0; i < 10; i++ {
		g.Tick(FixedDT, Input{})
	}
	if cnt.n != run {
		t.Fatalf("scene advanced while paused: %d -> %d", run, cnt.n)
	}
	g.Tick(FixedDT, Input{Step: true})
	if cnt.n != run+1 {
		t.Fatalf("step should advance exactly one fixed step, got %d", cnt.n-run)
	}
	g.Tick(FixedDT, Input{Pause: true}) // play
	g.Tick(FixedDT, Input{})
	g.Tick(FixedDT, Input{})
	if cnt.n <= run+1 {
		t.Fatal("should be running again")
	}
	g.Tick(FixedDT, Input{ToggleEdit: true})
	if g.Ed.On || g.Ed.Paused {
		t.Fatal("closing the editor resumes play")
	}
}

func TestEditorPickDragHideDelete(t *testing.T) {
	g, _, bx := editorGame(t)
	g.SetEditor(true)
	click(g, V(320, 220))
	if g.Ed.Selected == nil || g.Ed.Selected.Name != "panel" {
		t.Fatalf("should select panel, got %+v", g.Ed.Selected)
	}
	// drag by (+40,+20)
	g.Tick(FixedDT, Input{Pointer: V(320, 220), Down: true})
	g.Tick(FixedDT, Input{Pointer: V(360, 240), Down: true})
	g.Tick(FixedDT, Input{Pointer: V(360, 240)})
	if bx.x != 340 || bx.y != 220 {
		t.Fatalf("drag moved box to %v,%v", bx.x, bx.y)
	}
	// world object: picked at its projected position and moved by world units
	p := g.scene.Cam.ToScreen(V(5, 3))
	click(g, p)
	w := g.Ed.Selected
	if w == nil || w.Name != "worldthing" {
		t.Fatalf("should select world object, got %+v", w)
	}
	g.Tick(FixedDT, Input{Pointer: p, Down: true})
	g.Tick(FixedDT, Input{Pointer: V(p.X+20, p.Y-10), Down: true})
	g.Tick(FixedDT, Input{Pointer: V(p.X+20, p.Y-10)})
	if w.Pos.X != 7 || w.Pos.Y != 4 {
		t.Fatalf("world drag: %v", w.Pos)
	}
	g.Do("Hide")
	if w.Active {
		t.Fatal("hide should deactivate")
	}
	g.Do("Delete")
	g.Tick(FixedDT, Input{Step: true})
	if g.scene.Find("worldthing") != nil {
		t.Fatal("delete should remove the object")
	}
}

func TestEditorHierarchySelectAndInspect(t *testing.T) {
	g, _, _ := editorGame(t)
	g.SetEditor(true)
	h := g.Hierarchy()
	if len(h) != 3 || h[len(h)-1].Name != "panel" {
		t.Fatalf("hierarchy back to front: %v", len(h))
	}
	lr := g.listRect()
	click(g, V(20, lr.y+24+edRowH/2)) // first row
	if g.Ed.Selected != h[0] {
		t.Fatalf("list tap should select the first row, got %+v", g.Ed.Selected)
	}
	rows := g.Inspect()
	if rows[0][1] != h[0].Name {
		t.Fatalf("inspector name: %v", rows[0])
	}
}

func TestEditorBlocksSceneInput(t *testing.T) {
	g, _, _ := editorGame(t)
	var pressed bool
	g.scene.OnUpdate = func(float64) { pressed = pressed || g.In.Pressed || g.In.Down }
	g.SetEditor(true)
	g.Do("Pause")
	g.Do("Pause") // playing
	click(g, V(500, 400))
	if pressed {
		t.Fatal("scene must not see clicks while editing")
	}
	g.Draw(NewRecorder()) // overlay draws
}

func TestEditorToolbarButtons(t *testing.T) {
	g, cnt, _ := editorGame(t)
	g.SetEditor(true)
	n := cnt.n
	click(g, V(8+88+10, 10)) // Step
	if cnt.n != n+1 {
		t.Fatalf("Step button: %d", cnt.n-n)
	}
	click(g, V(8+4*88+10, 10)) // Close
	if g.Ed.On {
		t.Fatal("Close button should leave the editor")
	}
}
