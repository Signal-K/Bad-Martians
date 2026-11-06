package game

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/signal-k/bad-martians/engine"
)

// Setup registers every scene and opens the menu.
func Setup(g *engine.Game) {
	g.Register("menu", menuScene)
	g.Register("label", labelScene)
	g.Register("build", buildScene)
	g.Register("run", runScene)
	g.Register("result", resultScene)
	g.Load("menu", nil)
}

func newScene() *engine.Scene {
	s := &engine.Scene{}
	s.Cam = engine.Camera{Zoom: 1, Pos: engine.V(ScreenW/2, ScreenH/2)}
	return s
}

func title(s *engine.Scene, text string) {
	AddDraw(s, 9, func(_ engine.Camera, r engine.Renderer) {
		r.Text(text, 140, 22, 26, colText, engine.AlignLeft)
	})
}

// ---- Menu ------------------------------------------------------------------

func menuScene(g *engine.Game, _ any) *engine.Scene {
	s := newScene()
	prog := LoadProgress(g.Host)
	AddDraw(s, 0, func(_ engine.Camera, r engine.Renderer) {
		drawSky(r, 0)
		r.Text("BAD MARTIANS", ScreenW/2, 40, 64, colText, engine.AlignCenter)
		r.Text("Label the terrain. Build a rover. Cross Mars.", ScreenW/2, 118, 20, colDim, engine.AlignCenter)
	})
	const cw, ch, gap = 162.0, 190.0, 12.0
	x0 := (ScreenW - (cw*5 + gap*4)) / 2
	for i, def := range LevelDefs {
		def := def
		x, y := x0+float64(i)*(cw+gap), 190.0
		b := AddButton(s, x, y, cw, ch, "", func() { g.Load("label", NewSession(BuildLevel(def))) })
		b.Disabled = !prog.Unlocked(def.ID)
		stars := prog.Levels[def.ID].Stars
		b.Custom = func(b *Button, r engine.Renderer) {
			fill := colPanel
			if !b.Disabled {
				fill = colPanel2
				if b.Held {
					fill = colAccent
				}
			}
			r.Rect(b.X, b.Y, b.W, b.H, fill, true)
			r.Rect(b.X, b.Y, b.W, b.H, colDim.Alpha(120), false)
			r.Text(fmt.Sprint(def.ID), b.X+b.W/2, b.Y+10, 48, colText, engine.AlignCenter)
			r.Text(def.Name, b.X+b.W/2, b.Y+78, 17, colText, engine.AlignCenter)
			r.Text(fmt.Sprintf("%.0f m", def.Length), b.X+b.W/2, b.Y+104, 15, colDim, engine.AlignCenter)
			if b.Disabled {
				r.Text("Locked", b.X+b.W/2, b.Y+148, 16, colDim, engine.AlignCenter)
			} else {
				drawStars(r, b.X+b.W/2, b.Y+158, 11, stars)
			}
		}
	}
	AddDraw(s, 5, func(_ engine.Camera, r engine.Renderer) {
		r.Text("Finish a level to unlock the next.", ScreenW/2, 410, 16, colDim, engine.AlignCenter)
		r.Text("Terrain classes follow the AI4Mars labelling scheme.", ScreenW/2, 500, 14, colDim, engine.AlignCenter)
	})
	return s
}

// ---- Label (AI4Mars activity) ---------------------------------------------

