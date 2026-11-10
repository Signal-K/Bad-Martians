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
	run.Speed = DriveSpeed
	s.Cam.Zoom = 38
	s.Cam.Pos = run.Rover.Center()
	endTimer := 0.0

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
		// the rover's name, and whether it is still clean enough to salvage from
		state, sc := "no damage", colGood
		if run.Damaged() {
			state, sc = "scraped: no salvage", colWarn
		}
		name := ses.Name
		if name == "" {
			name = "Rover"
		}
		r.Text(name, 146, 62, sizeSmall, colInk, engine.AlignLeft)
		r.Text(state, 146+textW(name, sizeSmall)+14, 62, sizeSmall, sc, engine.AlignLeft)
		if run.Outcome != Running && (run.Outcome != Scrapped || endTimer > 1.1) {
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
	// Photo: snap the world without the HUD, with a caption, to share.
	flash := 0.0
	shots := 0
	cam := AddButton(s, ScreenW-114, 10, 90, 44, "Photo", func() {
		shots++
		g.RequestPhoto(fmt.Sprintf("haphazard-martians-mission-%d-%d.png", lvl.ID, shots))
		flash = 1
	})
	cam.Icon, cam.HideLabel = IconCamera, true
	s.OnPhoto = func(r engine.Renderer) {
		line := fmt.Sprintf("Mission %d · %s · %.0f m", lvl.ID, lvl.Name, run.Progress()*(lvl.GoalX-lvl.StartX))
		if ses.Name != "" {
			line = ses.Name + " · " + line
		}
		w := textW(line, sizeSmall) + 206
		r.Rect(16, ScreenH-46, w, 32, colTile.Alpha(200), true)
		r.Text("HAPHAZARD MARTIANS", 28, ScreenH-40, 13, colNeon, engine.AlignLeft)
		r.Text(line, 198, ScreenH-41, sizeSmall, colInk, engine.AlignLeft)
	}
	AddDraw(s, 40, func(_ engine.Camera, r engine.Renderer) {
		if flash <= 0 {
			return
		}
		r.Rect(0, 0, ScreenW, ScreenH, engine.Color{R: 255, G: 255, B: 255, A: uint8(200 * flash * flash)}, true)
		if flash < 0.85 {
			msg := "Photo saved"
			w := textW(msg, sizeBody) + 32
			r.Rect(ScreenW-24-w, 64, w, 36, colPanel.Alpha(uint8(240*min(1, flash*3))), true)
			r.Text(msg, ScreenW-24-w/2, 72, sizeBody, colInk.Alpha(uint8(255*min(1, flash*3))), engine.AlignCenter)
		}
	})

	// Stuck: offer to scrap the rover and start again, at the cost of its parts.
	boom := -1.0 // seconds since the rover was scrapped; <0 before
	restart := AddButton(s, ScreenW/2-130, ScreenH-104, 260, 74, "Restart", func() {
		run.Scrap()
		boom = 0
	})
	restart.Icon, restart.Primary, restart.Hidden = IconRetry, true, true
	AddDraw(s, 15, func(cam engine.Camera, r engine.Renderer) {
		if !restart.Hidden {
			msg := "Stuck? Restart scraps this rover."
			if ses.Exp != nil {
				msg = fmt.Sprintf("Stuck? Restart scraps this rover: %d parts lost.", ses.Design.Cost())
			}
			w := textW(msg, sizeSmall) + 28
			r.Rect(ScreenW/2-w/2, ScreenH-142, w, 30, colPanel.Alpha(230), true)
			r.Text(msg, ScreenW/2, ScreenH-137, sizeSmall, colInk, engine.AlignCenter)
			p := 0.5 + 0.5*sinf(run.Time*6)
			outline(r, restart.X-3-3*p, restart.Y-3-3*p, restart.W+6+6*p, restart.H+6+6*p, colWarn.Alpha(uint8(140+110*p)))
		}
	})
	AddDraw(s, 6, func(cam engine.Camera, r engine.Renderer) { // the blast, in the world
		if boom < 0 || boom > 1.2 {
			return
		}
		c := cam.ToScreen(run.Rover.Center())
		k := boom / 1.2
		rad := (0.6 + 4*k) * cam.Zoom
		r.Circle(c, rad, colWarn.Alpha(uint8(200*(1-k))), true)
		r.Circle(c, rad*0.6, colChipLt.Alpha(uint8(230*(1-k))), true)
		for i := 0; i < 10; i++ {
			a := float64(i) * 0.628
			d := rad * (1.1 + 0.3*float64(i%3))
			r.Line(c.Add(engine.V(cosf(a)*rad*0.7, sinf(a)*rad*0.7)), c.Add(engine.V(cosf(a)*d, sinf(a)*d)), 3, colChipLt.Alpha(uint8(255*(1-k))))
		}
	})
	// The stick: right drives, left reverses, up and down lean the rover.
	// It sits bottom-left, under the left thumb.
	stick := &Stick{Home: engine.V(120, ScreenH-112), R: 62, Grab: 150}
	stick.Block = func(p engine.Vec2) bool { // the Restart button takes its own taps
		return !restart.Hidden && p.X >= restart.X && p.X <= restart.X+restart.W && p.Y >= restart.Y && p.Y <= restart.Y+restart.H
	}
	so := s.Spawn("stick")
	so.Layer = 20
	so.Add(stick)
	AddDraw(s, 19, func(_ engine.Camera, r engine.Renderer) {
		if !stick.Held() {
			r.Text("drive", stick.Home.X, stick.Home.Y+stick.R+18, 12, colMuted, engine.AlignCenter)
		}
	})

	// tutorial: the last step is learning to drive
	if tut := ses.Tut; tut != nil && !tut.Done {
		clock := 0.0
		o := s.Spawn("tutorial")
		o.Layer = 30
		o.Add(DrawFunc(func(_ engine.Camera, r engine.Renderer) {
			if tut.Done {
				return
			}
			clock += 1.0 / 60
			drawCoach(r, clock, engine.Rect{X: ScreenW/2 - 190, Y: 76, W: 380, H: 48},
				"Push the stick right to drive.", stick.Box())
		}))
	}

	s.OnUpdate = func(dt float64) {
		th, lean := stick.X, stick.Y
		if g.In.Right {
			th = 1
		}
		if g.In.Left {
			th = -1
		}
		if g.In.Up {
			lean = 1
		}
		if g.In.DownKey {
			lean = -1
		}
		if tut := ses.Tut; tut != nil && !tut.Done && th > 0.5 {
			if tut.Gas += dt; tut.Gas >= tutGasSeconds {
				FinishTutorial(g.Host, tut)
			}
		}
		flash = max(0, flash-dt/1.6)
		run.Lean = lean
		run.Step(dt, th)
		run.Debris(dt)
		if boom >= 0 {
			boom += dt
		}
		restart.Hidden = !run.IsStuck()
		c := run.Rover.Center()
		target := engine.V(c.X+3, c.Y+1)
		s.Cam.Pos = s.Cam.Pos.Add(target.Sub(s.Cam.Pos).Mul(0.12))
		if run.Outcome != Running {
			endTimer += dt
			if endTimer > 1.6+map[bool]float64{true: 0.8}[run.Outcome == Scrapped] {
				g.Load(SceneResult, &ResultArg{Session: ses, Outcome: run.Outcome, Stars: run.Stars(),
					Time: run.Time, Samples: run.SamplesGot(), Total: len(run.Got), Score: run.Score(),
					Distance: run.Progress() * (lvl.GoalX - lvl.StartX), TopSpeed: run.TopSpeed, MaxAir: run.MaxAir,
					Damaged: run.Damaged()})
			}
		}
	}
	return s
}
