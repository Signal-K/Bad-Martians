package game

import (
	"fmt"

	"github.com/signal-k/haphazard-martians/engine"
)

// briefingScene opens an expedition: it hands the player the one kit of parts
// that has to last all three missions, and shows the route ahead.
func briefingScene(g *engine.Game, arg any) *engine.Scene {
	e := arg.(*Expedition)
	s := newScene()
	clock := 0.0
	addBackdrop(s)
	AddDraw(s, 2, func(_ engine.Camera, r engine.Renderer) {
		r.Text("Expedition briefing", 28, 16, sizeTitle, colInk, engine.AlignLeft)
		r.Text(fmt.Sprintf("%d missions, one kit. Make these parts last the whole way.", len(ExpeditionLevels)),
			28, 54, sizeBody, colMuted, engine.AlignLeft)

		drawPanel(r, 28, 116, 500, 330, "Your kit")
		for i, k := range []Kind{KFrame, KWheel, KMotor, KMast} {
			// rows arrive one after another, like crates being unloaded
			t := clamp01((clock - 0.15*float64(i)) / 0.35)
			if t <= 0 {
				continue
			}
			y := 150 + float64(i)*72
			x := 28 + 30*(1-t)
			a := uint8(255 * t)
			r.Rect(x+20, y, 460, 62, colPanel.Alpha(a), true)
			if k == KMast {
				drawPart(r, k, engine.V(x+62, y+42), 30, 0)
			} else {
				drawPart(r, k, engine.V(x+62, y+31), 40, 0)
			}
			n := e.Inv[k]
			r.Text(fmt.Sprint(n), x+118, y+12, 34, colNeon.Alpha(a), engine.AlignLeft)
			p, _ := PartOf(k)
			name := p.Name + "s"
			r.Text(name, x+186, y+19, 22, colInk.Alpha(a), engine.AlignLeft)
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
			"loses every part on it.",
			"No motor wheel or mast left: it is over.",
		} {
			r.Text(ln, 576, 316+float64(i)*24, sizeSmall, colMuted, engine.AlignLeft)
		}
	})
	addBack(s, func() { g.Load(SceneMenu, nil) })
	start := AddButton(s, ScreenW-284, 470, 260, 60, fmt.Sprintf("Start Mission %d", ExpeditionLevels[e.Index]), func() {
		g.Load(SceneBuild, ExpeditionSession(g.Host, e))
	})
	start.Primary, start.Icon = true, IconRight
	s.OnUpdate = func(dt float64) { clock += dt }
	return s
}

func clamp01(t float64) float64 { return max(0, min(1, t)) }