// paintTerrainSample draws a pseudo-photo of a terrain class in a panel. The
// look is deterministic per (level, segment) so players see a stable image.
func paintTerrainSample(r engine.Renderer, lvl *Level, idx int, x, y, w, h float64) {
	seg := lvl.Segments[idx]
	rg := newRng(lvl.Seed*131 + uint64(idx)*7919)
	base := Classes[seg.Class].Color
	j := func() uint8 { return uint8(rg.intn(24)) }
	base = engine.RGB(clamp8(int(base.R)+int(j())-12), clamp8(int(base.G)+int(j())-12), clamp8(int(base.B)+int(j())-12))
	if seg.Class == BigRock {
		base = Classes[Soil].Color
	}
	r.Rect(x, y, w, h, base, true)
	switch seg.Class {
	case Soil:
		for i := 0; i < 70; i++ {
			c := engine.RGB(clamp8(int(base.R)-30+rg.intn(60)), clamp8(int(base.G)-24+rg.intn(48)), clamp8(int(base.B)-20+rg.intn(40)))
			r.Circle(engine.V(x+rg.float()*w, y+rg.float()*h), 1+rg.float()*2.2, c, true)
		}
	case Sand:
		for i := 0; i < 9; i++ {
			yy := y + 8 + float64(i)*h/9
			prev := engine.V(x, yy)
			for xx := 6.0; xx <= w; xx += 6 {
				p := engine.V(x+xx, yy+3*sinf(xx*0.18+float64(i)*1.7))
				r.Line(prev, p, 1.4, lerpColor(base, engine.RGB(255, 226, 170), 0.5))
				prev = p
			}
		}
	case Bedrock:
		dark := lerpColor(base, engine.RGB(30, 24, 24), 0.55)
		for row := 0; row < 4; row++ {
			yy := y + float64(row)*h/4
			r.Line(engine.V(x, yy), engine.V(x+w, yy), 2, dark)
			off := rg.rangef(10, w-10)
			r.Line(engine.V(x+off, yy), engine.V(x+off+rg.rangef(-8, 8), yy+h/4), 2, dark)
		}
	case BigRock:
		for i := 0; i < 70; i++ {
			r.Circle(engine.V(x+rg.float()*w, y+rg.float()*h), 1+rg.float()*1.8, lerpColor(base, engine.RGB(40, 26, 20), 0.3), true)
		}
		for i := 0; i < 2+rg.intn(2); i++ {
			c := engine.V(x+14+rg.float()*(w-28), y+22+rg.float()*(h-44))
			rad := 12 + rg.float()*14
			r.Circle(c, rad, engine.RGB(70, 58, 58), true)
			r.Circle(c.Add(engine.V(-rad*0.25, -rad*0.3)), rad*0.55, engine.RGB(104, 90, 88), true)
		}
	}
}

func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func labelScene(g *engine.Game, arg any) *engine.Scene {
	ses := arg.(*Session)
	lvl := ses.Level
	s := newScene()
	sel := 0
	n := len(lvl.Segments)
	const pw, ph, pgap, py = 106.0, 150.0, 6.0, 112.0
	px0 := (ScreenW - (pw*float64(n) + pgap*float64(n-1))) / 2

	AddDraw(s, 0, func(_ engine.Camera, r engine.Renderer) {
		r.Rect(0, 0, ScreenW, ScreenH, colPanel, true)
		r.Text(fmt.Sprintf("SURVEY  ·  Level %d  %s", lvl.ID, lvl.Name), 30, 22, 26, colText, engine.AlignLeft)
		r.Text("Navcam strip, left to right. Pick a panel, then say what the ground is made of.", 30, 70, 16, colDim, engine.AlignLeft)
		r.Text(fmt.Sprintf("Labelled %d/%d   ·   Parts budget +%d credits (now %d)", len(ses.Labels), n, ses.Bonus(), ses.Budget()),
			30, 290, 16, colText, engine.AlignLeft)
	})
	for i := 0; i < n; i++ {
		i := i
		b := AddButton(s, px0+float64(i)*(pw+pgap), py, pw, ph, "", func() { sel = i })
		b.Custom = func(b *Button, r engine.Renderer) {
			paintTerrainSample(r, lvl, i, b.X, b.Y, b.W, b.H)
			if c, ok := ses.Labels[i]; ok {
				r.Rect(b.X, b.Y+b.H-26, b.W, 26, colPanel.Alpha(220), true)
				r.Text(c.Info().Name, b.X+b.W/2, b.Y+b.H-22, 15, colText, engine.AlignCenter)
			} else {
				r.Text("?", b.X+b.W/2, b.Y+b.H-30, 22, colText, engine.AlignCenter)
			}
			if i == sel {
				r.Rect(b.X-3, b.Y-3, b.W+6, b.H+6, colAccent, false)
				r.Rect(b.X-4, b.Y-4, b.W+8, b.H+8, colAccent, false)
			}
		}
	}
	cw := (ScreenW - 60 - 3*10) / 4
	for c := Class(0); c < NumClasses; c++ {
		c := c
		b := AddButton(s, 30+float64(c)*(cw+10), 330, cw, 60, "", func() {
			ses.Labels[sel] = c
			for k := 1; k <= n; k++ { // jump to the next unlabelled panel
				j := (sel + k) % n
				if _, done := ses.Labels[j]; !done {
					sel = j
					break
				}
			}
		})
		b.Custom = func(b *Button, r engine.Renderer) {
			fill := colPanel2
			if b.Held {
				fill = colAccent
			}
			r.Rect(b.X, b.Y, b.W, b.H, fill, true)
			r.Rect(b.X, b.Y, 10, b.H, Classes[c].Color, true)
			r.Text(Classes[c].Name, b.X+24, b.Y+8, 20, colText, engine.AlignLeft)
			r.Text(Classes[c].Hint, b.X+24, b.Y+34, 12, colDim, engine.AlignLeft)
			if cur, ok := ses.Labels[sel]; ok && cur == c {
				r.Rect(b.X, b.Y, b.W, b.H, colText, false)
			}
		}
	}
	AddDraw(s, 5, func(_ engine.Camera, r engine.Renderer) {
		r.Text("Each label earns +1 credit. Some panels are calibration images: get those right for +2 more.", 30, 410, 14, colDim, engine.AlignLeft)
	})
	AddButton(s, 30, 450, 140, 52, "Menu", func() { g.Load("menu", nil) })
	AddButton(s, 190, 450, 200, 52, "Skip survey", func() { g.Load("build", ses) })
	done := AddButton(s, ScreenW-250, 450, 220, 52, "Done: build rover", func() {
		ses.SubmitLabels(g.Host)
		g.Load("build", ses)
	})
	done.Fill = colAccent
	s.OnUpdate = func(float64) { done.Disabled = len(ses.Labels) == 0 }
	return s
}

