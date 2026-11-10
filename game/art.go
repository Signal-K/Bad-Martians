package game

import (
	"math"

	"github.com/signal-k/haphazard-martians/engine"
	"github.com/signal-k/haphazard-martians/physics"
)

// World art: the sky, the ground, rover parts and their shadows. The drive and
// the rover bay both draw through these, so a rover looks the same in both.

// shade moves c toward white (t > 0) or black (t < 0).
func shade(c engine.Color, t float64) engine.Color {
	out := lerpColor(c, engine.RGB(0, 0, 0), -t)
	if t > 0 {
		out = lerpColor(c, engine.RGB(255, 255, 255), t)
	}
	out.A = c.A
	return out
}

var (
	colSteel     = engine.RGB(168, 176, 190)
	colSteelLt   = engine.RGB(214, 220, 230)
	colSteelDk   = engine.RGB(92, 98, 114)
	colIron      = engine.RGB(30, 32, 40) // outlines and dark housings
	colRubber    = engine.RGB(42, 40, 46)
	colLens      = engine.RGB(120, 220, 255)
	colBeacon    = engine.RGB(255, 84, 72)
	colFloodWarm = engine.RGB(255, 236, 196)
)

// ngon returns n points on a circle of radius rad around c, starting at a0.
func ngon(c engine.Vec2, rad float64, n int, a0 float64) []engine.Vec2 {
	pts := make([]engine.Vec2, n)
	for i := range pts {
		a := a0 + float64(i)*2*math.Pi/float64(n)
		pts[i] = engine.V(c.X+rad*math.Cos(a), c.Y-rad*math.Sin(a))
	}
	return pts
}

// ellipse is a filled ellipse made of a polygon.
func ellipse(r engine.Renderer, c engine.Vec2, rx, ry float64, col engine.Color) {
	pts := make([]engine.Vec2, 16)
	for i := range pts {
		a := float64(i) * 2 * math.Pi / 16
		pts[i] = engine.V(c.X+rx*math.Cos(a), c.Y+ry*math.Sin(a))
	}
	r.Poly(pts, col)
}

// arc strokes part of a circle from angle a0 to a1 (radians, CCW on screen).
func arc(r engine.Renderer, c engine.Vec2, rad, a0, a1, w float64, col engine.Color) {
	const n = 8
	prev := engine.V(c.X+rad*math.Cos(a0), c.Y-rad*math.Sin(a0))
	for i := 1; i <= n; i++ {
		a := a0 + (a1-a0)*float64(i)/n
		p := engine.V(c.X+rad*math.Cos(a), c.Y-rad*math.Sin(a))
		r.Line(prev, p, w, col)
		prev = p
	}
}

// truss draws a lattice girder from a to b, w pixels wide: a dark web, two
// rails lit from above, and zigzag bracing between them.
func truss(rd engine.Renderer, a, b engine.Vec2, w float64) {
	d := b.Sub(a)
	L := d.Len()
	if L < 1 {
		return
	}
	u := d.Mul(1 / L)
	n := engine.V(-u.Y, u.X)
	if n.Y > 0 { // keep n pointing up the screen so the lit rail is on top
		n = n.Mul(-1)
	}
	h := w / 2
	rd.Line(a, b, w+2, colIron.Alpha(230))
	rd.Line(a, b, w*0.7, colSteelDk.Alpha(120))
	top0, top1 := a.Add(n.Mul(h*0.62)), b.Add(n.Mul(h*0.62))
	bot0, bot1 := a.Sub(n.Mul(h*0.62)), b.Sub(n.Mul(h*0.62))
	seg := math.Max(2, math.Round(L/(w*1.4)))
	step := 1 / seg
	for i := 0.0; i < seg; i++ {
		p0 := bot0.Add(bot1.Sub(bot0).Mul(i * step))
		p1 := top0.Add(top1.Sub(top0).Mul((i + 0.5) * step))
		p2 := bot0.Add(bot1.Sub(bot0).Mul((i + 1) * step))
		rd.Line(p0, p1, math.Max(1, w*0.16), colSteel)
		rd.Line(p1, p2, math.Max(1, w*0.16), colSteel)
	}
	rd.Line(top0, top1, math.Max(1.2, w*0.22), colSteelLt)
	rd.Line(bot0, bot1, math.Max(1.2, w*0.22), colSteelDk)
}

