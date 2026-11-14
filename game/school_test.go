package game

import (
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

// driveLesson runs d on a level with the throttle held, until it ends or is stuck.
func driveLesson(l *Level, d Design) *Run {
	r := NewRun(l, d)
	for i := 0; i < 60*60 && r.Outcome == Running && !r.IsStuck(); i++ {
		r.Step(1.0/60, 1)
	}
	return r
}

// Every lesson must fail with the rover it opens with and pass once fixed,
// within its budget: otherwise it teaches nothing.
func TestEachLessonFailsBrokenAndPassesFixed(t *testing.T) {
	for i := range Lessons {
		l := &Lessons[i]
		ses := LessonSession(i)
		for _, c := range []struct {
			name string
			d    Design
			want bool
		}{{"broken", l.Broken, false}, {"fixed", l.Fixed, true}} {
			if errs := c.d.Validate(ses.Budget()); len(errs) > 0 {
				t.Fatalf("%s %s: %v (cost %d, budget %d)", l.Title, c.name, errs, c.d.Cost(), ses.Budget())
			}
			r := driveLesson(BuildLevel(l.Level), c.d)
			got := l.Passed(r.Outcome, r.SamplesGot(), len(r.Got))
			t.Logf("%s %s: outcome %v, %.0f%%, samples %d/%d, %.1fs", l.Title, c.name, r.Outcome, r.Progress()*100, r.SamplesGot(), len(r.Got), r.Time)
			if got != c.want {
				t.Errorf("%s %s: passed=%v, want %v", l.Title, c.name, got, c.want)
			}
		}
	}
}

func TestSchoolProgress(t *testing.T) {
	h := newHarness(t).g.Host
	if SchoolDone(h) != 0 || NextLesson(h) != 0 {
		t.Fatal("a new player starts at lesson 1")
	}
	PassLesson(h, 1)
	PassLesson(h, 1)
	if SchoolDone(h) != 1 || NextLesson(h) != 1 {
		t.Fatalf("after lesson 1: done %d next %d", SchoolDone(h), NextLesson(h))
	}
	PassLesson(h, len(Lessons))
	if NextLesson(h) != 0 {
		t.Fatal("once all are done the school starts over")
	}
}

// From the menu: the first lesson opens with its broken rover, GO skips the
// naming, a fixed rover passes, and Next lesson opens lesson 2. Lesson runs
// leave mission progress and the survey's run records alone.
func TestShapeSchoolFlow(t *testing.T) {
	h := newHarness(t)
	h.g.Load(SceneMenu, nil)
	h.frames(2, engine.Input{})
	h.clickButton("Shape school")
	if h.g.Scene().Name != SceneBuild {
		t.Fatalf("expected the bay, got %s", h.g.Scene().Name)
	}
	ses := LessonSession(0)
	if ses.Design.Cost() != Lessons[0].Broken.Cost() {
		t.Fatal("the bay opens with the lesson's broken rover")
	}
	ses.Design = Lessons[0].Fixed
	h.g.Load(SceneBuild, ses)
	h.frames(2, engine.Input{})
	h.clickButton("GO")
	if h.g.Scene().Name != SceneRun {
		t.Fatalf("lessons drive straight away, got %s", h.g.Scene().Name)
	}
	for i := 0; i < 60*60 && h.g.Scene().Name == SceneRun; i++ {
		h.g.Tick(engine.FixedDT, engine.Input{Right: true})
	}
	h.frames(2, engine.Input{})
	if h.g.Scene().Name != SceneResult || h.button("Next lesson").Disabled {
		t.Fatalf("a fixed rover should pass: scene %s", h.g.Scene().Name)
	}
	if SchoolDone(h.h) != 1 {
		t.Fatalf("lesson 1 should be recorded, got %d", SchoolDone(h.h))
	}
	for _, e := range h.h.Events {
		if e.Name == "run" {
			t.Fatal("a lesson run must not be sent as a survey run")
		}
	}
	if len(LoadProgress(h.h).Levels) != 0 {
		t.Fatal("a lesson must not touch mission progress")
	}
	h.clickButton("Next lesson")
	if h.g.Scene().Name != SceneBuild {
		t.Fatalf("Next lesson opens the bay, got %s", h.g.Scene().Name)
	}
}