// ---- Build ----------------------------------------------------------------

const (
	bCell = 44.0
	bGX   = 30.0
	bGY   = 100.0
)

func buildScene(g *engine.Game, arg any) *engine.Scene {
	ses := arg.(*Session)
	lvl := ses.Level
	s := newScene()
	// Starter rover only if it fits this level's budget.
	if len(ses.Design.Cells) > 0 && ses.Design.Cost() > ses.Budget() {
		ses.Design = Design{}
	}
	tool := KWheel
	painting := false
	lastCell := [2]int{-1, -1}
	var errs []string

	cellRect := func(cx, cy int) (float64, float64) {
		return bGX + float64(cx)*bCell, bGY + float64(GridH-1-cy)*bCell
	}
	AddDraw(s, 0, func(_ engine.Camera, r engine.Renderer) {
		r.Rect(0, 0, ScreenW, ScreenH, colPanel, true)
		r.Text(fmt.Sprintf("BUILD  ·  Level %d  %s", lvl.ID, lvl.Name), 140, 22, 26, colText, engine.AlignLeft)
		used, budget := ses.Design.Cost(), ses.Budget()
		cc := colText
		if used > budget {
			cc = colBad
		}
		r.Text(fmt.Sprintf("Credits %d / %d", used, budget), ScreenW-30, 24, 22, cc, engine.AlignRight)
		if len(errs) > 0 {
			r.Text(errs[0], 30, 66, 17, colBad, engine.AlignLeft)
		} else {
			r.Text("Ready to launch. Hold right (or the Gas button) to drive.", 30, 66, 17, colGood, engine.AlignLeft)
		}
		// grid
		r.Rect(bGX, bGY, bCell*GridW, bCell*GridH, colPanel2.Alpha(140), true)
		for x := 0; x <= GridW; x++ {
			r.Line(engine.V(bGX+float64(x)*bCell, bGY), engine.V(bGX+float64(x)*bCell, bGY+bCell*GridH), 1, colDim.Alpha(50))
		}
		for y := 0; y <= GridH; y++ {
			r.Line(engine.V(bGX, bGY+float64(y)*bCell), engine.V(bGX+bCell*GridW, bGY+float64(y)*bCell), 1, colDim.Alpha(50))
		}
		r.Rect(bGX, bGY+bCell*GridH, bCell*GridW, 8, Classes[Soil].Color, true)
		for _, c := range ses.Design.Cells {
			for _, o := range [][2]int{{1, 0}, {0, 1}, {1, 1}, {-1, 1}} {
				if _, ok := ses.Design.At(c.X+o[0], c.Y+o[1]); ok {
					ax, ay := cellRect(c.X, c.Y)
					bx, by := cellRect(c.X+o[0], c.Y+o[1])
					drawBeam(r, engine.V(ax+bCell/2, ay+bCell/2), engine.V(bx+bCell/2, by+bCell/2), bCell)
				}
			}
		}
		for _, c := range ses.Design.Cells {
			x, y := cellRect(c.X, c.Y)
			drawPart(r, c.K, engine.V(x+bCell/2, y+bCell/2), bCell, 0)
		}
		// course preview, coloured by the player's own labels
		const sx, sy, sw, sh = 30.0, 440.0, 670.0, 44.0
		r.Text("Course, as you labelled it (grey = unlabelled)", sx, sy-22, 14, colDim, engine.AlignLeft)
		x0, x1 := lvl.Terrain.X0, lvl.Terrain.X0+lvl.Terrain.DX*float64(lvl.Terrain.Len()-1)
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, h := range lvl.Terrain.H {
			lo, hi = math.Min(lo, h), math.Max(hi, h)
		}
		hh := math.Max(hi-lo, 2)
		pt := func(x, h float64) engine.Vec2 {
			return engine.V(sx+(x-x0)/(x1-x0)*sw, sy+sh-4-(h-lo)/hh*(sh-12))
		}
		for i, seg := range lvl.Segments {
			col := engine.RGB(110, 100, 100)
			if c, ok := ses.Labels[i]; ok {
				col = Classes[c].Color
			}
			var poly []engine.Vec2
			poly = append(poly, engine.V(pt(seg.X0, 0).X, sy+sh))
			for x := seg.X0; x <= seg.X1; x += 1 {
				poly = append(poly, pt(x, lvl.Terrain.HeightAt(x)))
			}
			poly = append(poly, pt(seg.X1, lvl.Terrain.HeightAt(seg.X1)), engine.V(pt(seg.X1, 0).X, sy+sh))
			r.Poly(poly, col)
		}
		for _, sm := range lvl.Samples {
			drawStar(r, pt(sm.X, lvl.Terrain.HeightAt(sm.X)+0.5).X, sy+4, 6, colStar)
		}
		gx := pt(lvl.GoalX, 0).X
		r.Line(engine.V(gx, sy), engine.V(gx, sy+sh), 2, colText)
		// palette selection marker
	})

	// palette
	tools := []Kind{KFrame, KWheel, KMotor, KMast, ""}
	for i, k := range tools {
		k := k
		b := AddButton(s, 600, bGY+float64(i)*58, 330, 52, "", func() { tool = k })
		b.Custom = func(b *Button, r engine.Renderer) {
			fill := colPanel2
			if tool == k {
				fill = engine.RGB(110, 62, 44)
			}
			r.Rect(b.X, b.Y, b.W, b.H, fill, true)
			if tool == k {
				r.Rect(b.X, b.Y, b.W, b.H, colAccent, false)
			}
			if k == "" {
				r.Text("Erase", b.X+70, b.Y+16, 20, colText, engine.AlignLeft)
				return
			}
			p, _ := PartOf(k)
			drawPart(r, k, engine.V(b.X+34, b.Y+b.H/2+4), 36, 0)
			r.Text(p.Name, b.X+70, b.Y+16, 20, colText, engine.AlignLeft)
			r.Text(fmt.Sprintf("%d cr", p.Cost), b.X+b.W-14, b.Y+16, 18, colDim, engine.AlignRight)
		}
	}
	AddButton(s, 30, 16, 90, 40, "Back", func() { g.Load("label", ses) })
	go_ := AddButton(s, 730, 430, 200, 62, "GO", func() { g.Load("run", ses) })
	go_.Fill = colAccent

	s.OnUpdate = func(float64) {
		errs = ses.Design.Validate(ses.Budget())
		go_.Disabled = len(errs) > 0
		in := g.In
		cx := int((in.Pointer.X - bGX) / bCell)
		cy := GridH - 1 - int((in.Pointer.Y-bGY)/bCell)
		inGrid := in.Pointer.X >= bGX && in.Pointer.Y >= bGY && cx >= 0 && cx < GridW && cy >= 0 && cy < GridH &&
			in.Pointer.X < bGX+bCell*GridW && in.Pointer.Y < bGY+bCell*GridH
		if in.Pressed && inGrid {
			painting, lastCell = true, [2]int{-1, -1}
		}
		if !in.Down {
			painting = false
		}
		if painting && inGrid && (cx != lastCell[0] || cy != lastCell[1]) {
			lastCell = [2]int{cx, cy}
			ses.Design.Set(cx, cy, tool)
		}
	}
	return s
}

