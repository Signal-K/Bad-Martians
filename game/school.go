package game

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/signal-k/haphazard-martians/engine"
)

// Lesson is one Shape school exercise: the bay opens with a rover that is
// wrong in one way, the status line says what is wrong, and a short course
// only a fixed rover can pass. It teaches arrangements, not parts: a player
// who just piles blocks up learns why the pile does not drive.
type Lesson struct {
	Title      string // short name, e.g. "Wheels down"
	Teach      string // what is wrong and what to do, in the bay's status line
	Pass       string // what the fix taught, on the debrief
	Fail       string // a nudge when the run did not pass
	Broken     Design // the rover the bay opens with
	Fixed      Design // one rover that passes; tests use it
	Level      LevelDef
	AllSamples bool // passing needs every sample, not just the finish
}

// schoolLevel is a short, gentle course for a lesson.
func schoolLevel(id int, name string, credits int, length float64) LevelDef {
	d := LevelDef{ID: id, Name: name, Blurb: "Shape school.", Biome: "plains", Length: length, Seed: 11,
		Credits: credits, Par: 20, Hill: 0.3, Traction: 1}
	d.Mix[Soil], d.Mix[Bedrock] = 4, 3
	return d
}

func design(cells ...Cell) Design { return Design{Cells: cells} }

// longFrame is a wheel, n frames and a motor in a row, with the mast on a
// two-frame tower in the middle. braced adds a second row of frames above the
// first, which turns every square into two triangles.
func longFrame(n int, braced bool) Design {
	d := Design{}
	d.Set(0, 0, KWheel)
	for x := 1; x <= n; x++ {
		d.Set(x, 0, KFrame)
		if braced {
			d.Set(x, 1, KFrame)
		}
	}
	d.Set(n+1, 0, KMotor)
	d.Set(3, 1, KFrame)
	d.Set(3, 2, KFrame)
	d.Set(3, 3, KMast)
	return d
}

var Lessons = []Lesson{
	{
		Title:  "Wheels down",
		Teach:  "The motor wheel is up in the frame. Move it to the bottom.",
		Pass:   "A motor wheel only drives when it touches the ground.",
		Fail:   "Is the motor wheel on the bottom row? Test drop to check.",
		Broken: design(Cell{0, 0, KWheel}, Cell{1, 0, KFrame}, Cell{2, 0, KWheel}, Cell{0, 1, KFrame}, Cell{1, 1, KMotor}, Cell{1, 2, KMast}),
		Fixed:  design(Cell{0, 0, KWheel}, Cell{1, 0, KFrame}, Cell{2, 0, KMotor}, Cell{0, 1, KFrame}, Cell{1, 1, KFrame}, Cell{1, 2, KMast}),
		Level:  schoolLevel(801, "Wheels down", 12, 40),
	},
	{
		Title:  "Brace it",
		Teach:  "One long row of frame bends. Add a row on top to brace it.",
		Pass:   "Two rows make triangles, and triangles do not fold.",
		Fail:   "Does the test drop still say the frame bends?",
		Broken: longFrame(6, false),
		Fixed:  longFrame(6, true),
		Level:  schoolLevel(802, "Brace it", 22, 40),
	},
	{
		Title:      "Reach high",
		Teach:      "The middle sample floats high. Raise the mast to reach it.",
		Pass:       "A taller rover reaches the high samples.",
		Fail:       "Collect all three samples. Stand the mast on a frame.",
		Broken:     StarterDesign(),
		Fixed:      design(Cell{0, 0, KWheel}, Cell{1, 0, KFrame}, Cell{2, 0, KFrame}, Cell{3, 0, KMotor}, Cell{1, 1, KFrame}, Cell{2, 1, KFrame}, Cell{2, 2, KMast}),
		Level:      schoolLevel(803, "Reach high", 14, 40),
		AllSamples: true,
	},
}

// LessonSession opens lesson i (0-based) with its broken rover in the bay.
func LessonSession(i int) *Session {
	l := Lessons[i]
	s := NewSession(BuildLevel(l.Level))
	s.Design = Design{Cells: append([]Cell(nil), l.Broken.Cells...)}
	s.Lesson = i + 1
	s.Name = "Trainee"
	return s
}

// LessonOf is the lesson a session is running, or nil outside Shape school.
func LessonOf(s *Session) *Lesson {
	if s == nil || s.Lesson < 1 || s.Lesson > len(Lessons) {
		return nil
	}
	return &Lessons[s.Lesson-1]
}

// Passed reports whether a finished run passes the lesson.
func (l *Lesson) Passed(o Outcome, samples, total int) bool {
	return o == Won && (!l.AllSamples || samples == total)
}

