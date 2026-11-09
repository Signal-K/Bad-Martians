package game

import (
	"fmt"
	"math"

	"github.com/signal-k/haphazard-martians/engine"
	"github.com/signal-k/haphazard-martians/physics"
)

// Rover bay geometry (screen pixels). The grid stands in the world: one cell is
// one metre, drawn at bCell pixels, and its bottom edge is the mission's real
// ground at the start line, so parts look exactly as they will on the drive.
const (
	bCell    = 52.0
	bGX      = (ScreenW - bCell*GridW) / 2
	bGround  = 360.0 // screen y of the ground under the grid
	bGY      = bGround - bCell*GridH
	bTrayY   = 420.0 // top of the parts tray
	bTileW   = 100.0
	bTileH   = 100.0
	bTileGap = 10.0
	bSwipe   = 40.0 // horizontal travel on the tray that counts as a swipe
)

// bayTools is the order of the parts tray; "" is the eraser.
var bayTools = []Kind{KFrame, KWheel, KMotor, KMast, ""}

// Specs summarises a design; each is 0..10 segments.
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

// CanAdd is how many more of part k fit in the budget: the number on its tray
// tile. The mast is 1 until one is placed (placing another moves it).
func CanAdd(d Design, k Kind, budget int) int {
	p, ok := PartOf(k)
	if !ok {
		return 0
	}
	if k == KMast {
		for _, c := range d.Cells {
			if c.K == KMast {
				return 0
			}
		}
		if budget-d.Cost() >= p.Cost {
			return 1
		}
		return 0
	}
	return max(0, (budget-d.Cost())/p.Cost)
}

// Pose is how a settled rover sits compared with its blueprint.
type Pose struct {
	Tilt float64 // radians, positive = nose (right side) up
	Bend float64 // RMS metres of shape change after removing the tilt
}

// PoseOf fits the best rotation from the blueprint to the bodies (2D
// Procrustes): the rotation is the tilt, what is left over is bending.
func PoseOf(r *Rover) Pose {
	n := float64(len(r.Bodies))
	if n < 2 {
		return Pose{}
	}
	var dc, pc engine.Vec2
	for i, c := range r.Cells {
		dc = dc.Add(engine.V(float64(c.X), float64(c.Y)))
		pc = pc.Add(r.Bodies[i].Pos)
	}
	dc, pc = dc.Mul(1/n), pc.Mul(1/n)
	var sc, ss float64
	for i, c := range r.Cells {
		d := engine.V(float64(c.X), float64(c.Y)).Sub(dc)
		p := r.Bodies[i].Pos.Sub(pc)
		ss += d.X*p.Y - d.Y*p.X
		sc += d.X*p.X + d.Y*p.Y
	}
	th := math.Atan2(ss, sc)
	var res float64
	for i, c := range r.Cells {
		d := engine.V(float64(c.X), float64(c.Y)).Sub(dc)
		rot := engine.V(d.X*math.Cos(th)-d.Y*math.Sin(th), d.X*math.Sin(th)+d.Y*math.Cos(th))
		e := r.Bodies[i].Pos.Sub(pc).Sub(rot)
		res += e.X*e.X + e.Y*e.Y
	}
	return Pose{Tilt: th, Bend: math.Sqrt(res / n)}
}

