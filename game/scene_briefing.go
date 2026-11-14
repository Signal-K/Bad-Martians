package game

import (
	"fmt"

	"github.com/signal-k/haphazard-martians/engine"
)

// briefingScene shows the budget and route for the three-mission expedition.
func briefingScene(g *engine.Game, arg any) *engine.Scene {
	e := arg.(*Expedition)
	s := newScene()
	clock := 0.0
	addBackdrop(s)
	AddDraw(s, 2, func(_ engine.Camera, r engine.Renderer) {
		r.Text("Expedition briefing", 28, 16, sizeTitle, colInk, engine.AlignLeft)
		r.Text(fmt.Sprintf("Stage %d of %d · one budget. Progress saves after each attempt.", e.Index+1, len(ExpeditionLevels)),
			28, 54, sizeBody, colMuted, engine.AlignLeft)

		drawPanel(r, 28, 116, 500, 330, "Your budget")
		r.Text(fmt.Sprintf("%d credits", e.Credits), 52, 158, 38, colNeon, engine.AlignLeft)
		for i, ln := range []string{
			"Build and modify your rover with these credits.",
			"Surviving parts can be reused or swapped.",
			"Lost rovers cost their full value.",
			"Finish under par for +2 credits next stage.",
			"Finish the expedition to earn a free retry.",
		} {
			r.Text(ln, 52, 230+float64(i)*32, sizeSmall, colMuted, engine.AlignLeft)
		}

		drawPanel(r, 552, 116, 380, 330, "The route")
		for i, id := range ExpeditionLevels {
			l := GetLevel(id)
			y := 152 + float64(i)*50
			c := engine.V(578, y+13)
			r.Circle(c, 13, colNeon, i == e.Index)
			r.Circle(c, 13, colNeon, false)
			if i+1 < len(ExpeditionLevels) {
				r.Line(c.Add(engine.V(0, 14)), c.Add(engine.V(0, 36)), 2, colMuted)
			}
			r.Text(fmt.Sprintf("Mission %d · %s", l.ID, l.Name), 604, y, sizeBody, colInk, engine.AlignLeft)
			r.Text(fmt.Sprintf("%.0f m", l.Length), 912, y+2, sizeSmall, colMuted, engine.AlignRight)
		}
		for i, ln := range []string{
			"A finished mission keeps your rover.",
			"A crash, or scrapping a stuck rover,",
			"costs the rover’s full value.",
			"No credits for a basic rover: it is over.",
		} {
			r.Text(ln, 576, 316+float64(i)*24, sizeSmall, colMuted, engine.AlignLeft)
		}
	})
	addBack(s, func() { g.Load(SceneMenu, nil) })
	start := AddButton(s, ScreenW-284, 470, 260, 60, fmt.Sprintf("Start Mission %d", ExpeditionLevels[e.Index]), func() {
		SaveExpedition(g.Host, e)
		g.Load(SceneBuild, ExpeditionSession(g.Host, e))
	})
	start.Primary, start.Icon = true, IconRight
	s.OnUpdate = func(dt float64) { clock += dt }
	return s
}

func clamp01(t float64) float64 { return max(0, min(1, t)) }