func drawBeam(rd engine.Renderer, a, b engine.Vec2, z float64) {
	// trim the ends so the girder meets the node plates rather than their centres
	d := b.Sub(a)
	if L := d.Len(); L > 0.6*z {
		u := d.Mul(1 / L)
		a, b = a.Add(u.Mul(0.22*z)), b.Sub(u.Mul(0.22*z))
	}
	truss(rd, a, b, 0.24*z)
}

// drawPart draws one part centred at c with z pixels per metre. angle is a
// wheel's rotation; other parts ignore it.
func drawPart(rd engine.Renderer, k Kind, c engine.Vec2, z, angle float64) {
	p, _ := PartOf(k)
	switch {
	case p.Wheel:
		drawWheel(rd, c, p.Radius*z, z, angle, p.Motor)
	case p.Kind == KMast:
		drawMast(rd, c, z)
	default:
		drawNode(rd, c, z)
	}
}

// drawNode is a frame joint: a bolted hex plate.
func drawNode(rd engine.Renderer, c engine.Vec2, z float64) {
	R := 0.32 * z
	rd.Poly(ngon(c, R+0.05*z, 6, math.Pi/6), colIron)
	rd.Poly(ngon(c, R, 6, math.Pi/6), colSteel)
	rd.Poly(ngon(c.Add(engine.V(-0.03*z, -0.03*z)), R*0.8, 6, math.Pi/6), colSteelLt)
	rd.Circle(c, R*0.62, colSteel, true)
	rd.Circle(c, R*0.62, colSteelDk, false)
	for _, b := range ngon(c, R*0.66, 4, math.Pi/4) {
		rd.Circle(b, 0.05*z, colSteelDk, true)
		rd.Circle(b.Add(engine.V(-0.012*z, -0.012*z)), 0.022*z, colSteelLt, true)
	}
	rd.Circle(c, R*0.2, colSteelDk, true)
}

// drawWheel is a knobbly tyre on a spoked rim. A motor wheel has an orange rim
// and a hub motor.
func drawWheel(rd engine.Renderer, c engine.Vec2, R, z, angle float64, motor bool) {
	rim := colSteel
	if motor {
		rim = colMars
	}
	rd.Circle(c, R+0.03*z, colIron, true)
	// tread lugs turn with the wheel, so you can see it roll
	const lugs = 14
	for i := 0; i < lugs; i++ {
		a := angle + float64(i)*2*math.Pi/lugs
		dir := engine.V(math.Cos(a), -math.Sin(a))
		rd.Line(c.Add(dir.Mul(R*0.8)), c.Add(dir.Mul(R*1.03)), 0.12*z, colIron)
	}
	rd.Circle(c, R*0.9, colRubber, true)
	rd.Circle(c, R*0.74, shade(colRubber, 0.12), true)
	arc(rd, c, R*0.83, 1.9, 2.9, 0.05*z, engine.RGB(255, 255, 255).Alpha(40)) // sheen
	rd.Circle(c, R*0.6, shade(rim, -0.35), true)
	rd.Circle(c, R*0.54, rim, true)
	rd.Circle(c, R*0.42, shade(rim, -0.25), true)
	for i := 0; i < 5; i++ {
		a := angle + float64(i)*2*math.Pi/5
		dir := engine.V(math.Cos(a), -math.Sin(a))
		rd.Line(c.Add(dir.Mul(R*0.16)), c.Add(dir.Mul(R*0.5)), 0.08*z, shade(rim, 0.15))
	}
	if motor {
		rd.Circle(c, R*0.3, colIron, true)
		for i := 0; i < 8; i++ { // cooling fins
			a := angle + float64(i)*math.Pi/4
			dir := engine.V(math.Cos(a), -math.Sin(a))
			rd.Line(c.Add(dir.Mul(R*0.14)), c.Add(dir.Mul(R*0.28)), 0.035*z, engine.RGB(80, 74, 76))
		}
		rd.Circle(c, R*0.12, engine.RGB(255, 176, 90), true)
		rd.Circle(c, R*0.06, engine.RGB(255, 236, 200), true)
	} else {
		rd.Circle(c, R*0.17, colSteelDk, true)
		rd.Circle(c, R*0.09, colSteelLt, true)
	}
}

