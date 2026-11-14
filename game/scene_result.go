package game

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/signal-k/haphazard-martians/engine"
)

type ResultArg struct {
	Session  *Session
	Outcome  Outcome
	Stars    int
	Time     float64
	Samples  int
	Total    int
	Score    Score
	Distance float64 // metres of the course covered
	TopSpeed float64 // m/s
	MaxAir   float64 // longest jump, seconds
	Damaged  bool    // a frame or the mast hit the ground during the drive
	Recorded bool    // progress saved and run event sent (set on first show)

	Salvaged Kind   // extra part an undamaged finish gave back
	Name     string // the rover that made this run
}

// sampleResult is the standalone fixture: a won mission with plausible numbers.
func sampleResult(g *engine.Game) *ResultArg {
	l := GetLevel(1)
	ses := NewSessionFor(g.Host, l)
	if len(ses.Design.Cells) == 0 { // bays start empty; the fixture needs a rover
		ses.Design = StarterDesign()
	}
	ses.Name = SuggestName(0)
	return &ResultArg{Salvaged: KMotor, Name: ses.Name, Session: ses, Outcome: Won, Stars: 2, Time: l.Par * 1.2, Samples: 2, Total: len(l.Samples),
		Score: ScoreFor(l, true, 2, l.Par*1.2), Recorded: true,
		Distance: l.GoalX - l.StartX, TopSpeed: 4.6, MaxAir: 0.7}
}

