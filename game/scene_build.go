package game

import (
	"fmt"
	"math"

	"github.com/signal-k/haphazard-martians/engine"
)

// Build grid geometry (screen pixels). Cargo layout: the board panel holds the
// grid, the rail holds parts and specs.
const (
	bCell = 44.0
	bGX   = 65.0
	bGY   = 115.0
)

// Specs summarises a design for the Specs panel; each is 0..10 segments.
type Specs struct{ Power, Traction, Mass int }

func SpecsOf(d Design) Specs {
	var motors, wheels int
	var mass float64
	for _, c := range d.Cells {
		p, _ := PartOf(c.K)
		mass += p.Mass
		if p.Motor {
			motors++
		} else if p.Wheel {
			wheels++
		}
	}
	cap10 := func(v int) int { return int(math.Min(10, math.Max(0, float64(v)))) }
	return Specs{Power: cap10(motors * 3), Traction: cap10((wheels + motors) * 2), Mass: cap10(int(math.Round(mass / 1.4)))}
}

func buildScene(g *engine.Game, arg any) *engine.Scene {
	ses := arg.(*Session)
	lvl := ses.Level
	s := newScene()
	// Keep the starter rover only if it fits this mission's budget.
	if len(ses.Design.Cells) > 0 && ses.Design.Cost() > ses.Budget() {
		ses.Design = Design{}
	}
	tool := KWheel
	painting := false
	lastCell := [2]int{-1, -1}
	var errs []string

	cellXY := func(cx, cy int) (float64, float64) {
		return bGX + float64(cx)*bCell, bGY + float64(GridH-1-cy)*bCell
	}
	addBackdrop(s)
	AddDraw(s, 1, func(_ engine.Camera, r engine.Renderer) {
		r.Text(fmt.Sprintf("Rover bay · Mission %d", lvl.ID), 28, 14, sizeTitle, colInk, engine.AlignLeft)
		if len(errs) > 0 {
			r.Text(errs[0], 28, 48, sizeBody, colWarn, engine.AlignLeft)
		} else {
			r.Text("Ready. Hold right arrow or the gas tile to drive.", 28, 48, sizeBody, colGood, engine.AlignLeft)
		}
		used, budget := ses.Design.Cost(), ses.Budget()
		col := colChip
		if used > budget {
			col = colWarn
		}
		txt := fmt.Sprintf("%d / %d", used, budget)
		w := textW(txt, sizeBody) + 46
		drawChip(r, ScreenW-24-w, 16, IconCredit, txt, col)

		// board: the chassis grid
		drawPanel(r, 24, 88, 610, 362, "Chassis")
		for cy := 0; cy < GridH; cy++ {
			for cx := 0; cx < GridW; cx++ {
				x, y := cellXY(cx, cy)
				r.Rect(x+2, y+2, bCell-4, bCell-4, colPanel.Alpha(150), true)
				r.Rect(x+2, y+2, bCell-4, bCell-4, colMuted.Alpha(60), false)
			}
		}
		r.Rect(bGX, bGY+bCell*GridH+4, bCell*GridW, 6, Classes[Soil].Color, true)
		for _, c := range ses.Design.Cells { // links first, tiles on top
			for _, o := range [][2]int{{1, 0}, {0, 1}, {1, 1}, {-1, 1}} {
				if _, ok := ses.Design.At(c.X+o[0], c.Y+o[1]); ok {
					ax, ay := cellXY(c.X, c.Y)
					bx, by := cellXY(c.X+o[0], c.Y+o[1])
					r.Line(engine.V(ax+bCell/2, ay+bCell/2), engine.V(bx+bCell/2, by+bCell/2), 5, colInk)
				}
			}
		}
		for _, c := range ses.Design.Cells {
			x, y := cellXY(c.X, c.Y)
			r.Rect(x+3, y+3, bCell-6, bCell-6, colTile, true)
			outline(r, x+3, y+3, bCell-6, bCell-6, colInk)
			drawIcon(r, partIcon(c.K), x+bCell/2, y+bCell/2, bCell*0.62, colInk)
		}

		// rail: parts description and specs
		drawPanel(r, 650, 88, 286, 200, "Parts")
		if tool == "" {
			r.Text("Erase", 666, 256, sizeBody, colInk, engine.AlignLeft)
			r.Text("Tap a part to remove it.", 666, 278-2, sizeSmall, colMuted, engine.AlignLeft)
		} else {
			p, _ := PartOf(tool)
			r.Text(p.Name, 666, 252, sizeBody, colInk, engine.AlignLeft)
			r.Text(fmt.Sprintf("%d cr", p.Cost), 920, 252, sizeBody, colMuted, engine.AlignRight)
		}
		drawPanel(r, 650, 306, 286, 144, "Specs")
		sp := SpecsOf(ses.Design)
		for i, row := range []struct {
			n string
			v int
		}{{"Power", sp.Power}, {"Traction", sp.Traction}, {"Mass", sp.Mass}} {
			y := 326 + float64(i)*38
			r.Text(row.n, 666, y, sizeSmall, colInk, engine.AlignLeft)
			drawMeter(r, 750, y+1, 170, 10, row.v, colNeon)
		}

		// footer: the course, coloured by the player's own review
		const sx, sy, sw, sh = 108.0, 484.0, 504.0, 38.0
		drawPanel(r, 96, 472, 528, 58, "Course")
		x0, x1 := lvl.Terrain.X0, lvl.Terrain.X0+lvl.Terrain.DX*float64(lvl.Terrain.Len()-1)
		lo, hi := math.Inf(1), math.Inf(-1)
		for _, h := range lvl.Terrain.H {
			lo, hi = math.Min(lo, h), math.Max(hi, h)
		}
		hh := math.Max(hi-lo, 2)
		pt := func(x, h float64) engine.Vec2 {
			return engine.V(sx+(x-x0)/(x1-x0)*sw, sy+sh-3-(h-lo)/hh*(sh-12))
		}
		for i, seg := range lvl.Segments {
			col := colStarOff
			if c, ok := ses.Labels[i]; ok {
				col = Classes[c].Color
			}
			poly := []engine.Vec2{engine.V(pt(seg.X0, 0).X, sy+sh)}
			for x := seg.X0; x <= seg.X1; x++ {
				poly = append(poly, pt(x, lvl.Terrain.HeightAt(x)))
			}
			poly = append(poly, pt(seg.X1, lvl.Terrain.HeightAt(seg.X1)), engine.V(pt(seg.X1, 0).X, sy+sh))
			r.Poly(poly, col)
		}
		for _, sm := range lvl.Samples {
			drawStar(r, pt(sm.X, 0).X, sy+6, 5, colStar)
		}
		gx := pt(lvl.GoalX, 0).X
		r.Line(engine.V(gx, sy), engine.V(gx, sy+sh), 2, colInk)
	})

	// parts palette: a 3-wide grid of tiles
	tools := []Kind{KFrame, KWheel, KMotor, KMast, ""}
	const ts, tg = 60.0, 12.0
	tx0 := 650 + (286-(3*ts+2*tg))/2
	for i, k := range tools {
		k := k
		name := "Erase"
		if k != "" {
			p, _ := PartOf(k)
			name = p.Name
		}
		b := AddButton(s, tx0+float64(i%3)*(ts+tg), 108+float64(i/3)*(ts+tg), ts, ts, name, func() { tool = k })
		b.Custom = func(b *Button, r engine.Renderer) {
			st := TileNormal
			if tool == k {
				st = TileSelected
			}
			if b.Held {
				st = TileHeld
			}
			drawTile(r, b.X, b.Y, b.W, partIcon(k), st)
		}
	}
	addBack(s, func() { g.Load(SceneMenu, nil) })
	launch := AddButton(s, 730, 470, 206, 60, "GO", func() { g.Load(SceneRun, ses) })
	launch.Primary, launch.Icon = true, IconRight

	s.OnUpdate = func(float64) {
		errs = ses.Design.Validate(ses.Budget())
		launch.Disabled = len(errs) > 0
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