// drawMast is the Navcam: a turret, a pole, and a stereo camera head with an
// antenna.
func drawMast(rd engine.Renderer, c engine.Vec2, z float64) {
	body := engine.RGB(90, 170, 230)
	rd.Circle(c, 0.34*z, colIron, true)
	rd.Circle(c, 0.3*z, shade(body, -0.25), true)
	rd.Circle(c.Add(engine.V(-0.03*z, -0.04*z)), 0.24*z, body, true)
	rd.Circle(c.Add(engine.V(-0.08*z, -0.09*z)), 0.08*z, shade(body, 0.4), true)
	top := c.Add(engine.V(0, -0.82*z))
	rd.Line(c, top, 0.14*z, colIron)
	rd.Line(c, top, 0.08*z, colSteel)
	rd.Line(c.Add(engine.V(-0.015*z, 0)), top.Add(engine.V(-0.015*z, 0)), 0.025*z, colSteelLt)
	for _, f := range []float64{0.45, 0.65} { // collars
		y := c.Y - f*z
		rd.Line(engine.V(c.X-0.07*z, y), engine.V(c.X+0.07*z, y), 0.04*z, colIron)
	}
	// antenna whip with a beacon
	ab := top.Add(engine.V(0.2*z, 0))
	at := ab.Add(engine.V(0.08*z, -0.36*z))
	rd.Line(ab, at, 0.03*z, colIron)
	rd.Circle(at, 0.05*z, colBeacon, true)
	rd.Circle(at, 0.09*z, colBeacon.Alpha(60), true)
	// camera head
	hw, hh := 0.56*z, 0.28*z
	hx, hy := top.X-hw/2, top.Y-hh
	rd.Rect(hx-0.03*z, hy-0.03*z, hw+0.06*z, hh+0.06*z, colIron, true)
	rd.Rect(hx, hy, hw, hh, engine.RGB(58, 64, 80), true)
	rd.Rect(hx, hy, hw, hh*0.3, engine.RGB(92, 100, 120), true)
	for _, dx := range []float64{-0.13, 0.13} {
		l := engine.V(top.X+dx*z, hy+hh*0.56)
		rd.Circle(l, 0.085*z, colIron, true)
		rd.Circle(l, 0.06*z, colLens, true)
		rd.Circle(l.Add(engine.V(-0.02*z, -0.02*z)), 0.02*z, engine.RGB(255, 255, 255), true)
	}
}

// drawShadows lays a soft contact shadow on the ground under each part. It
// fades as the part rises, so a rover reads as sitting on the terrain.
func drawShadows(rd engine.Renderer, cam engine.Camera, t *physics.Terrain, pos []engine.Vec2, radius []float64) {
	for i, p := range pos {
		gy := t.HeightAt(p.X)
		lift := p.Y - radius[i] - gy
		a := 1 - lift/2.5
		if a <= 0 {
			continue
		}
		g := cam.ToScreen(engine.V(p.X, gy))
		rx := (radius[i]*1.3 + 0.15*lift) * cam.Zoom
		ellipse(rd, g, rx, 0.14*cam.Zoom, engine.RGB(40, 14, 8).Alpha(uint8(90*math.Min(1, a))))
	}
}

// RoverShadows is drawShadows for a live rover.
func roverShadows(rd engine.Renderer, cam engine.Camera, t *physics.Terrain, rv *Rover) {
	pos := make([]engine.Vec2, len(rv.Bodies))
	rad := make([]float64, len(rv.Bodies))
	for i, b := range rv.Bodies {
		pos[i], rad[i] = b.Pos, b.Radius
	}
	drawShadows(rd, cam, t, pos, rad)
}

