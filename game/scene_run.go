package game

import (
	"fmt"

	"github.com/signal-k/haphazard-martians/engine"
)

func runScene(g *engine.Game, arg any) *engine.Scene {
	ses := arg.(*Session)
	lvl := ses.Level
	s := newScene()
	run := NewRun(lvl, ses.Design)
	s.Cam.Zoom = 38
	s.Cam.Pos = run.Rover.Center()
	endTimer := 0.0
	var gas, rev *Button

	ro := s.Spawn("rover")
	ro.Layer = 5
	ro.Add(run.Rover)

	AddDraw(s, 0, func(cam engine.Camera, r engine.Renderer) {
		drawSky(r, cam.Pos.X)
		t := lvl.Terrain
		drawTerrain(r, cam, t)
		roverShadows(r, cam, t, run.Rover)
		gb := cam.ToScreen(engine.V(lvl.GoalX, t.HeightAt(lvl.GoalX)))
		r.Line(gb, gb.Add(engine.V(0, -4*cam.Zoom)), 4, colInk)
		r.Poly([]engine.Vec2{gb.Add(engine.V(2, -4*cam.Zoom)), gb.Add(engine.V(2+1.6*cam.Zoom, -3.5*cam.Zoom)), gb.Add(engine.V(2, -3*cam.Zoom))}, colMars)
		for i, sm := range lvl.Samples {
			if run.Got[i] {
				continue
			}
			c := cam.ToScreen(sm)
			bob := 3 * sinf(run.Time*3+float64(i))
			r.Circle(engine.V(c.X, c.Y+bob), 14, colChipLt.Alpha(60), true)
			drawStar(r, c.X, c.Y+bob, 10, colChipLt)
		}
	})
	// HUD: progress, clock, samples, and the live score
	AddDraw(s, 10, func(_ engine.Camera, r engine.Renderer) {
		drawPanel(r, 130, 8, 700, 48, "")
		r.Rect(146, 22, 300, 14, colPanel2, true)
		r.Rect(146, 22, 300*run.Progress(), 14, colNeon, true)
		outline(r, 146, 22, 300, 14, colInk)
		drawIcon(r, IconClock, 474, 29, 22, colInk)
		r.Text(fmtTime(run.Time), 490, 19, sizeBody, colInk, engine.AlignLeft)
		for i, got := range run.Got {
			c := colStarOff
			if got {
				c = colChipLt
			}
			drawStar(r, 600+float64(i)*26, 30, 10, c)
		}
		r.Text(fmt.Sprintf("Par %s", fmtTime(lvl.Par)), 700, 19, sizeSmall, colMuted, engine.AlignLeft)
		if run.Outcome != Running {
			col, head := colGood, "Mission complete"
			if run.Outcome != Won {
				col, head = colWarn, "Mission failed"
			}
			drawPanel(r, ScreenW/2-230, 190, 460, 110, "")
			r.Text(head, ScreenW/2, 206, 32, col, engine.AlignCenter)
			r.Text(run.Outcome.Message(), ScreenW/2, 252, sizeBody, colInk, engine.AlignCenter)
		}
	})
	ab := AddButton(s, 24, 10, 90, 44, "Abort", func() { g.Load(SceneBuild, ses) })
	ab.Icon = IconBack
	ab.Label, ab.HideLabel = "Abort", true
	gas = AddButton(s, ScreenW-234, ScreenH-110, 210, 86, "GAS", nil)
	gas.Icon = IconRight
	rev = AddButton(s, 24, ScreenH-110, 210, 86, "REVERSE", nil)
	rev.Icon = IconLeft

	s.OnUpdate = func(dt float64) {
		th := 0.0
		if g.In.Right || gas.Held {
			th++
		}
		if g.In.Left || rev.Held {
			th--
		}
		run.Step(dt, th)
		c := run.Rover.Center()
		target := engine.V(c.X+3, c.Y+1)
		s.Cam.Pos = s.Cam.Pos.Add(target.Sub(s.Cam.Pos).Mul(0.12))
		if run.Outcome != Running {
			endTimer += dt
			if endTimer > 1.6 {
				g.Load(SceneResult, &ResultArg{Session: ses, Outcome: run.Outcome, Stars: run.Stars(),
					Time: run.Time, Samples: run.SamplesGot(), Total: len(run.Got), Score: run.Score()})
			}
		}
	}
	return s
}