// bendLimit is the RMS shape change above which the bay says the frame bends.
const bendLimit = 0.04

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
	clock := 0.0
	placedAt := map[[2]int]float64{} // snap animation start, per cell

	// tray gesture state
	var press engine.Vec2
	trayDown, swiped, dragging := false, false, false
	var dragKind Kind = "-"

	// test drop: live physics on the real start terrain
	testing := false
	var testWorld *physics.World
	var testRover *Rover
	testT := 0.0

	// One camera for the grid, the ground and the test drop: grid column 0,
	// row 0 sits on the terrain at the run's spawn point.
	originX := lvl.StartX + 2
	h0 := lvl.Terrain.HeightAt(originX)
	cam := engine.Camera{Zoom: bCell, Screen: engine.V(ScreenW, ScreenH)}
	cam.Pos = engine.V(originX+(ScreenW/2-(bGX+bCell/2))/bCell, h0+(bGround-ScreenH/2)/bCell)

	cellXY := func(cx, cy int) (float64, float64) { // top-left of a cell
		return bGX + float64(cx)*bCell, bGY + float64(GridH-1-cy)*bCell
	}
	cellAt := func(p engine.Vec2) (int, int, bool) {
		if p.X < bGX || p.Y < bGY || p.X >= bGX+bCell*GridW || p.Y >= bGY+bCell*GridH {
			return 0, 0, false
		}
		return int((p.X - bGX) / bCell), GridH - 1 - int((p.Y-bGY)/bCell), true
	}
	centre := func(cx, cy int) engine.Vec2 {
		x, y := cellXY(cx, cy)
		return engine.V(x+bCell/2, y+bCell/2)
	}
	toolIndex := func(k Kind) int {
		for i, t := range bayTools {
			if t == k {
				return i
			}
		}
		return 0
	}
	tileX := func(i int) float64 {
		w := float64(len(bayTools))*bTileW + float64(len(bayTools)-1)*bTileGap
		return (ScreenW-w)/2 + float64(i)*(bTileW+bTileGap)
	}
	tileAt := func(p engine.Vec2) (Kind, bool) {
		for i, k := range bayTools {
			x := tileX(i)
			if p.X >= x && p.X <= x+bTileW && p.Y >= bTrayY+8 && p.Y <= bTrayY+8+bTileH {
				return k, true
			}
		}
		return "", false
	}
	place := func(cx, cy int, k Kind) {
		before, had := ses.Design.At(cx, cy)
		ses.Design.Set(cx, cy, k)
		if k != "" && (!had || before != k) {
			placedAt[[2]int{cx, cy}] = clock
		}
	}
	stopTest := func() { testing, testWorld, testRover = false, nil, nil }
	var testBtn *Button
	startTest := func() {
		if len(ses.Design.Cells) == 0 {
			return
		}
		testWorld = physics.NewWorld()
		testWorld.Terrain = lvl.Terrain
		testRover = SpawnRover(testWorld, ses.Design, originX, h0)
		testing, testT = true, 0
	}

	// sky and ground: the same art as the drive
	AddDraw(s, 0, func(_ engine.Camera, r engine.Renderer) {
		drawSky(r, 0)
		drawTerrain(r, cam, lvl.Terrain)
	})

	// the grid, the parts, and the test drop
	AddDraw(s, 2, func(_ engine.Camera, r engine.Renderer) {
		if testing {
			drawTestDrop(r, cam, testRover, testT)
			return
		}
		for cy := 0; cy < GridH; cy++ { // sockets
			for cx := 0; cx < GridW; cx++ {
				x, y := cellXY(cx, cy)
				r.Rect(x+3, y+3, bCell-6, bCell-6, colSpace.Alpha(46), true)
				r.Rect(x+3, y+3, bCell-6, bCell-6, colInk.Alpha(34), false)
				c := centre(cx, cy)
				r.Circle(c, 2.5, colInk.Alpha(70), true)
			}
		}
		// hover: where a part would go, and the beams it would snap to
		hover, hk := [2]int{-1, -1}, tool
		if dragging {
			hk = dragKind
		}
		if cx, cy, ok := cellAt(g.In.Pointer); ok && (dragging || !trayDown) && hk != "-" {
			hover = [2]int{cx, cy}
			x, y := cellXY(cx, cy)
			hc := colNeon
			if hk == "" {
				hc = colWarn
			}
			outline(r, x+1, y+1, bCell-2, bCell-2, hc.Alpha(200))
			if hk != "" {
				for _, o := range [][2]int{{1, 0}, {0, 1}, {1, 1}, {-1, 1}, {-1, 0}, {0, -1}, {-1, -1}, {1, -1}} {
					if _, ok := ses.Design.At(cx+o[0], cy+o[1]); ok {
						r.Line(centre(cx, cy), centre(cx+o[0], cy+o[1]), 4, colNeon.Alpha(170))
					}
				}
			}
		}
		for _, c := range ses.Design.Cells { // beams first, parts on top
			for _, o := range [][2]int{{1, 0}, {0, 1}, {1, 1}, {-1, 1}} {
				if _, ok := ses.Design.At(c.X+o[0], c.Y+o[1]); ok {
					drawBeam(r, centre(c.X, c.Y), centre(c.X+o[0], c.Y+o[1]), bCell)
				}
			}
		}
		for _, c := range ses.Design.Cells {
			z := bCell
			if t0, ok := placedAt[[2]int{c.X, c.Y}]; ok {
				if a := (clock - t0) / 0.22; a < 1 { // snap: pop in, then settle
					z *= 1 + 0.28*math.Sin(a*math.Pi)
					r.Circle(centre(c.X, c.Y), bCell*(0.4+0.5*a), colNeon.Alpha(uint8(160*(1-a))), false)
				}
			}
			drawPart(r, c.K, centre(c.X, c.Y), z, 0)
		}
		if hover[0] >= 0 && hk != "" && hk != "-" {
			if _, ok := ses.Design.At(hover[0], hover[1]); !ok { // ghost of the part
				c := centre(hover[0], hover[1])
				r.Circle(c, bCell*0.36, colInk.Alpha(60), true)
				drawIcon(r, partIcon(hk), c.X, c.Y, bCell*0.5, colInk.Alpha(170))
			}
		}
	})

	// chrome: top bar, terrain map, tray backing, specs
	AddDraw(s, 9, func(_ engine.Camera, r engine.Renderer) {
		r.Rect(0, 0, ScreenW, 64, colTile.Alpha(215), true)
		r.Rect(0, 63, ScreenW, 2, colInk.Alpha(60), true)
		r.Text(fmt.Sprintf("Rover bay · Mission %d", lvl.ID), 92, 8, 22, colInk, engine.AlignLeft)
		switch {
		case testing:
			r.Text("Test drop: this is how it sits on the start line.", 92, 38, sizeSmall, colChipLt, engine.AlignLeft)
		case len(errs) > 0:
			r.Text(errs[0], 92, 38, sizeSmall, colWarn, engine.AlignLeft)
		case len(ses.Design.Cells) == 0:
			r.Text("Tap, swipe or drag a part from the tray onto the grid.", 92, 38, sizeSmall, colChipLt, engine.AlignLeft)
		default:
			r.Text("Ready. Test to see how it sits, or GO to drive.", 92, 38, sizeSmall, colGood, engine.AlignLeft)
		}
		used, budget := ses.Design.Cost(), ses.Budget()
		col := colChip
		if used > budget {
			col = colWarn
		}
		txt := fmt.Sprintf("%d / %d", used, budget)
		w := textW(txt, sizeBody) + 46
		drawChip(r, testBtn.X-14-w, 14, IconCredit, txt, col)

		// terrain map: the whole course, coloured by the player's own review
		const sx, sy, sw, sh = bGX, 376.0, bCell * GridW, 34.0
		r.Rect(sx-8, sy-6, sw+16, sh+12, colTile.Alpha(200), true)
		outline(r, sx-8, sy-6, sw+16, sh+12, colInk.Alpha(120))
		r.Text("Course", sx-18, sy+8, sizeSmall, colInk, engine.AlignRight)
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
		// the bay's own stretch of ground, then the finish
		bx0, bx1 := pt(originX, 0).X, pt(originX+GridW, 0).X
		r.Rect(bx0, sy, math.Max(3, bx1-bx0), sh, colNeon.Alpha(70), true)
		gx := pt(lvl.GoalX, 0).X
		r.Line(engine.V(gx, sy), engine.V(gx, sy+sh), 2, colInk)

		// tray backing and specs
		r.Rect(0, bTrayY, ScreenW, ScreenH-bTrayY, colTile.Alpha(225), true)
		r.Rect(0, bTrayY, ScreenW, 2, colInk.Alpha(60), true)
		sp := SpecsOf(ses.Design)
		for i, row := range []struct {
			n string
			v int
		}{{"Power", sp.Power}, {"Grip", sp.Traction}, {"Mass", sp.Mass}} {
			y := bTrayY + 22 + float64(i)*30
			r.Text(row.n, 20, y, sizeSmall, colInk, engine.AlignLeft)
			drawMeter(r, 82, y+2, 82, 10, row.v, colNeon)
		}
		// a part being dragged from the tray follows the pointer
		if dragging && dragKind != "-" {
			p := g.In.Pointer
			if dragKind == "" {
				drawIcon(r, IconErase, p.X, p.Y, 34, colWarn)
			} else {
				drawPart(r, dragKind, p, bCell, 0)
			}
		}
	})

	// parts tray: one tile per part, with what it looks like on the rover,
	// its cost and how many more you can add
	for i, k := range bayTools {
		k := k
		name := "Erase"
		if k != "" {
			p, _ := PartOf(k)
			name = p.Name
		}
		b := AddButton(s, tileX(i), bTrayY+8, bTileW, bTileH, name, func() {
			if !swiped {
				tool = k
				stopTest()
			}
		})
		b.Custom = func(b *Button, r engine.Renderer) {
			sel := tool == k
			n := CanAdd(ses.Design, k, ses.Budget())
			fill, line := colPanel, colInk.Alpha(150)
			if sel {
				fill, line = colPanel2, colNeon
			}
			if b.Held {
				fill = colInk.Alpha(40)
			}
			r.Rect(b.X, b.Y, b.W, b.H, fill, true)
			outline(r, b.X, b.Y, b.W, b.H, line)
			if sel {
				outline(r, b.X-3, b.Y-3, b.W+6, b.H+6, colNeon.Alpha(110))
			}
			cx := b.X + b.W/2
			label, cost := "Erase", "free"
			if k == "" {
				drawIcon(r, IconErase, cx, b.Y+36, 36, colWarn)
			} else {
				p, _ := PartOf(k)
				drawPart(r, k, engine.V(cx, b.Y+38), 46, 0)
				label, cost = p.Name, fmt.Sprintf("%d cr", p.Cost)
			}
			ink := colInk
			if k != "" && n == 0 && k != KMast {
				ink = colMuted
			}
			r.Text(label, cx, b.Y+66, sizeSmall, ink, engine.AlignCenter)
			r.Text(cost, cx, b.Y+82, 12, colMuted, engine.AlignCenter)
			if k != "" { // quantity badge
				bc := colChip
				if n == 0 {
					bc = colStarOff
				}
				c := engine.V(b.X+b.W-14, b.Y+14)
				r.Circle(c, 13, bc, true)
				r.Circle(c, 13, colInk, false)
				r.Text(fmt.Sprint(n), c.X, c.Y-8, sizeSmall, colInk, engine.AlignCenter)
			}
		}
	}

	back := AddButton(s, 16, 10, 60, 44, "Back", func() { g.Load(SceneMenu, nil) })
	back.Icon, back.HideLabel = IconBack, true
	testBtn = AddButton(s, ScreenW-16-132, 10, 132, 44, "Test", func() {
		if testing {
			stopTest()
		} else {
			startTest()
		}
	})
	testBtn.Custom = func(b *Button, r engine.Renderer) {
		fill, ink := colTile, colInk
		if testing {
			fill, ink = colChipLt, colTile
		}
		if b.Held {
			fill, ink = colInk, colTile
		}
		r.Rect(b.X, b.Y, b.W, b.H, fill, true)
		outline(r, b.X, b.Y, b.W, b.H, colInk)
		label := "Test drop"
		if testing {
			label = "Back to grid"
		}
		r.Text(label, b.X+b.W/2, b.Y+12, sizeSmall+1, ink, engine.AlignCenter)
	}
	launch := AddButton(s, ScreenW-20-170, bTrayY+22, 170, 72, "GO", func() { g.Load(SceneRun, ses) })
	launch.Primary, launch.Icon = true, IconRight

	s.OnUpdate = func(dt float64) {
		clock += dt
		errs = ses.Design.Validate(ses.Budget())
		launch.Disabled = len(errs) > 0
		testBtn.Disabled = len(ses.Design.Cells) == 0
		if testing {
			testWorld.Step(dt)
			testT += dt
		}
		in := g.In
		p := in.Pointer

		// tray gestures: tap selects (the tile button), a sideways swipe steps
		// the selection, dragging up carries a part onto the grid
		if in.Pressed {
			swiped, dragging = false, false
			trayDown = p.Y >= bTrayY
			press = p
			dragKind = "-"
			if k, ok := tileAt(p); ok && trayDown {
				dragKind = k
			}
		}
		if trayDown && in.Down && dragKind != "-" && p.Y < bTrayY-6 {
			dragging = true
		}
		if in.Released && trayDown {
			if dragging {
				if cx, cy, ok := cellAt(p); ok {
					stopTest()
					place(cx, cy, dragKind)
					tool = dragKind
				}
				swiped = true
			} else if dx := p.X - press.X; math.Abs(dx) > bSwipe && math.Abs(p.Y-press.Y) < 70 {
				step := 1
				if dx > 0 {
					step = -1
				}
				tool = bayTools[(toolIndex(tool)+step+len(bayTools))%len(bayTools)]
				stopTest()
				swiped = true
			}
		}
		if !in.Down {
			trayDown, dragging = false, false
		}

		// painting on the grid with the selected part
		cx, cy, inGrid := cellAt(p)
		if in.Pressed && inGrid && !testing {
			painting, lastCell = true, [2]int{-1, -1}
		}
		if !in.Down {
			painting = false
		}
		if painting && inGrid && (cx != lastCell[0] || cy != lastCell[1]) {
			lastCell = [2]int{cx, cy}
			place(cx, cy, tool)
		}
	}
	return s
}