// drawSky paints the Martian day: a butterscotch gradient, a small pale sun
// with its blue halo, Phobos, and three parallax layers of mesas and dunes
// fading into dust haze.
func drawSky(r engine.Renderer, camX float64) {
	top, mid, hor := engine.RGB(96, 62, 70), engine.RGB(170, 108, 92), engine.RGB(240, 192, 148)
	const bands = 36
	for i := 0; i < bands; i++ {
		t := float64(i) / (bands - 1)
		col := lerpColor(top, mid, math.Min(1, t*1.6))
		if t > 0.6 {
			col = lerpColor(mid, hor, (t-0.6)/0.4)
		}
		r.Rect(0, ScreenH*float64(i)/bands, ScreenW, ScreenH/bands+1, col, true)
	}
	sun := engine.V(892-camX*0.4, 132)
	for i := 6; i >= 1; i-- {
		r.Circle(sun, 14+float64(i)*11, engine.RGB(196, 214, 236).Alpha(uint8(8+i)), true)
	}
	r.Circle(sun, 15, engine.RGB(250, 248, 236), true)
	ph := engine.V(196-camX*0.6, 74)
	r.Poly(ngon(ph, 7, 7, 0.3), engine.RGB(176, 150, 140))
	r.Circle(ph.Add(engine.V(-1.5, -1.5)), 3.5, engine.RGB(204, 180, 168), true)

	// far mesas: flat tops come from clipping a sum of sines
	mesa := func(wx float64) float64 {
		v := math.Sin(wx*0.006) + 0.6*math.Sin(wx*0.0137+2) + 0.3*math.Sin(wx*0.031+1)
		return math.Max(-0.2, math.Min(0.55, v))
	}
	ridge(r, camX, 0.06, func(wx float64) float64 { return 300 - 70*mesa(wx) }, engine.RGB(200, 140, 118))
	haze(r, 230, 340, 56)
	ridge(r, camX, 0.15, func(wx float64) float64 {
		return 336 + 22*math.Sin(wx*0.011) + 11*math.Sin(wx*0.025+1)
	}, engine.RGB(196, 128, 98))
	haze(r, 300, 390, 40)
	ridge(r, camX, 0.35, func(wx float64) float64 {
		return 384 + 20*math.Sin(wx*0.017) + 10*math.Sin(wx*0.039+1)
	}, engine.RGB(166, 102, 82))
}

// haze lays dust over the layers behind it, thickest at the bottom.
func haze(r engine.Renderer, y0, y1 float64, a uint8) {
	const n = 10
	h := (y1 - y0) / n
	for i := 0; i < n; i++ {
		r.Rect(0, y0+float64(i)*h, ScreenW, h+1, engine.RGB(240, 192, 148).Alpha(uint8(float64(a)*float64(i+1)/n)), true)
	}
}

func ridge(r engine.Renderer, camX, par float64, y func(float64) float64, col engine.Color) {
	pts := []engine.Vec2{engine.V(0, ScreenH)}
	for x := 0.0; x <= ScreenW+12; x += 12 {
		pts = append(pts, engine.V(x, y(x+camX*par*40)))
	}
	pts = append(pts, engine.V(ScreenW+12, ScreenH))
	r.Poly(pts, col)
	// a lit crest line
	for i := 2; i < len(pts)-1; i++ {
		r.Line(pts[i-1], pts[i], 1.5, shade(col, 0.18))
	}
}

// depthShade darkens the ground with depth: translucent black bands over
// whatever layer lies there, so rock 10 m down reads darker than rock at 2 m.
var depthShade = []struct {
	top, bot float64
	alpha    uint8
}{
	{1.4, 3.2, 36},
	{3.2, 7, 77},
	{7, 60, 115},
}

