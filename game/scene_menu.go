package game

import (
	"fmt"
	"math"

	"github.com/signal-k/haphazard-martians/engine"
)

func menuScene(g *engine.Game, _ any) *engine.Scene {
	s := newScene()
	prog := LoadProgress(g.Host)
	addBackdrop(s)
	AddDraw(s, 2, func(_ engine.Camera, r engine.Renderer) {
		// left-aligned so the Shape school button fits beside the title
		r.Text("HAPHAZARD MARTIANS", 56, 26, 52, colInk, engine.AlignLeft)
		r.Text("Build a rover. Cross Mars. Help us read its terrain.", 58, 86, sizeBody, colMuted, engine.AlignLeft)
		drawPanel(r, 56, 150, 848, 290, "Missions")
		total := 0
		for _, lp := range prog.Levels {
			total += lp.Score
		}
		r.Text(fmt.Sprintf("%d total score · expedition best %d", total, prog.BestExpedition), 60, 446, sizeSmall, colMuted, engine.AlignLeft)
		r.Text("Finish a mission to unlock the next.", 904, 446, sizeSmall, colMuted, engine.AlignRight)
	})
	// Tiles share the panel's width, so the row fits however many missions ship.
	const th, gap, pad = 232.0, 10.0, 16.0
	n := float64(len(LevelDefs))
	tw := math.Floor((848 - 2*pad - gap*(n-1)) / n)
	x0 := 56 + (848-(tw*n+gap*(n-1)))/2
	for i, def := range LevelDefs {
		def := def
		x, y := x0+float64(i)*(tw+gap), 184.0
		b := AddButton(s, x, y, tw, th, fmt.Sprintf("Mission %d", def.ID), func() {
			g.Load(SceneBuild, NewSessionFor(g.Host, BuildLevel(def)))
		})
		b.Disabled = !prog.Unlocked(def.ID)
		lp := prog.Levels[def.ID]
		b.Custom = func(b *Button, r engine.Renderer) {
			st := TileNormal
			switch {
			case b.Disabled:
				st = TileDisabled
			case b.Held:
				st = TileHeld
			}
			fill, ink, sub := colTile, colInk, colMuted
			line := colInk
			if st == TileHeld {
				fill, ink, sub = colInk, colTile, colTile
			}
			if st == TileDisabled {
				line, ink, sub = colMuted.Alpha(90), colMuted.Alpha(120), colMuted.Alpha(100)
			}
			r.Rect(b.X, b.Y, b.W, b.H, fill, true)
			outline(r, b.X, b.Y, b.W, b.H, line)
			cx := b.X + b.W/2
			r.Text(fmt.Sprint(def.ID), cx, b.Y+14, 52, ink, engine.AlignCenter)
			r.Text(def.Name, cx, b.Y+84, 15, ink, engine.AlignCenter)
			r.Text(fmt.Sprintf("%.0f m", def.Length), cx, b.Y+108, sizeSmall, sub, engine.AlignCenter)
			if b.Disabled {
				drawIcon(r, IconGear, cx, b.Y+168, 34, ink)
				return
			}
			drawStars(r, cx, b.Y+160, 11, lp.Stars)
			if lp.Score > 0 {
				r.Text(fmt.Sprintf("best %d", lp.Score), cx, b.Y+190, sizeSmall, sub, engine.AlignCenter)
			} else {
				r.Text("not yet run", cx, b.Y+190, sizeSmall, sub, engine.AlignCenter)
			}
		}
	}
	if m, ok, next := CurrentMission(); ok || next != nil {
		label := "Daily mission"
		if !ok {
			m, label = *next, "Coming "+next.Date
		}
		b := AddButton(s, 500, 462, 340, 64, label, func() {
			g.Load(SceneBuild, NewSessionFor(g.Host, BuildLevel(m.Level)))
		})
		b.Disabled = !ok
		b.Custom = func(b *Button, r engine.Renderer) {
			fill, ink, sub, line := colTile, colInk, colMuted, colInk
			switch {
			case b.Disabled:
				line, ink, sub = colMuted.Alpha(90), colMuted.Alpha(120), colMuted.Alpha(100)
			case b.Held:
				fill, ink, sub = colInk, colTile, colTile
			}
			r.Rect(b.X, b.Y, b.W, b.H, fill, true)
			outline(r, b.X, b.Y, b.W, b.H, line)
			r.Text(label, b.X+14, b.Y+8, sizeSmall, colNeon, engine.AlignLeft)
			r.Text(m.Title, b.X+14, b.Y+26, 18, ink, engine.AlignLeft)
			rw := fmt.Sprintf("+%d parts", m.Reward.Credits)
			if m.Reward.FreeRetry > 0 {
				rw += fmt.Sprintf("  +%d free retry", m.Reward.FreeRetry)
			}
			r.Text(rw, b.X+b.W-14, b.Y+8, sizeSmall, sub, engine.AlignRight)
			if lp := prog.Levels[m.Level.ID]; lp.Score > 0 {
				r.Text(fmt.Sprintf("best %d", lp.Score), b.X+b.W-14, b.Y+34, sizeSmall, sub, engine.AlignRight)
			}
		}
	}
	ex := AddButton(s, 56, 462, 420, 64, fmt.Sprintf("Expedition · %d missions", len(ExpeditionLevels)), func() {
		g.Load(SceneBriefing, NewExpedition())
	})
	ex.Icon = IconRover
	ex.Primary = true
	// Shape school: short lessons on how a rover should be arranged
	sl := "Shape school"
	if n := SchoolDone(g.Host); n > 0 && n < len(Lessons) {
		sl = fmt.Sprintf("Shape school %d/%d", n, len(Lessons))
	}
	sc := AddButton(s, 904-196, 44, 196, 52, sl, // clear of the host page's Sign in button
		func() { g.Load(SceneBuild, LessonSession(NextLesson(g.Host))) })
	sc.Icon = IconRover
	st := AddButton(s, ScreenW-84, 470, 60, 60, "Style sheet", func() { g.Load(SceneStyle, nil) })
	st.Icon, st.HideLabel = IconGear, true
	return s
}