// drawTestDrop draws the dropped rover with its tilt against a level line, and
// says whether it sits level and holds its shape.
func drawTestDrop(r engine.Renderer, cam engine.Camera, rv *Rover, t float64) {
	rv.Draw(nil, cam, r)
	pose := PoseOf(rv)
	c := cam.ToScreen(rv.Center())
	// level reference and the chassis axis through the centre
	const L = 150.0
	r.Line(engine.V(c.X-L, c.Y), engine.V(c.X+L, c.Y), 2, colInk.Alpha(120))
	d := engine.V(math.Cos(pose.Tilt), -math.Sin(pose.Tilt)).Mul(L)
	r.Line(c.Sub(d), c.Add(d), 3, colNeon)
	deg := pose.Tilt * 180 / math.Pi
	tilt := "Sits level"
	if math.Abs(deg) >= 2 {
		dir := "nose up"
		if deg < 0 {
			dir = "nose down"
		}
		tilt = fmt.Sprintf("Tilts %.0f° %s", math.Abs(deg), dir)
	}
	drawPanel(r, 24, 80, 320, 78, "")
	r.Text(tilt, 40, 92, sizeBody, colInk, engine.AlignLeft)
	switch {
	case t < 1.2:
		r.Text("Settling…", 40, 122, sizeSmall, colMuted, engine.AlignLeft)
	case pose.Bend > bendLimit:
		r.Text("Frame bends: add a diagonal brace.", 40, 122, sizeSmall, colWarn, engine.AlignLeft)
	default:
		r.Text("Frame holds its shape.", 40, 122, sizeSmall, colGood, engine.AlignLeft)
	}
}