// ---- Run ------------------------------------------------------------------

func runScene(g *engine.Game, arg any) *engine.Scene {
	ses := arg.(*Session)
	lvl := ses.Level
	s := newScene()
	run := NewRun(lvl, ses.Design)
	s.Cam.Zoom = 38
	s.Cam.Pos = run.Rover.Center()
	endTimer := 0.0
	var gas, brake *Button

	ro := s.Spawn("rover")
	ro.Layer = 5
	ro.Add(run.Rover)

	AddDraw(s, 0, func(cam engine.Camera, r engine.Renderer) {
		drawSky(r, cam.Pos.X)
		t := lvl.Terrain
		half := cam.Screen.X/2/cam.Zoom + 1
		i0 := int((cam.Pos.X-half-t.X0)/t.DX) - 1
		i1 := int((cam.Pos.X+half-t.X0)/t.DX) + 1
		if i0 < 0 {
			i0 = 0
		}
		if i1 > t.Len()-2 {
			i1 = t.Len() - 2
		}
		for i := i0; i <= i1; i++ {
			a := cam.ToScreen(engine.V(t.X0+float64(i)*t.DX, t.H[i]))
			b := cam.ToScreen(engine.V(t.X0+float64(i+1)*t.DX, t.H[i+1]))
			col := Classes[t.Seg[i]].Color
			if i%2 == 0 {
				col = lerpColor(col, engine.RGB(0, 0, 0), 0.06)
			}
			r.Poly([]engine.Vec2{a, b, engine.V(b.X+0.5, ScreenH+4), engine.V(a.X-0.5, ScreenH+4)}, col)
			r.Line(a, b, 3, lerpColor(col, engine.RGB(255, 235, 210), 0.35))
		}
		// goal flag
		gb := cam.ToScreen(engine.V(lvl.GoalX, t.HeightAt(lvl.GoalX)))
		r.Line(gb, gb.Add(engine.V(0, -4*cam.Zoom)), 4, colText)
		r.Poly([]engine.Vec2{gb.Add(engine.V(2, -4*cam.Zoom)), gb.Add(engine.V(2+1.6*cam.Zoom, -3.5*cam.Zoom)), gb.Add(engine.V(2, -3*cam.Zoom))}, colAccent)
		// samples
		for i, sm := range lvl.Samples {
			if run.Got[i] {
				continue
			}
			c := cam.ToScreen(sm)
			bob := 3 * sinf(run.Time*3+float64(i))
			r.Circle(engine.V(c.X, c.Y+bob), 14, engine.RGB(120, 220, 255).Alpha(60), true)
			drawStar(r, c.X, c.Y+bob, 10, engine.RGB(140, 230, 255))
		}
	})
	// HUD
	AddDraw(s, 10, func(_ engine.Camera, r engine.Renderer) {
		r.Rect(330, 18, 300, 10, colPanel.Alpha(200), true)
		r.Rect(330, 18, 300*run.Progress(), 10, colAccent, true)
		r.Text(fmtTime(run.Time), 330, 34, 20, colText, engine.AlignLeft)
		for i, got := range run.Got {
			c := colPanel2
			if got {
				c = engine.RGB(140, 230, 255)
			}
			drawStar(r, 618-float64(len(run.Got)-1-i)*26, 48, 10, c)
		}
		if run.Outcome != Running {
			col := colGood
			if run.Outcome != Won {
				col = colBad
			}
			r.Text(run.Outcome.Message(), ScreenW/2, 220, 34, col, engine.AlignCenter)
		}
	})
	AddButton(s, 30, 16, 90, 40, "Abort", func() { g.Load("build", ses) })
	gas = AddButton(s, ScreenW-190, ScreenH-110, 160, 80, "GAS", nil)
	brake = AddButton(s, 30, ScreenH-110, 160, 80, "REVERSE", nil)
	gas.Fill, brake.Fill = engine.RGB(120, 70, 40), engine.RGB(70, 50, 60)

	s.OnUpdate = func(dt float64) {
		th := 0.0
		if g.In.Right || gas.Held {
			th++
		}
		if g.In.Left || brake.Held {
			th--
		}
		run.Step(dt, th)
		c := run.Rover.Center()
		target := engine.V(c.X+3, c.Y+1)
		s.Cam.Pos = s.Cam.Pos.Add(target.Sub(s.Cam.Pos).Mul(0.12))
		if run.Outcome != Running {
			endTimer += dt
			if endTimer > 1.6 {
				g.Load("result", &ResultArg{Session: ses, Outcome: run.Outcome, Stars: run.Stars(),
					Time: run.Time, Samples: run.SamplesGot(), Total: len(run.Got)})
			}
		}
	}
	return s
}

