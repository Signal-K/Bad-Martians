package game

import (
	"fmt"
	"math"
	"time"

	"github.com/signal-k/haphazard-martians/engine"
)

// The Mars map is the campaign's home: the planet, with each chapter pinned to
// the place it is set, and the rover's journey drawn from one to the next.
// Tap a place to read about it; tap it again, or Enter, to open its route.

const (
	mapCX, mapCY = 330.0, 214.0 // the disc's centre
	mapR         = 140.0        // and radius
	mapPin       = 26.0         // pin badge radius
	mapPanel     = 358.0        // top of the chapter panel
)

// selChapter is the place last picked on the map; -1 picks the one the rover
// is parked at. It outlives the scene so coming back from a route keeps it.
var selChapter = -1

// placePos is where a chapter lies on the map: a spot on the disc, or a moon's
// centre on its orbit.
func placePos(c *Chapter) engine.Vec2 {
	return engine.V(mapCX+c.Pin.X*mapR, mapCY+c.Pin.Y*mapR)
}

// moonRad is a moon's drawn radius on the map; 0 for places on the disc.
func moonRad(c *Chapter) float64 {
	switch c.Key {
	case "phobos":
		return 20
	case "deimos":
		return 15
	case "moon":
		return 18
	}
	return 0
}

// pinPos is where a chapter's badge sits. On the disc it is the place itself;
// a moon's badge floats above it, so the moon stays in view.
func pinPos(c *Chapter) engine.Vec2 {
	p := placePos(c)
	if rad := moonRad(c); rad > 0 {
		p.Y -= rad + mapPin + 8
	}
	return p
}

// drawMars paints the planet: a rust disc lit from the upper left, dark
// regions, Olympus Mons, Valles Marineris and both polar caps. Everything is
// kept inside the disc, since the renderer cannot clip.
func drawMars(r engine.Renderer, c engine.Vec2, R float64) {
	r.Circle(c, R*1.14, engine.RGB(214, 120, 84).Alpha(16), true)
	r.Circle(c, R*1.07, engine.RGB(214, 120, 84).Alpha(30), true)
	r.Circle(c, R, engine.RGB(150, 74, 48), true)
	r.Circle(c.Add(engine.V(-R*0.11, -R*0.09)), R*0.88, engine.RGB(190, 100, 64), true)
	r.Circle(c.Add(engine.V(-R*0.16, -R*0.13)), R*0.62, engine.RGB(206, 116, 74).Alpha(150), true)
	// dark plains (Syrtis, Acidalia) and a bright basin (Hellas)
	r.Circle(c.Add(engine.V(R*0.36, -R*0.1)), R*0.2, engine.RGB(112, 56, 42).Alpha(150), true)
	r.Circle(c.Add(engine.V(R*0.28, -R*0.02)), R*0.12, engine.RGB(112, 56, 42).Alpha(150), true)
	r.Circle(c.Add(engine.V(-R*0.5, -R*0.34)), R*0.16, engine.RGB(120, 60, 44).Alpha(130), true)
	r.Circle(c.Add(engine.V(-R*0.12, R*0.62)), R*0.17, engine.RGB(226, 150, 108).Alpha(140), true)
	// Olympus Mons, and the long scar of Valles Marineris
	r.Circle(c.Add(engine.V(-R*0.55, -R*0.04)), R*0.09, engine.RGB(220, 138, 96), true)
	r.Circle(c.Add(engine.V(-R*0.55, -R*0.04)), R*0.045, engine.RGB(160, 84, 56), true)
	r.Line(c.Add(engine.V(-R*0.4, R*0.2)), c.Add(engine.V(R*0.22, R*0.3)), 7, engine.RGB(100, 48, 36).Alpha(190))
	r.Line(c.Add(engine.V(-R*0.4, R*0.2)), c.Add(engine.V(R*0.22, R*0.3)), 3, engine.RGB(70, 34, 28).Alpha(190))
	// polar caps: the part of the disc beyond a chord, its inner edge bowed
	cap := func(chord, bow float64, col engine.Color) {
		sign := math.Copysign(1, chord)
		phi := math.Acos(math.Abs(chord)) // half the cap's angle, from the pole
		pole := sign * math.Pi / 2
		var pts []engine.Vec2
		for k := 0; k <= 16; k++ { // along the limb
			t := pole - phi + 2*phi*float64(k)/16
			pts = append(pts, c.Add(engine.V(math.Cos(t)*R, math.Sin(t)*R)))
		}
		first, last := pts[0], pts[len(pts)-1]
		for k := 1; k < 16; k++ { // and back along the inner edge, bowed into the disc
			f := float64(k) / 16
			u := 2*f - 1
			x := last.X + (first.X-last.X)*f
			pts = append(pts, engine.V(x, first.Y-sign*bow*R*(1-u*u)))
		}
		r.Poly(pts, col)
	}
	cap(-0.8, 0.07, engine.RGB(244, 236, 230))
	cap(0.9, 0.05, engine.RGB(230, 220, 214))
	r.Circle(c, R, engine.RGB(60, 24, 20).Alpha(40), false)
}