const schoolKey = "hm.school"

// SchoolDone is how many lessons the player has passed, in order.
func SchoolDone(h engine.Host) int {
	n, _ := strconv.Atoi(string(h.Load(schoolKey)))
	return min(max(n, 0), len(Lessons))
}

// PassLesson records lesson i (1-based) as passed.
func PassLesson(h engine.Host, i int) {
	if i > SchoolDone(h) {
		h.Save(schoolKey, []byte(strconv.Itoa(i)))
	}
}

// NextLesson is the 0-based lesson the menu opens: the first not yet passed,
// or the first again once all are done.
func NextLesson(h engine.Host) int {
	if n := SchoolDone(h); n < len(Lessons) {
		return n
	}
	return 0
}

// lessonResult is the debrief after a Shape school run: passed or not, what the
// lesson taught, and the rover that ran. Lesson runs leave mission progress and
// the survey's run records alone.
func lessonResult(g *engine.Game, ra *ResultArg) *engine.Scene {
	ses := ra.Session
	l := LessonOf(ses)
	passed := l.Passed(ra.Outcome, ra.Samples, ra.Total)
	if !ra.Recorded {
		ra.Recorded = true
		if passed {
			PassLesson(g.Host, ses.Lesson)
		}
		p, _ := json.Marshal(map[string]any{"lesson": ses.Lesson, "title": l.Title, "passed": passed,
			"outcome": ra.Outcome.Message(), "samples": ra.Samples, "cost": ses.Design.Cost()})
		g.Host.Emit("lesson", p)
	}
	last := ses.Lesson == len(Lessons)
	s := newScene()
	addBackdrop(s)
	AddDraw(s, 2, func(_ engine.Camera, r engine.Renderer) {
		head, col, msg := "Lesson passed", colGood, l.Pass
		if !passed {
			head, col, msg = "Not yet", colWarn, l.Fail
		}
		if passed && last {
			head = "Shape school complete"
		}
		r.Text(head, 28, 16, sizeTitle, col, engine.AlignLeft)
		r.Text(fmt.Sprintf("Shape school %d/%d · %s", ses.Lesson, len(Lessons), l.Title), 28, 54, sizeBody, colMuted, engine.AlignLeft)

		drawPanel(r, 28, 124, 904, 310, "Your rover")
		out := ra.Outcome.Message()
		if ra.Outcome == Scrapped { // the lesson rover is a trainee: nothing is lost
			out = "It got stuck, so it was scrapped."
		}
		if ra.Outcome == Won && l.AllSamples {
			out = fmt.Sprintf("%s Samples %d/%d.", out, ra.Samples, ra.Total)
		}
		r.Text(out, 52, 150, sizeBody, colInk, engine.AlignLeft)
		r.Text(msg, 52, 182, sizeBody, col, engine.AlignLeft)
		if passed && last {
			r.Text("Use these shapes on the missions: wheels down, braced, mast up high.", 52, 214, sizeSmall, colMuted, engine.AlignLeft)
		}
		// the rover that ran, drawn as it was built
		const z = 36.0
		w, h := 0, 0
		for _, c := range ses.Design.Cells {
			w, h = max(w, c.X+1), max(h, c.Y+1)
		}
		ox, oy := ScreenW/2-float64(w)*z/2, 410.0
		at := func(x, y int) engine.Vec2 { return engine.V(ox+(float64(x)+0.5)*z, oy-(float64(y)+0.5)*z) }
		for _, c := range ses.Design.Cells {
			for _, o := range [][2]int{{1, 0}, {0, 1}, {1, 1}, {-1, 1}} {
				if _, ok := ses.Design.At(c.X+o[0], c.Y+o[1]); ok {
					drawBeam(r, at(c.X, c.Y), at(c.X+o[0], c.Y+o[1]), z)
				}
			}
		}
		for _, c := range ses.Design.Cells {
			drawPart(r, c.K, at(c.X, c.Y), z, 0)
		}
	})
	addBack(s, func() { g.Load(SceneMenu, nil) })
	AddButton(s, 100, 470, 220, 60, "Try again", func() { g.Load(SceneBuild, ses) }).Icon = IconRover
	label := "Next lesson"
	if last {
		label = "Missions"
	}
	next := AddButton(s, 730, 470, 206, 60, label, func() {
		if last {
			g.Load(SceneMenu, nil)
			return
		}
		g.Load(SceneBuild, LessonSession(ses.Lesson))
	})
	next.Primary, next.Icon = true, IconRight
	next.Disabled = !passed
	return s
}
