package game

import (
	"fmt"

	"github.com/signal-k/haphazard-martians/engine"
)

// ReviewArg opens the review scene. Return names the scene to go back to (with
// ReturnArg) when the player is done.
type ReviewArg struct {
	Session   *Session
	Return    string
	ReturnArg any
}

// reviewScene is the AI4Mars activity. In the story, Mission Control wants a
// second opinion on the accuracy of the rover's terrain read. In practice the
// player labels what the ground is made of, and earns build credits for it.
func reviewScene(g *engine.Game, arg any) *engine.Scene {
	ra := arg.(*ReviewArg)
	ses := ra.Session
	lvl := ses.Level
	s := newScene()
	sel := 0
	n := len(lvl.Segments)
	const pw, ph, pgap, py = 106.0, 140.0, 6.0, 128.0
	px0 := (ScreenW - (pw*float64(n) + pgap*float64(n-1))) / 2

	addBackdrop(s)
	AddDraw(s, 2, func(_ engine.Camera, r engine.Renderer) {
		r.Text("Review the accuracy", 28, 16, sizeTitle, colInk, engine.AlignLeft)
		r.Text(fmt.Sprintf("Mission Control wants a second opinion on the navcam's terrain read for Mission %d.", lvl.ID),
			28, 54, sizeBody, colMuted, engine.AlignLeft)
		drawPanel(r, 20, 116, ScreenW-40, 164, "Navcam strip")
		drawChip(r, 28, 288, IconCredit, fmt.Sprintf("+%d credits", ses.Bonus()), colChip)
		drawChip(r, 28+textW(fmt.Sprintf("+%d credits", ses.Bonus()), sizeBody)+60, 288, IconReview,
			fmt.Sprintf("%d/%d reviewed", len(ses.Labels), n), colChipLt)
	})
	for i := 0; i < n; i++ {
		i := i
		b := AddButton(s, px0+float64(i)*(pw+pgap), py, pw, ph, fmt.Sprintf("Panel %d", i+1), func() { sel = i })
		b.Custom = func(b *Button, r engine.Renderer) {
			paintTerrainSample(r, lvl, i, b.X, b.Y, b.W, b.H)
			if c, ok := ses.Labels[i]; ok {
				r.Rect(b.X, b.Y+b.H-28, b.W, 28, colTile.Alpha(225), true)
				r.Text(c.Info().Name, b.X+b.W/2, b.Y+b.H-24, 15, colInk, engine.AlignCenter)
			} else {
				r.Rect(b.X+b.W/2-14, b.Y+b.H-34, 28, 28, colTile.Alpha(200), true)
				r.Text("?", b.X+b.W/2, b.Y+b.H-34, 22, colInk, engine.AlignCenter)
			}
			if i == sel {
				outline(r, b.X-4, b.Y-4, b.W+8, b.H+8, colNeon)
			}
		}
	}
	cw := (ScreenW - 40 - 3*10) / 4
	for c := Class(0); c < NumClasses; c++ {
		c := c
		b := AddButton(s, 20+float64(c)*(cw+10), 336, cw, 64, Classes[c].Name, func() {
			ses.Labels[sel] = c
			for k := 1; k <= n; k++ { // jump to the next unreviewed panel
				j := (sel + k) % n
				if _, done := ses.Labels[j]; !done {
					sel = j
					break
				}
			}
		})
		b.Custom = func(b *Button, r engine.Renderer) {
			fill, ink, sub := colTile, colInk, colMuted
			if b.Held {
				fill, ink, sub = colInk, colTile, colTile
			}
			r.Rect(b.X, b.Y, b.W, b.H, fill, true)
			outline(r, b.X, b.Y, b.W, b.H, colInk)
			r.Rect(b.X+3, b.Y+3, 10, b.H-6, Classes[c].Color, true)
			r.Text(Classes[c].Name, b.X+26, b.Y+8, 19, ink, engine.AlignLeft)
			r.Text(Classes[c].Hint, b.X+26, b.Y+36, 12, sub, engine.AlignLeft)
			if cur, ok := ses.Labels[sel]; ok && cur == c {
				outline(r, b.X-3, b.Y-3, b.W+6, b.H+6, colNeon)
			}
		}
	}
	AddDraw(s, 5, func(_ engine.Camera, r engine.Renderer) {
		r.Text("Each review earns +1 build credit. Some panels are calibration images: get those right for +2 more.",
			20, 414, sizeSmall, colMuted, engine.AlignLeft)
	})
	addBack(s, func() { g.Load(ra.Return, ra.ReturnArg) })
	done := AddButton(s, ScreenW-264, 470, 240, 60, "Send review", func() {
		ses.SubmitLabels(g.Host)
		g.Load(ra.Return, ra.ReturnArg)
	})
	done.Primary, done.Icon = true, IconReview
	s.OnUpdate = func(float64) { done.Disabled = len(ses.Labels) == 0 }
	return s
}