// drawMoon paints a small lumpy moon at c.
func drawMoon(r engine.Renderer, c engine.Vec2, rad float64, body *Body, seed uint64) {
	rg := newRng(seed)
	var pts []engine.Vec2
	for k := 0; k < 18; k++ {
		a := float64(k) / 18 * 2 * math.Pi
		pts = append(pts, c.Add(engine.V(math.Cos(a)*rad*(1.18-0.3*rg.float()), math.Sin(a)*rad*(0.95-0.2*rg.float()))))
	}
	r.Poly(pts, body.Tint)
	r.Circle(c.Add(engine.V(-rad*0.25, -rad*0.1)), rad*0.3, shade(body.Tint, 0.18), true)
	r.Circle(c.Add(engine.V(rad*0.3, rad*0.2)), rad*0.2, shade(body.Tint, 0.18), true)
}

// arcLine draws a circle's arc from angle a0 to a1.
func arcLine(r engine.Renderer, c engine.Vec2, rad, a0, a1, w float64, col engine.Color) {
	n := int(math.Max(2, math.Abs(a1-a0)*rad/6))
	prev := c.Add(engine.V(math.Cos(a0)*rad, math.Sin(a0)*rad))
	for k := 1; k <= n; k++ {
		a := a0 + (a1-a0)*float64(k)/float64(n)
		p := c.Add(engine.V(math.Cos(a)*rad, math.Sin(a)*rad))
		r.Line(prev, p, w, col)
		prev = p
	}
}

// legPoints samples the curved leg of the journey from a to b.
func legPoints(a, b engine.Vec2) []engine.Vec2 {
	mid := a.Add(b).Mul(0.5)
	d := b.Sub(a)
	ctrl := mid.Add(engine.V(-d.Y, d.X).Mul(0.22))
	const n = 28
	pts := make([]engine.Vec2, n+1)
	for k := 0; k <= n; k++ {
		t := float64(k) / n
		u := 1 - t
		pts[k] = a.Mul(u * u).Add(ctrl.Mul(2 * u * t)).Add(b.Mul(t * t))
	}
	return pts
}

