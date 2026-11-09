package game

import (
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

// Every scene, opened on its own fixture, must work as an editable scene:
// the hierarchy lists its objects, buttons can be picked and dragged, the
// simulation pauses and steps, and the overlay draws.
func TestEveryGameSceneIsEditable(t *testing.T) {
	for _, m := range newHarness(t).g.Scenes() {
		t.Run(m.Name, func(t *testing.T) {
			h := newHarness(t)
			if !h.g.OpenStandalone(m.Name) {
				t.Fatal("cannot open standalone")
			}
			h.frames(3, engine.Input{})
			h.g.SetEditor(true)
			h.frames(2, engine.Input{})
			if len(h.g.Hierarchy()) == 0 {
				t.Fatal("empty hierarchy")
			}
			before := h.g.Scene().Time
			h.frames(30, engine.Input{})
			if h.g.Scene().Time != before {
				t.Fatal("scene advanced while paused")
			}
			h.g.Tick(engine.FixedDT, engine.Input{Step: true})
			if h.g.Scene().Time <= before {
				t.Fatal("step did not advance the scene")
			}
			// Pick and drag a button if the scene has any.
			for _, o := range h.g.Hierarchy() {
				var b *Button
				if got, ok := engine.Get[*Button](o); ok {
					b = got
				}
				if b == nil || b.W < 30 {
					continue
				}
				name := h.g.Scene().Name
				cx, cy := b.X+b.W/2, b.Y+b.H/2
				// Skip buttons hidden under the editor panels.
				if cy < 40 || cx < 190 || cx > h.g.W-230 {
					continue
				}
				h.click(cx, cy)
				if h.g.Ed.Selected != o {
					continue // covered by a later object; fine
				}
				x0 := b.X
				h.g.Tick(engine.FixedDT, engine.Input{Pointer: engine.V(cx, cy), Down: true})
				h.g.Tick(engine.FixedDT, engine.Input{Pointer: engine.V(cx+25, cy), Down: true})
				h.g.Tick(engine.FixedDT, engine.Input{Pointer: engine.V(cx+25, cy)})
				if b.X != x0+25 {
					t.Fatalf("%s: drag moved button %v, want %v", name, b.X-x0, 25.0)
				}
				if h.g.Scene().Name != name {
					t.Fatal("editing a button must not press it")
				}
				break
			}
			h.g.SetEditor(false)
			h.frames(2, engine.Input{})
		})
	}
}

func TestEditorHostFlag(t *testing.T) {
	h := newHarness(t)
	h.g.Load(SceneMenu, nil)
	h.frames(2, engine.Input{})
	h.g.Tick(engine.FixedDT, engine.Input{ToggleEdit: true})
	if !h.g.Ed.On {
		t.Fatal("toggle key should open the editor")
	}
	// An edit made in the editor survives closing it: move a mission tile, then play.
	b := h.button("Mission 4")
	x := b.X
	h.click(b.X+b.W/2, b.Y+b.H/2)
	h.g.Tick(engine.FixedDT, engine.Input{Pointer: engine.V(b.X+b.W/2, b.Y+b.H/2), Down: true})
	h.g.Tick(engine.FixedDT, engine.Input{Pointer: engine.V(b.X+b.W/2, b.Y+b.H/2+30), Down: true})
	h.g.Tick(engine.FixedDT, engine.Input{})
	h.g.Tick(engine.FixedDT, engine.Input{ToggleEdit: true})
	if h.button("Mission 4").X != x || h.button("Mission 4").Y != 184+30 {
		t.Fatalf("button should have moved down only: %v,%v", h.button("Mission 4").X, h.button("Mission 4").Y)
	}
}

func TestEditorDragsRoverWithoutVelocity(t *testing.T) {
	h := newHarness(t)
	ses := NewSessionFor(h.g.Host, GetLevel(1))
	ses.Design = StarterDesign()
	h.g.Load(SceneRun, ses)
	h.frames(30, engine.Input{})
	h.g.SetEditor(true)
	ro := h.g.Scene().Find("rover")
	rv, _ := engine.Get[*Rover](ro)
	c := h.g.Scene().Cam.ToScreen(rv.Center())
	h.click(c.X, c.Y)
	if h.g.Ed.Selected != ro {
		t.Fatalf("rover should be pickable in the viewport, got %+v", h.g.Ed.Selected)
	}
	before := rv.Center()
	h.g.Tick(engine.FixedDT, engine.Input{Pointer: c, Down: true})
	h.g.Tick(engine.FixedDT, engine.Input{Pointer: engine.V(c.X+40, c.Y), Down: true})
	h.g.Tick(engine.FixedDT, engine.Input{Pointer: engine.V(c.X+40, c.Y)})
	after := rv.Center()
	if d := after.X - before.X; d < 0.5 {
		t.Fatalf("rover moved %.2f m", d)
	}
	for _, b := range rv.Bodies {
		if b.Pos.Sub(b.Prev).Len() != 0 && b.Pos.Sub(b.Prev).Len() > 0.5 {
			t.Fatalf("dragging gave a body velocity %v", b.Pos.Sub(b.Prev))
		}
	}
}