// ---- Result ----------------------------------------------------------------

type ResultArg struct {
	Session *Session
	Outcome Outcome
	Stars   int
	Time    float64
	Samples int
	Total   int
}

func resultScene(g *engine.Game, arg any) *engine.Scene {
	ra := arg.(*ResultArg)
	ses := ra.Session
	lvl := ses.Level
	s := newScene()

	prog := LoadProgress(g.Host)
	if ra.Outcome == Won {
		prog.Record(lvl.ID, ra.Stars, ra.Time)
		prog.Save(g.Host)
	}
	design := make([]map[string]any, 0, len(ses.Design.Cells))
	for _, c := range ses.Design.Cells {
		design = append(design, map[string]any{"x": c.X, "y": c.Y, "k": string(c.K)})
	}
	payload, _ := json.Marshal(map[string]any{
		"level": lvl.ID, "won": ra.Outcome == Won, "outcome": ra.Outcome.Message(), "stars": ra.Stars,
		"time": math.Round(ra.Time*10) / 10, "samples": ra.Samples, "cost": ses.Design.Cost(), "design": design,
	})
	g.Host.Emit("run", payload)

	AddDraw(s, 0, func(_ engine.Camera, r engine.Renderer) {
		drawSky(r, 0)
		r.Rect(0, 90, ScreenW, 300, colPanel.Alpha(210), true)
		head, col := "MISSION COMPLETE", colGood
		if ra.Outcome != Won {
			head, col = "MISSION FAILED", colBad
		}
		r.Text(head, ScreenW/2, 110, 48, col, engine.AlignCenter)
		r.Text(ra.Outcome.Message(), ScreenW/2, 170, 20, colDim, engine.AlignCenter)
		drawStars(r, ScreenW/2, 240, 26, ra.Stars)
		r.Text(fmt.Sprintf("Time %s   ·   Samples %d/%d   ·   Cost %d credits", fmtTime(ra.Time), ra.Samples, ra.Total, ses.Design.Cost()),
			ScreenW/2, 300, 20, colText, engine.AlignCenter)
		r.Text("Stars: finish, all samples, then beat par "+fmtTime(lvl.Par), ScreenW/2, 340, 15, colDim, engine.AlignCenter)
	})
	AddButton(s, 90, 420, 220, 60, "Retry (edit rover)", func() { g.Load("build", ses) })
	AddButton(s, 370, 420, 220, 60, "Menu", func() { g.Load("menu", nil) })
	next := AddButton(s, 650, 420, 220, 60, "Next level", func() {
		g.Load("label", NewSession(GetLevel(lvl.ID+1)))
	})
	next.Fill = colAccent
	next.Disabled = ra.Outcome != Won || GetLevel(lvl.ID+1) == nil
	return s
}
