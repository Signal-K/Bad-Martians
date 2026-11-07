package game

import (
	"fmt"

	"github.com/signal-k/haphazard-martians/engine"
)

func menuScene(g *engine.Game, _ any) *engine.Scene {
	s := newScene()
	prog := LoadProgress(g.Host)
	addBackdrop(s)
	AddDraw(s, 2, func(_ engine.Camera, r engine.Renderer) {
		r.Text("HAPHAZARD MARTIANS", ScreenW/2, 26, 52, colInk, engine.AlignCenter)
		r.Text("Build a rover. Cross Mars. Help us read its terrain.", ScreenW/2, 86, sizeBody, colMuted, engine.AlignCenter)
		drawPanel(r, 56, 150, 848, 290, "Missions")
		total := 0
		for _, lp := range prog.Levels {
			total += lp.Score
		}
		drawChip(r, 56, 468, IconStarIco, fmt.Sprintf("%d total score", total), colChip)
		r.Text("Finish a mission to unlock the next.", ScreenW-100, 474, sizeSmall, colMuted, engine.AlignRight)
	})
	const tw, th, gap = 150.0, 232.0, 14.0
	x0 := 56 + (848-(tw*5+gap*4))/2
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
	st := AddButton(s, ScreenW-84, 470, 60, 60, "Style sheet", func() { g.Load(SceneStyle, nil) })
	st.Icon, st.HideLabel = IconGear, true
	return s
}