func resultScene(g *engine.Game, arg any) *engine.Scene {
	ra := arg.(*ResultArg)
	ses := ra.Session
	lvl := ses.Level
	s := newScene()

	prog := LoadProgress(g.Host)
	if !ra.Recorded {
		ra.Recorded = true
		if ra.Outcome == Won {
			prog.Record(lvl.ID, ra.Stars, ra.Time, ra.Score.Total())
			prog.Save(g.Host)
		}
		design := make([]map[string]any, 0, len(ses.Design.Cells))
		for _, c := range ses.Design.Cells {
			design = append(design, map[string]any{"x": c.X, "y": c.Y, "k": string(c.K)})
		}
		payload, _ := json.Marshal(map[string]any{
			"level": lvl.ID, "won": ra.Outcome == Won, "outcome": ra.Outcome.Message(), "stars": ra.Stars,
			"time": math.Round(ra.Time*10) / 10, "samples": ra.Samples, "score": ra.Score.Total(),
			"cost": ses.Design.Cost(), "design": design, "rover": ses.Name, "damaged": ra.Damaged,
		})
		g.Host.Emit("run", payload)
		won := ra.Outcome == Won
		ra.Name = ses.Name
		if k, ok := SalvagePart(ses.Design); ok && won && !ra.Damaged {
			ra.Salvaged = k
		}
		if !won { // the rover stays out there; the next one is named after it
			ses.Name = SuccessorName(ses.Name)
		}
		if ses.Exp != nil {
			ses.Exp.Finish(won, ses.Design, ra.Score.Total(), ra.Time, lvl.Par)
			if ra.Salvaged != "" && !ses.Exp.Done {
				ses.Exp.Salvage(ses.Design)
			}
			ses.Exp.Name = ses.Name
			if ses.Exp.Over() && ses.Exp.Score > prog.BestExpedition {
				prog.BestExpedition = ses.Exp.Score
				prog.Save(g.Host)
			}
		}
	}
	exp := ses.Exp
	best := prog.Levels[lvl.ID].Score
	grade := GradeFor(lvl, ra)
	clock := 0.0
	s.OnUpdate = func(dt float64) { clock += dt }
	// tally counts a number up when its row's turn comes in the debrief
	tally := func(row int, v float64) float64 { return v * clamp01((clock-0.2-0.18*float64(row))/0.35) }

	addBackdrop(s)
	AddDraw(s, 2, func(_ engine.Camera, r engine.Renderer) {
		head, col := "Mission complete", colGood
		if ra.Outcome != Won {
			head, col = "Mission failed", colWarn
		}
		if exp != nil && exp.Done {
			head = "Expedition complete"
		} else if exp != nil && exp.Failed {
			head = "Expedition lost"
		}
		r.Text(head, 28, 16, sizeTitle, col, engine.AlignLeft)
		r.Text(ra.Outcome.Message(), 28, 54, sizeBody, colMuted, engine.AlignLeft)
		switch {
		case ra.Salvaged != "" && exp != nil:
			if !exp.Done {
				r.Text(fmt.Sprintf("No damage: salvaged a %s for credits.", partName(ra.Salvaged)), 28, 88, sizeSmall, colGood, engine.AlignLeft)
			}
		case ra.Salvaged != "":
			r.Text(fmt.Sprintf("No damage: salvaged a %s for the next mission.", partName(ra.Salvaged)), 28, 88, sizeSmall, colGood, engine.AlignLeft)
		case ra.Outcome == Won:
			r.Text("The rover took damage, so nothing could be salvaged.", 28, 88, sizeSmall, colMuted, engine.AlignLeft)
		}

		drawPanel(r, 28, 124, 520, 310, "Debrief")
		// the drive in three numbers
		for i, st := range []struct {
			ic Icon
			n  string
			v  string
		}{
			{IconRover, "Distance", fmt.Sprintf("%.0f m", tally(0, ra.Distance))},
			{IconRight, "Top speed", fmt.Sprintf("%.1f m/s", tally(1, ra.TopSpeed))},
			{IconStarIco, "Best jump", fmt.Sprintf("%.1f s", tally(2, ra.MaxAir))},
		} {
			x := 48 + float64(i)*116
			r.Rect(x, 146, 108, 74, colPanel, true)
			drawIcon(r, st.ic, x+18, 164, 18, colMuted)
			r.Text(st.n, x+32, 155, 12, colMuted, engine.AlignLeft)
			r.Text(st.v, x+54, 180, 20, colInk, engine.AlignCenter)
		}
		// the grade, stamped once the tally is done
		gk := clamp01((clock - 1.3) / 0.25)
		r.Rect(404, 146, 124, 74, colPanel, true)
		if gk > 0 {
			gc := grade.Color()
			sz := 52 * (1.6 - 0.6*gk)
			r.Text(grade.Letter, 444, 183-sz/2, sz, gc.Alpha(uint8(255*gk)), engine.AlignCenter)
			drawStars(r, 496, 172, 7, ra.Stars)
			r.Text("grade", 496, 190, 12, colMuted, engine.AlignCenter)
		}
		rows := []struct {
			n string
			v int
		}{{"Finish", ra.Score.Finish}, {fmt.Sprintf("Samples %d/%d", ra.Samples, ra.Total), ra.Score.Samples},
			{fmt.Sprintf("Time %s (par %s)", fmtTime(ra.Time), fmtTime(lvl.Par)), ra.Score.Time}}
		for i, row := range rows {
			y := 236 + float64(i)*32
			r.Text(row.n, 56, y, sizeBody, colInk, engine.AlignLeft)
			r.Text(fmt.Sprintf("+%.0f", tally(3+i, float64(row.v))), 520, y, sizeBody, colInk, engine.AlignRight)
			r.Line(engine.V(56, y+26), engine.V(520, y+26), 1, colMuted.Alpha(90))
		}
		r.Text("Score", 56, 336, 24, colNeon, engine.AlignLeft)
		r.Text(fmt.Sprintf("%.0f", tally(6, float64(ra.Score.Total()))), 520, 336, 24, colNeon, engine.AlignRight)
		rover := ra.roverName()
		parts := fmt.Sprintf("%s cost %d credits", rover, ses.Design.Cost())

		r.Text(fmt.Sprintf("Best %d   ·   %s", best, parts), 56, 380, sizeSmall, colMuted, engine.AlignLeft)

		drawPanel(r, 572, 124, 360, 310, "Rover")
		r.Text(ra.roverName(), 596, 154, 24, colInk, engine.AlignLeft)
		r.Text(fmt.Sprintf("%d parts · %d credits", len(ses.Design.Cells), ses.Design.Cost()),
			596, 194, sizeBody, colMuted, engine.AlignLeft)
		if exp != nil && exp.Done {
			r.Text("Expedition complete!", 596, 240, sizeSmall, colGood, engine.AlignLeft)
		} else if exp != nil && exp.Failed {
			r.Text("Not enough credits to continue.", 596, 240, sizeSmall, colWarn, engine.AlignLeft)
		} else if ra.Outcome == Won {
			r.Text("Ready for the next mission.", 596, 240, sizeSmall, colGood, engine.AlignLeft)
		} else {
			r.Text("Build another rover and try again.", 596, 240, sizeSmall, colMuted, engine.AlignLeft)
		}

		if exp != nil {
			msg := fmt.Sprintf("Budget: %d credits · expedition score %d", exp.Credits, exp.Score)
			if exp.Lost > 0 {
				msg = fmt.Sprintf("Lost %d credits · %d remaining", exp.Lost, exp.Credits)
			}
			if exp.ParReward > 0 {
				msg = fmt.Sprintf("Under par: +%d credits · %d available", exp.ParReward, exp.Credits)
			}
			r.Text(msg, 56, 408, sizeSmall, colNeon, engine.AlignLeft)
		}
	})
	addBack(s, func() { g.Load(SceneMenu, nil) })
	if exp != nil {
		// In an expedition a win moves on; a crash retries with what is left.
		again := AddButton(s, 100, 470, 220, 60, "Retry", func() { ses.Design = Design{}; g.Load(SceneBuild, ses) })
		again.Icon = IconRover
		again.Disabled = ra.Outcome == Won || exp.Over()
		label := "Next mission"
		if exp.Over() {
			label = "Finish"
		}
		next := AddButton(s, 730, 470, 206, 60, label, func() {
			if exp.Over() {
				g.Load(SceneMenu, nil)
				return
			}
			g.Load(SceneTravel, exp)
		})
		next.Primary, next.Icon = true, IconRight
		next.Disabled = ra.Outcome != Won && !exp.Over()
		return s
	}
	AddButton(s, 100, 470, 220, 60, "Retry", func() { g.Load(SceneBuild, ses) }).Icon = IconRover
	next := AddButton(s, 730, 470, 206, 60, "Next mission", func() {
		n := NewSessionFor(g.Host, GetLevel(lvl.ID+1))
		n.Name, n.Spare = ses.Name, ra.Salvaged // the rover and its salvage go on
		g.Load(SceneBuild, n)
	})
	next.Primary, next.Icon = true, IconRight
	next.Disabled = ra.Outcome != Won || GetLevel(lvl.ID+1) == nil
	return s
}

// roverName is the rover's name for the debrief.
func (ra *ResultArg) roverName() string {
	if ra.Name == "" {
		return "Rover"
	}
	return ra.Name
}
