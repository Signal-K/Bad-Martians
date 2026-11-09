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

	// Shapes sidebar: ready-made frame blocks, left of the gantry
	bSideX = 8.0
	bSideW = 124.0
	bSideY = 100.0 // top of the first card
	bCardH = 80.0
	bCardG = 6.0
)

// cardY is the top of shapes card i.
func cardY(i int) float64 { return bSideY + float64(i)*(bCardH+bCardG) }

// skipBox is the tutorial's Skip button, in the top bar.
var skipBox = engine.Rect{X: 560, Y: 10, W: 84, H: 44}

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
	blk, dragBlock := -1, -1   // selected / dragged shapes block, -1 for none
	idea, ideaUntil := -1, 0.0 // block whose idea the status line shows, and until when
	tut := ses.Tut
	var skip *engine.Rect // the tutorial's skip button, to keep taps off the grid

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
	cardAt := func(p engine.Vec2) int {
		for i := range Blocks {
			if p.X >= bSideX && p.X <= bSideX+bSideW && p.Y >= cardY(i) && p.Y <= cardY(i)+bCardH {
				return i
			}
		}
		return -1
	}
	stamp := func(i, cx, cy int) {
		x, y := Blocks[i].Anchor(cx, cy)
		for _, c := range Blocks[i].Stamp(&ses.Design, x, y) {
			placedAt[c] = clock
		}
	}
	showIdea := func(i int) { idea, ideaUntil = i, clock+5 }
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
		if tut != nil {
			tut.Tested = true
		}
	}

	// sky and ground: the same art as the drive
	AddDraw(s, 0, func(_ engine.Camera, r engine.Renderer) {
		drawSky(r, 0)
		drawTerrain(r, cam, lvl.Terrain)
		drawGantry(r, cam, lvl.Terrain, testing)
		if !testing { // shadows of the parts on the grid
			var pos []engine.Vec2
			var rad []float64
			for _, c := range ses.Design.Cells {
				p, _ := PartOf(c.K)
				pos = append(pos, cam.ToWorld(centre(c.X, c.Y)))
				rad = append(rad, p.Radius)
			}
			drawShadows(r, cam, lvl.Terrain, pos, rad)
		}
	})

	// the grid, the parts, and the test drop
	AddDraw(s, 2, func(_ engine.Camera, r engine.Renderer) {
		if testing {
			drawTestDrop(r, cam, lvl.Terrain, testRover)
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
		hb := blk // a block takes the place of the part tool
		if dragBlock >= 0 {
			hb = dragBlock
		} else if dragging {
			hb = -1
		}
		if cx, cy, ok := cellAt(g.In.Pointer); ok && hb >= 0 && (dragBlock >= 0 || !trayDown) {
			x0, y0 := Blocks[hb].Anchor(cx, cy)
			for _, c := range Blocks[hb].Cells {
				x, y := cellXY(x0+c.X, y0+c.Y)
				outline(r, x+1, y+1, bCell-2, bCell-2, colNeon.Alpha(200))
				ctr := centre(x0+c.X, y0+c.Y)
				r.Circle(ctr, bCell*0.36, colInk.Alpha(60), true)
				drawIcon(r, partIcon(c.K), ctr.X, ctr.Y, bCell*0.5, colInk.Alpha(170))
			}
			hk = "-"
		} else if hb >= 0 {
			hk = "-"
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
		if tut != nil && !tut.Done { // where the tutorial wants the next part
			a := uint8(120 + 135*(0.5+0.5*math.Sin(clock*5)))
			for _, c := range tut.hintCells(ses.Design, tut.BuildStep(ses.Design)) {
				x, y := cellXY(c[0], c[1])
				r.Rect(x+3, y+3, bCell-6, bCell-6, colNeon.Alpha(a/3), true)
				outline(r, x+3, y+3, bCell-6, bCell-6, colNeon.Alpha(a))
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
			msg, col := testVerdict(testRover, testT)
			r.Text(msg, 92, 38, sizeSmall+1, col, engine.AlignLeft)
		case tut != nil && !tut.Done: // the tutorial speaks in the status line
			msg := tutBuildText[tut.BuildStep(ses.Design)]
			if tut.BuildStep(ses.Design) >= tutTest && len(errs) > 0 {
				msg = "Almost: " + errs[0] + "."
			}
			r.Text(msg, 92, 38, sizeSmall+1, colNeon, engine.AlignLeft)
		case idea >= 0 && clock < ideaUntil:
			r.Text(Blocks[idea].Idea, 92, 38, sizeSmall, colChipLt, engine.AlignLeft)
		case len(ses.Design.Cells) == 0:
			r.Text("Drag a shape or a part onto the grid.", 92, 38, sizeSmall, colChipLt, engine.AlignLeft)
		case len(errs) > 0:
			r.Text(errs[0], 92, 38, sizeSmall, colWarn, engine.AlignLeft)
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
		// shapes sidebar backing
		sideB := cardY(len(Blocks)) + 2 // stop above the course map's label
		r.Rect(0, 64, bSideX+bSideW+8, sideB-64, colTile.Alpha(205), true)
		r.Rect(bSideX+bSideW+7, 64, 2, sideB-64, colInk.Alpha(60), true)
		r.Text("Shapes", bSideX+bSideW/2, 74, sizeSmall+1, colNeon, engine.AlignCenter)
		// a block being dragged from the sidebar follows the pointer
		if dragBlock >= 0 && dragging {
			drawBlockMini(r, Blocks[dragBlock], g.In.Pointer, bCell*0.6)
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
				tool, blk = k, -1
				stopTest()
			}
		})
		b.Custom = func(b *Button, r engine.Renderer) {
			sel := tool == k && blk < 0
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
				if k == KMast { // the camera head stands tall: sit it lower and smaller
					drawPart(r, k, engine.V(cx, b.Y+50), 34, 0)
				} else {
					drawPart(r, k, engine.V(cx, b.Y+38), 46, 0)
				}
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

	// shapes cards: ready-made frame blocks with the idea each one teaches
	for i, bl := range Blocks {
		i, bl := i, bl
		b := AddButton(s, bSideX, cardY(i), bSideW, bCardH, bl.Name, func() {
			if !swiped {
				blk = i // tap: the block becomes the tool, stamped by tapping the grid
				showIdea(i)
				stopTest()
			}
		})
		b.Custom = func(b *Button, r engine.Renderer) {
			sel := blk == i
			fill, line := colPanel, colInk.Alpha(150)
			if sel {
				fill, line = colPanel2, colNeon
			}
			if b.Held {
				fill = colInk.Alpha(40)
			}
			r.Rect(b.X, b.Y, b.W, b.H, fill, true)
			outline(r, b.X, b.Y, b.W, b.H, line)
			drawBlockMini(r, bl, engine.V(b.X+b.W/2, b.Y+28), 16)
			r.Text(bl.Name, b.X+b.W/2, b.Y+50, sizeSmall, colInk, engine.AlignCenter)
			r.Text(fmt.Sprintf("%d cr", bl.Cost()), b.X+b.W/2, b.Y+66, 12, colMuted, engine.AlignCenter)
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

	// tutorial: the status line says what to do next; here it only points at it
	if tut != nil {
		sk := skipBox
		skip = &sk
		sb := AddButton(s, sk.X, sk.Y, sk.W, sk.H, "Skip", func() {
			if !tut.Done {
				FinishTutorial(g.Host, tut)
			}
		})
		sb.Custom = func(b *Button, r engine.Renderer) {
			if tut.Done {
				return
			}
			fill := colTile
			if b.Held {
				fill = colInk.Alpha(60)
			}
			r.Rect(b.X, b.Y, b.W, b.H, fill, true)
			outline(r, b.X, b.Y, b.W, b.H, colInk.Alpha(120))
			r.Text("Skip", b.X+b.W/2, b.Y+13, sizeSmall, colMuted, engine.AlignCenter)
		}
		tray := func(k Kind) engine.Rect {
			return engine.Rect{X: tileX(toolIndex(k)), Y: bTrayY + 8, W: bTileW, H: bTileH}
		}
		beamCard := 0
		for i, bl := range Blocks {
			if bl.Name == "Beam" {
				beamCard = i
			}
		}
		o := s.Spawn("tutorial")
		o.Layer = 30
		o.Add(DrawFunc(func(_ engine.Camera, r engine.Renderer) {
			if tut.Done || dragging {
				return
			}
			var target engine.Rect
			switch tut.BuildStep(ses.Design) {
			case tutBeam:
				target = engine.Rect{X: bSideX, Y: cardY(beamCard), W: bSideW, H: bCardH}
			case tutWheel:
				target = tray(KWheel)
			case tutMotor:
				target = tray(KMotor)
			case tutMast:
				target = tray(KMast)
			case tutTest:
				target = engine.Rect{X: testBtn.X, Y: testBtn.Y, W: testBtn.W, H: testBtn.H}
			case tutGo:
				target = engine.Rect{X: launch.X, Y: launch.Y, W: launch.W, H: launch.H}
			}
			if len(errs) > 0 && tut.BuildStep(ses.Design) >= tutTest {
				target = engine.Rect{}
			}
			drawPointer(r, clock, target)
		}))
	}

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
			dragKind, dragBlock = "-", -1
			if k, ok := tileAt(p); ok && trayDown {
				dragKind = k
			}
			if i := cardAt(p); i >= 0 {
				dragBlock = i
			}
		}
		if trayDown && in.Down && dragKind != "-" && p.Y < bTrayY-6 {
			dragging = true
		}
		if dragBlock >= 0 && in.Down && p.X > bSideX+bSideW+10 {
			dragging = true
		}
		if in.Released && dragBlock >= 0 {
			if dragging {
				if cx, cy, ok := cellAt(p); ok {
					stopTest()
					stamp(dragBlock, cx, cy) // the part tool stays as it was
					showIdea(dragBlock)
				}
				swiped = true
			}
		}
		if in.Released && trayDown {
			if dragging {
				if cx, cy, ok := cellAt(p); ok {
					stopTest()
					place(cx, cy, dragKind)
					tool, blk = dragKind, -1
				}
				swiped = true
			} else if dx := p.X - press.X; math.Abs(dx) > bSwipe && math.Abs(p.Y-press.Y) < 70 {
				step := 1
				if dx > 0 {
					step = -1
				}
				tool, blk = bayTools[(toolIndex(tool)+step+len(bayTools))%len(bayTools)], -1
				stopTest()
				swiped = true
			}
		}
		if !in.Down {
			trayDown, dragging, dragBlock = false, false, -1
		}

		// painting on the grid with the selected part; a block stamps once
		cx, cy, inGrid := cellAt(p)
		onSkip := skip != nil && !tut.Done && p.X >= skip.X && p.X <= skip.X+skip.W && p.Y >= skip.Y && p.Y <= skip.Y+skip.H
		if in.Pressed && inGrid && !testing && !onSkip {
			if blk >= 0 {
				stamp(blk, cx, cy)
				return
			}
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

// drawTestDrop draws the dropped rover, with its chassis axis against a level
// line only when it tilts.
func drawTestDrop(r engine.Renderer, cam engine.Camera, ter *physics.Terrain, rv *Rover) {
	roverShadows(r, cam, ter, rv)
	rv.Draw(nil, cam, r)
	pose := PoseOf(rv)
	if math.Abs(pose.Tilt*180/math.Pi) < 2 {
		return
	}
	c := cam.ToScreen(rv.Center())
	const L = 150.0
	r.Line(engine.V(c.X-L, c.Y), engine.V(c.X+L, c.Y), 2, colInk.Alpha(120))
	d := engine.V(math.Cos(pose.Tilt), -math.Sin(pose.Tilt)).Mul(L)
	r.Line(c.Sub(d), c.Add(d), 3, colWarn)
}

// testVerdict is the one-line result of a test drop for the status line.
func testVerdict(rv *Rover, t float64) (string, engine.Color) {
	if t < 1.2 {
		return "Test drop: settling…", colChipLt
	}
	pose := PoseOf(rv)
	if pose.Bend > bendLimit {
		return "Test drop: the frame bends. Add a diagonal brace.", colWarn
	}
	if deg := pose.Tilt * 180 / math.Pi; math.Abs(deg) >= 2 {
		dir := "nose up"
		if deg < 0 {
			dir = "nose down"
		}
		return fmt.Sprintf("Test drop: tilts %.0f° %s.", math.Abs(deg), dir), colWarn
	}
	return "Test drop: sits level and holds its shape.", colGood
}

var colBlueprint = engine.RGB(150, 226, 255)

// drawGantry frames the build grid with two lattice towers on the real ground,
// a crossbeam of floodlights, and a landing pad under the grid. While a test
// drop runs the lamps dim so the rover stands out.
func drawGantry(r engine.Renderer, cam engine.Camera, t *physics.Terrain, dim bool) {
	const tw = 22.0 // tower width
	top := float64(bGY) - 26
	lx, rx := float64(bGX)-tw-10, float64(bGX+bCell*GridW)+10
	ground := func(x float64) float64 {
		return cam.ToScreen(engine.V(cam.ToWorld(engine.V(x, 0)).X, t.HeightAt(cam.ToWorld(engine.V(x, 0)).X))).Y
	}
	// light cones first, so everything else sits in front of them
	lamps := []float64{float64(bGX) + 90, ScreenW / 2, float64(bGX+bCell*GridW) - 90}
	glow := uint8(22)
	if dim {
		glow = 8
	}
	for _, lx := range lamps {
		g := ground(lx)
		r.Poly([]engine.Vec2{engine.V(lx-10, top+18), engine.V(lx+10, top+18), engine.V(lx+130, g), engine.V(lx-130, g)}, colFloodWarm.Alpha(glow))
		r.Poly([]engine.Vec2{engine.V(lx-6, top+18), engine.V(lx+6, top+18), engine.V(lx+60, g), engine.V(lx-60, g)}, colFloodWarm.Alpha(glow))
	}
	// landing pad: deck plates with hazard edges, following the ground
	x0, x1 := lx, rx+tw
	for x := x0; x < x1; x += 24 {
		xa, xb := x, math.Min(x+24, x1)
		ya, yb := ground(xa), ground(xb)
		r.Poly([]engine.Vec2{engine.V(xa, ya-3), engine.V(xb, yb-3), engine.V(xb, yb+9), engine.V(xa, ya+9)}, engine.RGB(64, 66, 76))
		r.Line(engine.V(xa, ya-3), engine.V(xb, yb-3), 2, engine.RGB(140, 146, 160))
		r.Line(engine.V(xa, ya+9), engine.V(xa, ya-3), 1, colIron)
	}
	for _, x := range []float64{x0, x1 - 30} {
		for k := 0.0; k < 30; k += 10 {
			ya, yb := ground(x+k), ground(x+k+10)
			col := colStar
			if int(k/10)%2 == 1 {
				col = colIron
			}
			r.Poly([]engine.Vec2{engine.V(x+k, ya+2), engine.V(x+k+10, yb+2), engine.V(x+k+10, yb+8), engine.V(x+k, ya+8)}, col)
		}
	}
	// towers and crossbeam
	for _, x := range []float64{lx, rx} {
		g := ground(x + tw/2)
		r.Rect(x-6, g-6, tw+12, 8, colIron, true)
		truss(r, engine.V(x+tw/2, g-4), engine.V(x+tw/2, top), tw)
	}
	truss(r, engine.V(lx, top+6), engine.V(rx+tw, top+6), 18)
	for _, x := range lamps {
		r.Rect(x-14, top+12, 28, 10, colIron, true)
		r.Rect(x-11, top+18, 22, 4, colFloodWarm, true)
	}
	// beacons on the tower tops
	for _, x := range []float64{lx + tw/2, rx + tw/2} {
		r.Circle(engine.V(x, top-8), 4, colBeacon, true)
		r.Circle(engine.V(x, top-8), 8, colBeacon.Alpha(50), true)
	}
}

// drawBlockMini draws a shapes block centred on c at z pixels per cell.
func drawBlockMini(r engine.Renderer, b Block, c engine.Vec2, z float64) {
	w, h := b.Size()
	at := func(cl Cell) engine.Vec2 {
		return engine.V(c.X+(float64(cl.X)-float64(w-1)/2)*z, c.Y-(float64(cl.Y)-float64(h-1)/2)*z)
	}
	d := Design{Cells: b.Cells}
	for _, cl := range b.Cells {
		for _, o := range [][2]int{{1, 0}, {0, 1}, {1, 1}, {-1, 1}} {
			if k, ok := d.At(cl.X+o[0], cl.Y+o[1]); ok {
				drawBeam(r, at(cl), at(Cell{cl.X + o[0], cl.Y + o[1], k}), z)
			}
		}
	}
	for _, cl := range b.Cells {
		drawPart(r, cl.K, at(cl), z, 0)
	}
}