// drawTerrain paints the visible stretch of a heightfield as a cross-section:
// a crust in the surface class colour that fades between stretches, then the
// pebble bed, packed soil and rock (see Ground), a lit rim, and pebbles,
// ripples, cracks or boulders by class. Decoration is seeded by sample index,
// so it holds still as you drive.
func drawTerrain(r engine.Renderer, cam engine.Camera, t *physics.Terrain) {
	g := GroundOf(t)
	half := cam.Screen.X/2/cam.Zoom + 1
	i0 := int((cam.Pos.X-half-t.X0)/t.DX) - 1
	i1 := int((cam.Pos.X+half-t.X0)/t.DX) + 1
	i0 = max(i0, 0)
	i1 = min(i1, t.Len()-2)
	z := cam.Zoom
	down := func(p engine.Vec2, m float64) engine.Vec2 { return engine.V(p.X, p.Y+m*z) }
	band := func(a, b engine.Vec2, ta, tb, ba, bb float64, c engine.Color) {
		a0, b0 := down(a, ta), down(b, tb)
		if a0.Y > ScreenH && b0.Y > ScreenH {
			return
		}
		a1, b1 := down(a, ba), down(b, bb)
		r.Poly([]engine.Vec2{engine.V(a0.X-0.5, a0.Y), engine.V(b0.X+0.5, b0.Y), engine.V(b1.X+0.5, b1.Y), engine.V(a1.X-0.5, a1.Y)}, c)
	}
	for i := i0; i <= i1; i++ {
		a := cam.ToScreen(engine.V(t.X0+float64(i)*t.DX, t.H[i]))
		b := cam.ToScreen(engine.V(t.X0+float64(i+1)*t.DX, t.H[i+1]))
		ca, cb := g.Crust[i], g.Crust[i+1]
		crust := lerpColor(ca, cb, 0.5)
		band(a, b, 0, 0, g.CrustD[i], g.CrustD[i+1], crust)
		band(a, b, g.CrustD[i], g.CrustD[i+1], g.PebD[i], g.PebD[i+1], lerpColor(colPebbleBed, crust, 0.2))
		band(a, b, g.PebD[i], g.PebD[i+1], g.RockD[i], g.RockD[i+1], lerpColor(colPacked, crust, 0.12))
		band(a, b, g.RockD[i], g.RockD[i+1], 60, 60, colRock)
		band(a, b, 0, 0, 0.22, 0.22, shade(crust, 0.14))
		for _, s := range depthShade {
			band(a, b, s.top, s.top, s.bot, s.bot, engine.RGB(16, 10, 14).Alpha(s.alpha))
		}
	}
	for i := i0; i <= i1; i++ {
		a := cam.ToScreen(engine.V(t.X0+float64(i)*t.DX, t.H[i]))
		b := cam.ToScreen(engine.V(t.X0+float64(i+1)*t.DX, t.H[i+1]))
		cls := Class(t.Seg[i])
		base := lerpColor(g.Crust[i], g.Crust[i+1], 0.5)
		rg := newRng(uint64(i)*2654435761 + 17)
		at := func(f, depth float64) engine.Vec2 { return down(a.Add(b.Sub(a).Mul(f)), depth) }
		lerpD := func(d []float64, f float64) float64 { return d[i] + (d[i+1]-d[i])*f }
		// the pebble bed: rounded stones of mixed tone
		for k := 0; k < 3; k++ {
			f := rg.float()
			top, bot := lerpD(g.CrustD, f), lerpD(g.PebD, f)
			if bot-top < 0.08 {
				continue
			}
			p := at(f, rg.rangef(top+0.04, bot-0.04))
			rad := rg.rangef(0.04, 0.1) * z
			c := lerpColor(colPebbleBed, engine.RGB(200, 170, 150), rg.rangef(0, 0.5))
			r.Circle(p.Add(engine.V(rad*0.2, rad*0.25)), rad, shade(colPebbleBed, -0.35), true)
			r.Circle(p, rad, c, true)
		}
		// grit in the packed soil, cracks in the rock
		{
			f := rg.float()
			top, bot := lerpD(g.PebD, f), lerpD(g.RockD, f)
			if bot-top > 0.1 {
				r.Circle(at(f, rg.rangef(top, bot)), rg.rangef(0.03, 0.07)*z, shade(colPacked, -0.25).Alpha(170), true)
			}
			if rg.float() < 0.35 {
				d := lerpD(g.RockD, f) + rg.rangef(0.3, 2.5)
				p := at(f, d)
				r.Line(p, p.Add(engine.V(rg.rangef(-0.2, 0.2)*z, rg.rangef(0.3, 0.8)*z)), 1.4, shade(colRock, -0.3))
			}
		}
		switch cls {
		case Sand:
			// cross-bedding in the sand crust
			f := rg.float()
			if d := lerpD(g.CrustD, f); d > 0.4 {
				p := at(f*0.6, rg.rangef(0.3, d-0.1))
				r.Line(p, p.Add(engine.V(0.4*z, rg.rangef(-0.05, 0.05)*z)), 1.4, shade(base, 0.22))
			}
		case Bedrock:
			if rg.float() < 0.5 {
				p := at(rg.float(), 0.06)
				r.Line(p, p.Add(engine.V(rg.rangef(-0.1, 0.1)*z, rg.rangef(0.2, 0.6)*z)), 2, shade(colRock, -0.32))
			}
		case BigRock:
			if rg.float() < 0.55 {
				drawRock(r, at(rg.float(), 0.1), rg.rangef(0.16, 0.34)*z, base, rg)
			}
		default:
			if rg.float() < 0.3 {
				drawRock(r, at(rg.float(), 0.03), rg.rangef(0.05, 0.11)*z, base, rg)
			}
		}
		if cls == Sand && rg.float() < 0.08 || cls == Bedrock && rg.float() < 0.15 {
			drawRock(r, at(rg.float(), 0.02), rg.rangef(0.04, 0.08)*z, shade(Classes[BigRock].Color, 0.2), rg)
		}
		r.Line(a, b, math.Max(2, 0.07*z), shade(base, 0.32))
	}
	for _, f := range g.Finds {
		c := cam.ToScreen(engine.V(f.X, t.HeightAt(f.X)+0))
		if c.X < -60 || c.X > cam.Screen.X+60 {
			continue
		}
		drawFind(r, down(c, f.Depth), f.Size*z, f.Vein)
	}
}