func mapScene(g *engine.Game, _ any) *engine.Scene {
	s := newScene()
	prog := LoadProgress(g.Host)
	here := CurrentChapter(prog) // where the rover is parked
	sel := here
	if selChapter >= 0 && selChapter < len(Chapters) {
		sel = selChapter
	}
	clock := 0.0

	enter := func(i int) {
		if !Chapters[i].Unlocked(prog) {
			return
		}
		selChapter = i
		g.Load(SceneMenu, &Chapters[i])
	}

	AddDraw(s, 0, func(_ engine.Camera, r engine.Renderer) {
		drawBackdrop(r)
		centre := engine.V(mapCX, mapCY)
		// the moons' orbits
		for i := range Chapters {
			if c := &Chapters[i]; c.IsMoon() && Bodies[c.Body].Orbits {
				rad := placePos(c).Sub(centre).Len()
				arcLine(r, centre, rad, -0.9, 0.9, 1.5, colInk.Alpha(46))
			}
		}
		drawMars(r, centre, mapR)
		for i := range Chapters {
			if c := &Chapters[i]; c.IsMoon() {
				drawMoon(r, placePos(c), moonRad(c), Bodies[c.Body], uint64(40+i))
			}
		}
		// the journey: driven in neon up to the furthest chapter, dashed beyond
		for i := 0; i < len(Chapters)-1; i++ {
			pts := legPoints(pinPos(&Chapters[i]), pinPos(&Chapters[i+1]))
			driven := Chapters[i+1].Unlocked(prog)
			for k := 0; k+1 < len(pts); k++ {
				if driven {
					r.Line(pts[k], pts[k+1], 4, colNeon.Alpha(230))
				} else if k%2 == 0 {
					r.Line(pts[k], pts[k+1], 3, colInk.Alpha(110))
				}
			}
		}
		// the rover, parked beside where the journey has reached
		rp := pinPos(&Chapters[here])
		bob := 2 * math.Sin(clock*3)
		drawIcon(r, IconRover, rp.X-mapPin-24, rp.Y+bob, 32, colInk)
	})

	// one pin per chapter. Locked chapters can still be selected, to read about
	// them, but not entered.
	pins := make([]*Button, len(Chapters))
	for i := range Chapters {
		i := i
		c := &Chapters[i]
		p := pinPos(c)
		b := AddButton(s, p.X-mapPin-4, p.Y-mapPin-4, 2*mapPin+8, 2*mapPin+8, c.Name, func() {
			if sel == i {
				enter(i)
				return
			}
			sel, selChapter = i, i
		})
		b.Custom = func(b *Button, r engine.Renderer) {
			ctr := engine.V(b.X+b.W/2, b.Y+b.H/2)
			locked := !c.Unlocked(prog)
			fill, ink, line := colTile, colInk, colInk
			switch {
			case locked:
				fill, ink, line = colSpace.Alpha(215), colInk.Alpha(130), colInk.Alpha(110)
			case b.Held:
				fill, ink = colInk, colTile
			case sel == i:
				fill, ink, line = colNeon, colTile, colNeon
			}
			if sel == i {
				pulse := 0.5 + 0.5*math.Sin(clock*4)
				r.Circle(ctr, mapPin+7+4*pulse, colNeon.Alpha(uint8(80+80*pulse)), false)
			}
			// progress ring: the share of the chapter's missions won
			r.Circle(ctr, mapPin+3, colInk.Alpha(60), false)
			if n := c.Done(prog); n > 0 {
				arcLine(r, ctr, mapPin+3, -math.Pi/2, -math.Pi/2+2*math.Pi*float64(n)/float64(len(c.Levels())), 4, colGood)
			}
			r.Circle(ctr, mapPin, fill, true)
			r.Circle(ctr, mapPin, line, false)
			r.Circle(ctr, mapPin-1, line, false)
			if c.Complete(prog) {
				m := engine.V(ctr.X-2, ctr.Y+9) // a tick, drawn: the font has no check mark
				r.Line(engine.V(ctr.X-11, ctr.Y), m, 5, ink)
				r.Line(m, engine.V(ctr.X+11, ctr.Y-11), 5, ink)
			} else {
				r.Text(fmt.Sprint(i+1), ctr.X, ctr.Y-15, 26, ink, engine.AlignCenter)
			}
			name := colInk
			if locked {
				name = colInk.Alpha(150)
			}
			lw := textW(c.Name, sizeSmall)
			drop := 0.0 // a moon's name goes under the moon, not under its badge
			if rad := moonRad(c); rad > 0 {
				drop = 2*rad + 8
			}
			r.Rect(ctr.X-lw/2-6, ctr.Y+mapPin+8+drop, lw+12, 22, colSpace.Alpha(170), true)
			r.Text(c.Name, ctr.X, ctr.Y+mapPin+10+drop, sizeSmall, name, engine.AlignCenter)
			if locked { // a padlock on the badge's shoulder
				lc := engine.V(ctr.X+mapPin-4, ctr.Y-mapPin+6)
				r.Circle(lc.Add(engine.V(0, -4)), 6, colMuted, false)
				r.Rect(lc.X-7, lc.Y-2, 14, 11, colMuted, true)
			}
		}
		pins[i] = b
	}

	// top bar: the journey so far
	AddDrawAt(s, 9, 0, -1, func(_ engine.Camera, r engine.Renderer) {
		r.Rect(viewL(), 0, viewR()-viewL(), rtTop, colTile.Alpha(215), true)
		r.Rect(viewL(), rtTop-1, viewR()-viewL(), 2, colInk.Alpha(60), true)
		r.Text("HAPHAZARD MARTIANS", 24, 8, 26, colInk, engine.AlignLeft)
		stars, total, won := 0, 0, 0
		for _, lp := range prog.Levels {
			stars += lp.Stars
			total += lp.Score
			if lp.Stars > 0 {
				won++
			}
		}
		r.Text(fmt.Sprintf("%d of %d stars · %d points", stars, 3*len(LevelDefs), total), 25, 40, sizeSmall, colMuted, engine.AlignLeft)
		if prog.FreeRetries > 0 {
			r.Text(fmt.Sprintf("Free retries: %d", prog.FreeRetries), ScreenW-24, 10, sizeSmall, colGood, engine.AlignRight)
		}
		r.Text(fmt.Sprintf("Travelled %.1f km · %d of %d missions", DistanceDriven(prog)/1000, won, len(LevelDefs)), ScreenW-24, 40, sizeSmall, colMuted, engine.AlignRight)
	})

	// the chapter panel, on the bottom edge
	AddDrawAt(s, 9, 0, 1, func(_ engine.Camera, r engine.Renderer) {
		c := &Chapters[sel]
		r.Rect(viewL(), mapPanel, viewR()-viewL(), viewB()-mapPanel, colTile.Alpha(235), true)
		r.Rect(viewL(), mapPanel, viewR()-viewL(), 2, colInk.Alpha(60), true)
		r.Text(fmt.Sprintf("Chapter %d of %d", sel+1, len(Chapters)), 24, mapPanel+10, sizeSmall, colNeon, engine.AlignLeft)
		r.Text(c.Name, 24, mapPanel+28, 24, colInk, engine.AlignLeft)
		r.Text(c.Blurb, 24, mapPanel+58, sizeSmall, colMuted, engine.AlignLeft)
		var meta string
		switch {
		case !c.Unlocked(prog):
			meta = fmt.Sprintf("Locked · win mission %d to reach it", c.First-1)
		default:
			got, most := c.Stars(prog)
			meta = fmt.Sprintf("%d of %d missions won · %d of %d stars", c.Done(prog), len(c.Levels()), got, most)
		}
		r.Text(meta, 24, mapPanel+78, sizeSmall, colChipLt, engine.AlignLeft)
	})
	ent := AddButton(s, ScreenW-24-220, mapPanel+16, 220, 64, "Enter chapter", func() { enter(sel) }).Pin(0, 1)
	ent.Primary, ent.Icon = true, IconRight

	// other ways to play, along the bottom
	savedExpedition := LoadExpedition(g.Host)
	expeditionLabel := fmt.Sprintf("Expedition · %d missions", len(ExpeditionLevels))
	if savedExpedition != nil {
		expeditionLabel = "Resume expedition"
	}
	ex := AddButton(s, 24, 466, 284, 60, expeditionLabel, func() {
		if savedExpedition != nil {
			g.Load(SceneBriefing, savedExpedition)
		} else {
			g.Load(SceneBriefing, NewExpedition())
		}
	}).Pin(0, 1)
	ex.Icon = IconRover
	if m, ok, next := CurrentMission(); ok || next != nil {
		label := "Daily mission"
		if !ok {
			m, label = *next, "Coming "+next.Date
		}
		dm := AddButton(s, 320, 466, 316, 60, label, func() {
			Track(g.Host, EvDailyStarted, map[string]any{"date": m.Date, "level": m.Level.ID})
			g.Load(SceneBuild, NewSessionFor(g.Host, BuildLevel(m.Level)))
		}).Pin(0, 1)
		dm.Disabled = !ok
		dm.Custom = func(b *Button, r engine.Renderer) {
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
			r.Text(m.Title, b.X+14, b.Y+28, 18, ink, engine.AlignLeft)
			r.Text(fmt.Sprintf("+%d build credits", m.Reward.Credits), b.X+b.W-14, b.Y+8, sizeSmall, sub, engine.AlignRight)
			if m.FreeRetry > 0 && !prog.ClaimedMissions[m.Date] {
				r.Text(fmt.Sprintf("+%d free retry on win", m.FreeRetry), b.X+b.W-14, b.Y+34, 11, sub, engine.AlignRight)
			}
		}
	}
	// Shape school and the style sheet are archived for the first release (MAR):
	// their scenes and tests stay, but nothing on the map opens them.
	acct := LoadAccount(g.Host)
	label := "Account"
	switch {
	case acct.Username != "":
		label = acct.Username
	case acct.SignedIn:
		label = "Set username"
	case acct.Sync != SyncDemo:
		label = "Sign in"
	}
	ac := AddButton(s, ScreenW-24-300, 466, 300, 60, clip(label, 18), func() { g.Load(SceneAccount, nil) }).Pin(0, 1)
	ac.Icon = IconGear
	ac.Custom = func(b *Button, r engine.Renderer) {
		fill, ink, line := colTile, colInk, colInk
		if b.Held {
			fill, ink = colInk, colTile
		}
		r.Rect(b.X, b.Y, b.W, b.H, fill, true)
		outline(r, b.X, b.Y, b.W, b.H, line)
		r.Text("Account", b.X+14, b.Y+8, sizeSmall, colNeon, engine.AlignLeft)
		r.Text(clip(label, 18), b.X+14, b.Y+28, 18, ink, engine.AlignLeft)
		text, good := acct.SyncLine(time.Now())
		_ = text
		dot := colWarn
		if good {
			dot = colGood
		}
		r.Circle(engine.V(b.X+b.W-20, b.Y+b.H/2), 6, dot, true)
	}

	s.OnUpdate = func(dt float64) {
		clock += dt
		ent.Disabled = !Chapters[sel].Unlocked(prog)
	}
	return s
}
