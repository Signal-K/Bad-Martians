package game

import (
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

// drag presses at a, moves to b over a few frames, and releases there.
func (h *harness) drag(a, b engine.Vec2) {
	h.frames(1, engine.Input{Pointer: a, Down: true})
	for i := 1; i <= 4; i++ {
		h.frames(1, engine.Input{Pointer: a.Add(b.Sub(a).Mul(float64(i) / 4)), Down: true})
	}
	h.frames(1, engine.Input{Pointer: b})
	h.frames(1, engine.Input{Pointer: b})
}

func cellCentre(cx, cy int) engine.Vec2 {
	return engine.V(bGX+float64(cx)*bCell+bCell/2, bGY+float64(GridH-1-cy)*bCell+bCell/2)
}

func TestTutorialFirstBuildAndDrive(t *testing.T) {
	h := &harness{t: t, h: engine.NewMemHost()}
	h.g = engine.NewGame(ScreenW, ScreenH, h.h)
	Setup(h.g)
	ses := NewSessionFor(h.h, GetLevel(1)) // what the Mission 1 card opens
	h.g.Load(SceneBuild, ses)
	h.frames(2, engine.Input{})
	if ses.Tut == nil || len(ses.Design.Cells) != 0 {
		t.Fatalf("first visit should start the tutorial on an empty grid")
	}
	if st := ses.Tut.BuildStep(ses.Design); st != tutBeam {
		t.Fatalf("step %d, want beam", st)
	}
	// drag the Beam from the shapes sidebar onto the glowing cells
	beam := h.button("Beam")
	h.drag(engine.V(beam.X+beam.W/2, beam.Y+beam.H/2), cellCentre(4, 1))
	for x := 3; x <= 5; x++ {
		if k, _ := ses.Design.At(x, 1); k != KFrame {
			t.Fatalf("beam should fill 3..5 on row 1, got %v", ses.Design.Cells)
		}
	}
	// wheel, motor wheel and mast where the tutorial points
	for _, st := range []struct {
		k    Kind
		name string
		step int
	}{{KWheel, "Free wheel", tutWheel}, {KMotor, "Motor wheel", tutMotor}, {KMast, "Navcam mast", tutMast}} {
		if got := ses.Tut.BuildStep(ses.Design); got != st.step {
			t.Fatalf("step %d, want %d", got, st.step)
		}
		hint := ses.Tut.hintCells(ses.Design, st.step)
		if len(hint) == 0 {
			t.Fatalf("no hint cells for %s", st.name)
		}
		h.clickButton(st.name)
		c := cellCentre(hint[0][0], hint[0][1])
		h.click(c.X, c.Y)
		if k, _ := ses.Design.At(hint[0][0], hint[0][1]); k != st.k {
			t.Fatalf("%s not placed at hint %v", st.name, hint[0])
		}
	}
	if errs := ses.Design.Validate(ses.Budget()); len(errs) > 0 {
		t.Fatalf("tutorial rover should be launchable: %v", errs)
	}
	h.clickButton("Test")
	if ses.Tut.BuildStep(ses.Design) != tutGo {
		t.Fatalf("test drop should advance to GO")
	}
	h.clickButton("Test") // back to the grid
	h.clickButton("GO")
	if h.g.Scene().Name != SceneLaunch {
		t.Fatalf("expected launch, got %s", h.g.Scene().Name)
	}
	h.clickButton("Send it out")
	if h.g.Scene().Name != SceneRun {
		t.Fatalf("expected run, got %s", h.g.Scene().Name)
	}
	h.frames(int(tutGasSeconds*60)+5, engine.Input{Right: true})
	if !ses.Tut.Done || !TutorialDone(h.h) {
		t.Fatalf("holding gas should finish the tutorial")
	}
	// the rover the tutorial teaches must be able to finish the mission
	run := NewRun(ses.Level, ses.Design)
	for run.Outcome == Running {
		run.Step(engine.FixedDT, 1)
	}
	if run.Outcome != Won {
		t.Fatalf("tutorial rover: %s", run.Outcome.Message())
	}
	if NewSessionFor(h.h, GetLevel(1)).Tut != nil {
		t.Fatalf("the tutorial must not come back once done")
	}
}

func TestTutorialSkip(t *testing.T) {
	h := &harness{t: t, h: engine.NewMemHost()}
	h.g = engine.NewGame(ScreenW, ScreenH, h.h)
	Setup(h.g)
	ses := NewSessionFor(h.h, GetLevel(1))
	h.g.Load(SceneBuild, ses)
	h.frames(2, engine.Input{})
	h.clickButton("Skip")
	if !TutorialDone(h.h) || len(ses.Design.Cells) != 0 {
		t.Fatalf("skip should end the tutorial without painting the grid: %v", ses.Design.Cells)
	}
}

func TestBlocksAnchorOnGrid(t *testing.T) {
	for _, b := range Blocks {
		w, hgt := b.Size()
		for _, p := range [][2]int{{-3, -3}, {GridW + 2, GridH + 2}, {5, 2}} {
			x, y := b.Anchor(p[0], p[1])
			if x < 0 || y < 0 || x+w > GridW || y+hgt > GridH {
				t.Fatalf("%s anchored off the grid at %d,%d", b.Name, x, y)
			}
		}
		var d Design
		x, y := b.Anchor(5, 0)
		b.Stamp(&d, x, y)
		if len(d.Cells) != len(b.Cells) || !d.connected() {
			t.Fatalf("%s should stamp as one connected piece", b.Name)
		}
	}
}
