package game

import (
	"fmt"
	"math"

	"github.com/signal-k/haphazard-martians/engine"
)

// travelScene is a skippable, windowed lander hop between expedition sites.
// The same expedition continues into the bay; travel never changes its budget.
func travelScene(g *engine.Game, arg any) *engine.Scene {
	e := arg.(*Expedition)
	s := newScene()
	clock := 0.0
	addBackdrop(s)
	addTitle(s, "Travelling to the next site", fmt.Sprintf("Expedition · stage %d of %d", e.Index+1, len(ExpeditionLevels)))
	AddDraw(s, 2, func(_ engine.Camera, r engine.Renderer) {
		drawPanel(r, 28, 108, 904, 334, "Lander hop")
		previous, next := GetLevel(ExpeditionLevels[e.Index-1]), e.Level()
		r.Text(previous.Name, 70, 155, sizeBody, colMuted, engine.AlignLeft)
		r.Text(next.Name, 890, 155, sizeBody, colInk, engine.AlignRight)
		r.Line(engine.V(100, 340), engine.V(860, 340), 3, colMuted)
		for i := 0; i < len(ExpeditionLevels); i++ {
			x := 100 + float64(i)*760/float64(len(ExpeditionLevels)-1)
			r.Circle(engine.V(x, 340), 8, colNeon, i <= e.Index)
		}
		p := math.Min(clock/3, 1)
		x := 210 + 540*p
		y := 270 - 70*math.Sin(math.Pi*p)
		// A simple lander silhouette makes the hop readable even on a phone.
		r.Rect(x-25, y-18, 50, 30, colNeon, true)
		r.Line(engine.V(x-20, y+12), engine.V(x-33, y+29), 4, colInk)
		r.Line(engine.V(x+20, y+12), engine.V(x+33, y+29), 4, colInk)
		r.Text(fmt.Sprintf("%d credits available · rover carried forward", e.Credits), 480, 385, sizeSmall, colMuted, engine.AlignCenter)
	})
	s.OnUpdate = func(dt float64) { clock += dt }
	continueButton := AddButton(s, 660, 470, 276, 60, "Continue to rover bay", func() {
		g.Load(SceneBuild, ExpeditionSession(g.Host, e))
	})
	continueButton.Primary, continueButton.Icon = true, IconRight
	return s
}