// drawFind is an interesting rock: a dark stone crossed by a bright mineral
// vein, with a faint glow so it reads as worth a closer look.
func drawFind(r engine.Renderer, c engine.Vec2, rad float64, vein engine.Color) {
	r.Circle(c, rad*2.2, vein.Alpha(34), true)
	r.Circle(c, rad*1.5, vein.Alpha(52), true)
	pts := make([]engine.Vec2, 7)
	for i := range pts {
		a := float64(i) * 2 * math.Pi / 7
		k := rad * (0.85 + 0.2*math.Sin(float64(i)*2.3))
		pts[i] = engine.V(c.X+k*1.2*math.Cos(a), c.Y-k*0.9*math.Sin(a))
	}
	r.Poly(pts, engine.RGB(58, 48, 54))
	r.Line(c.Add(engine.V(-rad*0.9, rad*0.25)), c.Add(engine.V(-rad*0.1, -rad*0.15)), math.Max(1.5, rad*0.22), vein)
	r.Line(c.Add(engine.V(-rad*0.1, -rad*0.15)), c.Add(engine.V(rad*0.85, -rad*0.4)), math.Max(1.5, rad*0.22), vein)
	r.Circle(c.Add(engine.V(-rad*0.4, -rad*0.45)), rad*0.18, engine.RGB(150, 140, 150), true)
}

// drawRock is an irregular stone lit from the upper left, half sunk at c.
func drawRock(r engine.Renderer, c engine.Vec2, rad float64, ground engine.Color, rg *rng) {
	col := lerpColor(ground, engine.RGB(70, 58, 58), 0.6)
	n := 7
	pts := make([]engine.Vec2, n)
	for i := range pts {
		a := float64(i) * 2 * math.Pi / float64(n)
		k := rad * rg.rangef(0.75, 1.1)
		pts[i] = engine.V(c.X+k*1.25*math.Cos(a), c.Y-k*0.85*math.Sin(a))
	}
	r.Poly(pts, shade(col, -0.25))
	r.Poly(ngon(c.Add(engine.V(-rad*0.18, -rad*0.22)), rad*0.62, 6, 0.4), col)
	r.Circle(c.Add(engine.V(-rad*0.35, -rad*0.4)), rad*0.22, shade(col, 0.28), true)
}
