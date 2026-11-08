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
	Recorded bool // progress saved and run event sent (set on first show)
}

// sampleResult is the standalone fixture: a won mission with plausible numbers.
func sampleResult(g *engine.Game) *ResultArg {
	l := GetLevel(1)
	ses := NewSessionFor(g.Host, l)
	return &ResultArg{Session: ses, Outcome: Won, Stars: 2, Time: l.Par * 1.2, Samples: 2, Total: 2,
		Score: ScoreFor(l, true, 2, l.Par*1.2), Recorded: true}
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
			"cost": ses.Design.Cost(), "design": design,
		})
		g.Host.Emit("run", payload)
		if ses.Exp != nil {
			ses.Exp.Finish(ra.Outcome == Won, ses.Design.Cost(), ra.Score.Total())
			if ses.Exp.Over() && ses.Exp.Score > prog.BestExpedition {
				prog.BestExpedition = ses.Exp.Score
				prog.Save(g.Host)
			}
		}
	}
	exp := ses.Exp
	best := prog.Levels[lvl.ID].Score
	reviewed := len(ses.Labels)

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

		drawPanel(r, 28, 124, 520, 310, "Mission report")
		drawStars(r, 288, 168, 22, ra.Stars)
		rows := []struct {
			n string
			v int
		}{{"Finish", ra.Score.Finish}, {fmt.Sprintf("Samples %d/%d", ra.Samples, ra.Total), ra.Score.Samples},
			{fmt.Sprintf("Time %s (par %s)", fmtTime(ra.Time), fmtTime(lvl.Par)), ra.Score.Time}}
		for i, row := range rows {
			y := 218 + float64(i)*36
			r.Text(row.n, 56, y, sizeBody, colInk, engine.AlignLeft)
			r.Text(fmt.Sprintf("+%d", row.v), 520, y, sizeBody, colInk, engine.AlignRight)
			r.Line(engine.V(56, y+28), engine.V(520, y+28), 1, colMuted.Alpha(90))
		}
		r.Text("Score", 56, 336, 24, colNeon, engine.AlignLeft)
		r.Text(fmt.Sprint(ra.Score.Total()), 520, 336, 24, colNeon, engine.AlignRight)
		r.Text(fmt.Sprintf("Best %d   ·   Rover cost %d credits", best, ses.Design.Cost()), 56, 384, sizeSmall, colMuted, engine.AlignLeft)

		drawPanel(r, 572, 124, 360, 310, "Mission Control")
		r.Text("Second opinion needed.", 596, 150, sizeBody, colInk, engine.AlignLeft)
		for i, ln := range []string{"Review the accuracy of the navcam's", "terrain read for this mission. Every", "review earns credits for your next", "rover build."} {
			r.Text(ln, 596, 184+float64(i)*22, sizeSmall, colMuted, engine.AlignLeft)
		}
		if exp != nil {
			msg := fmt.Sprintf("Expedition: %d parts left · score %d", exp.Parts, exp.Score)
			if exp.Lost > 0 {
				msg = fmt.Sprintf("Crash: %d parts lost · %d left", exp.Lost, exp.Parts)
			}
			r.Text(msg, 56, 408, sizeSmall, colNeon, engine.AlignLeft)
		}
		drawChip(r, 596, 380, IconReview, fmt.Sprintf("%d/%d reviewed", reviewed, len(lvl.Segments)), colChipLt)
	})
	addBack(s, func() { g.Load(SceneMenu, nil) })
	rv := AddButton(s, 596, 290, 312, 64, "Review accuracy", func() {
		g.Load(SceneReview, &ReviewArg{Session: ses, Return: SceneResult, ReturnArg: ra})
	})
	rv.Icon = IconReview
	if exp != nil {
		// In an expedition a win moves on; a crash retries with what is left.
		again := AddButton(s, 100, 470, 220, 60, "Retry", func() { g.Load(SceneBuild, ses) })
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
			g.Load(SceneBuild, ExpeditionSession(g.Host, exp))
		})
		next.Primary, next.Icon = true, IconRight
		next.Disabled = ra.Outcome != Won && !exp.Over()
		return s
	}
	AddButton(s, 100, 470, 220, 60, "Retry", func() { g.Load(SceneBuild, ses) }).Icon = IconRover
	next := AddButton(s, 730, 470, 206, 60, "Next mission", func() {
		g.Load(SceneBuild, NewSessionFor(g.Host, GetLevel(lvl.ID+1)))
	})
	next.Primary, next.Icon = true, IconRight
	next.Disabled = ra.Outcome != Won || GetLevel(lvl.ID+1) == nil
	return s
}
