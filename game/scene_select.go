package game

import (
	"fmt"
	"math"

	"github.com/signal-k/haphazard-martians/engine"
)

// Mission select is a traverse across Mars: the campaign's missions are stops
// along one route, left to right. Swipe the route to look along it, tap a
// stop to read its brief, then build a rover for it.

// Route geometry, in route pixels (the route scrolls under the screen).
const (
	rtStart  = 150.0 // route x of mission 1
	rtStep   = 140.0 // between stops
	rtTop    = 64.0  // the map sits between the top bar and the brief
	rtBottom = 352.0
	rtBadge  = 30.0 // stop badge radius
	rtPost   = 58.0 // badge centre above the ground
)

// selMission is the stop the player last picked; 0 picks the furthest
// unlocked one. It outlives the scene so Back from the bay returns to it.
var selMission = 0

func routeWidth() float64 { return 2*rtStart + rtStep*float64(len(LevelDefs)-1) }

func stopX(i int) float64 { return rtStart + rtStep*float64(i) }

// routeGround is the screen y of the ground at route x: rolling plains that
// climb towards Mount Sharp over the last missions.
func routeGround(x float64) float64 {
	n := float64(len(LevelDefs))
	climb := smooth((x - stopX(int(n*0.7))) / (stopX(int(n)-1) - stopX(int(n*0.7))))
	return 268 + 12*math.Sin(x*0.011) + 7*math.Sin(x*0.029+1) - 96*climb
}

var biomeGround = map[string]engine.Color{
	"plains":   engine.RGB(184, 98, 60),
	"dunes":    engine.RGB(214, 150, 86),
	"boulders": engine.RGB(150, 84, 58),
	"ridge":    engine.RGB(128, 74, 58),
	"crater":   engine.RGB(160, 92, 74),
	"ice":      engine.RGB(196, 196, 206),
}

func groundAt(i int) engine.Color {
	if c, ok := biomeGround[LevelDefs[i].Biome]; ok {
		return c
	}
	return biomeGround["plains"]
}

// furthest is the index of the furthest unlocked mission.
func furthest(prog *Progress) int {
	last := 0
	for i, d := range LevelDefs {
		if prog.Unlocked(d.ID) {
			last = i
		}
	}
	return last
}

func menuScene(g *engine.Game, _ any) *engine.Scene {
	s := newScene()
	prog := LoadProgress(g.Host)
	sel := furthest(prog)
	for i, d := range LevelDefs {
		if d.ID == selMission && prog.Unlocked(d.ID) {
			sel = i
		}
	}
	maxScroll := math.Max(0, routeWidth()-ScreenW)
	clampScroll := func(v float64) float64 { return math.Max(0, math.Min(maxScroll, v)) }
	scroll := clampScroll(stopX(sel) - ScreenW/2)
	target := scroll // eases towards a picked stop
	var vel, grabX, grabScroll, lastX float64
	grabbing, dragged := false, false
	clock := 0.0
	reached := furthest(prog)

	stops := make([]*Button, len(LevelDefs))
	var drawStop func(r engine.Renderer, i int, b *Button)

	// sky, route, stops' ground
	AddDraw(s, 0, func(_ engine.Camera, r engine.Renderer) {
		drawSky(r, scroll/40)
		// the ground, its colour easing from one mission's terrain to the next
		lo := math.Max(0, math.Floor(scroll/4)*4)
		for x := lo; x < scroll+ScreenW+4 && x < routeWidth(); x += 4 {
			f := (x + 5 - rtStart) / rtStep
			i := int(math.Max(0, math.Min(float64(len(LevelDefs)-1), math.Floor(f))))
			j := min(i+1, len(LevelDefs)-1)
			col := lerpColor(groundAt(i), groundAt(j), smooth((f-float64(i)-0.3)/0.4))
			a, b := engine.V(x-scroll, routeGround(x)), engine.V(x+4-scroll, routeGround(x+4))
			r.Poly([]engine.Vec2{a, engine.V(b.X+1, b.Y), engine.V(b.X+1, rtBottom), engine.V(a.X, rtBottom)}, col)
			r.Line(a, b, 2, shade(col, 0.22))
		}
		for _, band := range []float64{34, 64} { // deeper rock reads darker
			pts := []engine.Vec2{engine.V(-10, rtBottom)}
			for x := -10.0; x <= ScreenW+10; x += 10 {
				pts = append(pts, engine.V(x, routeGround(x+scroll)+band+6*math.Sin((x+scroll)*0.023)))
			}
			pts = append(pts, engine.V(ScreenW+10, rtBottom))
			r.Poly(pts, engine.RGB(40, 14, 10).Alpha(46))
		}
		// the traverse track: driven up to the furthest unlock, dashed beyond
		for i := 0; i < len(LevelDefs)-1; i++ {
			a, b := stopX(i), stopX(i+1)
			if b-scroll < -10 || a-scroll > ScreenW+10 {
				continue
			}
			for x := a; x < b; x += 14 {
				p := engine.V(x-scroll, routeGround(x)+10)
				q := engine.V(math.Min(x+8, b)-scroll, routeGround(math.Min(x+8, b))+10)
				if i < reached {
					r.Line(p, q, 4, colNeon.Alpha(220))
				} else {
					r.Line(p, q, 3, colInk.Alpha(90))
				}
			}
		}
		// posts under the badges; names under the ground line
		for i, d := range LevelDefs {
			x := stopX(i) - scroll
			if x < -80 || x > ScreenW+80 {
				continue
			}
			gy := routeGround(stopX(i))
			locked := !prog.Unlocked(d.ID)
			post := colInk
			if locked {
				post = colInk.Alpha(110)
			}
			r.Line(engine.V(x, gy+2), engine.V(x, gy-rtPost+rtBadge), 3, post)
			name := colInk
			if locked {
				name = colInk.Alpha(150)
			}
			r.Rect(x-textW(d.Name, sizeSmall)/2-6, gy+20, textW(d.Name, sizeSmall)+12, 22, colSpace.Alpha(150), true)
			r.Text(d.Name, x, gy+22, sizeSmall, name, engine.AlignCenter)
		}
		for i, b := range stops { // badges of stops at the screen edges
			if b.Hidden && b.X+b.W > 0 && b.X < ScreenW {
				drawStop(r, i, b)
			}
		}
		// the rover parked at the picked stop
		rx := stopX(sel) - scroll + rtBadge + 18
		drawIcon(r, IconRover, rx, routeGround(stopX(sel)+rtBadge+18)-12, 30, colInk)
	})

	// stops: one button per mission, moved with the route every frame. A stop
	// only half on screen is hidden (it cannot be tapped) and the map layer
	// draws its badge instead, so stops slide in and out smoothly.
	drawStop = func(r engine.Renderer, i int, b *Button) {
		d, lp := LevelDefs[i], prog.Levels[LevelDefs[i].ID]
		c := engine.V(b.X+b.W/2, b.Y+b.H/2)
		fill, ink, line := colTile, colInk, colInk
		switch {
		case b.Disabled:
			fill, ink, line = colSpace.Alpha(200), colInk.Alpha(120), colInk.Alpha(110)
		case b.Held:
			fill, ink = colInk, colTile
		case i == sel:
			fill, ink, line = colNeon, colTile, colNeon
		}
		if i == sel {
			pulse := 0.5 + 0.5*math.Sin(clock*4)
			r.Circle(c, rtBadge+6+4*pulse, colNeon.Alpha(uint8(80+80*pulse)), false)
		}
		r.Circle(c, rtBadge, fill, true)
		r.Circle(c, rtBadge, line, false)
		r.Circle(c, rtBadge-1, line, false)
		r.Text(fmt.Sprint(d.ID), c.X, c.Y-15, 26, ink, engine.AlignCenter)
		if b.Disabled { // a padlock on the badge's shoulder
			lc := engine.V(c.X+rtBadge-4, c.Y-rtBadge+6)
			r.Circle(lc.Add(engine.V(0, -4)), 6, colMuted, false)
			r.Rect(lc.X-7, lc.Y-2, 14, 11, colMuted, true)
			return
		}
		if lp.Stars > 0 {
			drawStars(r, c.X, c.Y-rtBadge-12, 6, lp.Stars)
		}
	}

	for i, d := range LevelDefs {
		i, d := i, d
		b := AddButton(s, 0, 0, rtBadge*2+8, rtBadge*2+8, fmt.Sprintf("Mission %d", d.ID), func() {
			if dragged {
				return
			}
			sel, selMission = i, d.ID
			target = clampScroll(stopX(i) - ScreenW/2)
		})
		b.Disabled = !prog.Unlocked(d.ID)
		b.Custom = func(b *Button, r engine.Renderer) { drawStop(r, i, b) }
		stops[i] = b
	}
	place := func() {
		for i, b := range stops {
			x := stopX(i) - scroll
			b.X, b.Y = x-b.W/2, routeGround(stopX(i))-rtPost-b.H/2
			b.Hidden = b.X < 0 || b.X+b.W > ScreenW
		}
	}
	place()

	// top bar and the brief for the picked mission
	AddDraw(s, 9, func(_ engine.Camera, r engine.Renderer) {
		r.Rect(0, 0, ScreenW, rtTop, colTile.Alpha(215), true)
		r.Rect(0, rtTop-1, ScreenW, 2, colInk.Alpha(60), true)
		r.Text("HAPHAZARD MARTIANS", 24, 8, 26, colInk, engine.AlignLeft)
		stars, total := 0, 0
		for _, lp := range prog.Levels {
			stars += lp.Stars
			total += lp.Score
		}
		r.Text(fmt.Sprintf("%d of %d stars · %d points", stars, 3*len(LevelDefs), total), 25, 40, sizeSmall, colMuted, engine.AlignLeft)

		// the brief
		r.Rect(0, rtBottom, ScreenW, ScreenH-rtBottom, colTile.Alpha(235), true)
		r.Rect(0, rtBottom, ScreenW, 2, colInk.Alpha(60), true)
		d := LevelDefs[sel]
		lp := prog.Levels[d.ID]
		r.Text(fmt.Sprintf("Mission %d", d.ID), 24, rtBottom+12, sizeSmall, colNeon, engine.AlignLeft)
		r.Text(d.Name, 24, rtBottom+30, 24, colInk, engine.AlignLeft)
		r.Text(d.Blurb, 24, rtBottom+62, sizeSmall, colMuted, engine.AlignLeft)
		meta := fmt.Sprintf("%.0f m · par %.0fs · %d credits", d.Length, d.Par, d.Credits)
		if rl := RuleLine(d); rl != "" {
			meta += " · " + rl
		}
		r.Text(meta, 24, rtBottom+82, sizeSmall, colChipLt, engine.AlignLeft)
		drawStars(r, 640, rtBottom+44, 10, lp.Stars)
		best := "not yet run"
		if lp.Score > 0 {
			best = fmt.Sprintf("best %d", lp.Score)
		}
		r.Text(best, 640, rtBottom+60, sizeSmall, colMuted, engine.AlignCenter)
	})

	build := AddButton(s, ScreenW-24-200, rtBottom+18, 200, 64, "Build rover", func() {
		g.Load(SceneBuild, NewSessionFor(g.Host, BuildLevel(LevelDefs[sel])))
	})
	build.Primary, build.Icon = true, IconRight

	// other ways to play, along the bottom
	ex := AddButton(s, 24, 466, 284, 60, fmt.Sprintf("Expedition · %d missions", len(ExpeditionLevels)), func() {
		g.Load(SceneBriefing, NewExpedition())
	})
	ex.Icon = IconRover
	if m, ok, next := CurrentMission(); ok || next != nil {
		label := "Daily mission"
		if !ok {
			m, label = *next, "Coming "+next.Date
		}
		dm := AddButton(s, 320, 466, 316, 60, label, func() {
			g.Load(SceneBuild, NewSessionFor(g.Host, BuildLevel(m.Level)))
		})
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
			r.Text(fmt.Sprintf("+%d parts", m.Reward.Credits), b.X+b.W-14, b.Y+8, sizeSmall, sub, engine.AlignRight)
		}
	}
	sl := "Shape school"
	if n := SchoolDone(g.Host); n > 0 && n < len(Lessons) {
		sl = fmt.Sprintf("Shape school %d/%d", n, len(Lessons))
	}
	sc := AddButton(s, 648, 466, 216, 60, sl, func() { g.Load(SceneBuild, LessonSession(NextLesson(g.Host))) })
	sc.Icon = IconRover
	st := AddButton(s, ScreenW-84, 466, 60, 60, "Style sheet", func() { g.Load(SceneStyle, nil) })
	st.Icon, st.HideLabel = IconGear, true

	// swiping the route: drag anywhere on the map, with a little fling
	s.OnUpdate = func(dt float64) {
		clock += dt
		in := g.In
		p := in.Pointer
		if in.Pressed && p.Y > rtTop && p.Y < rtBottom {
			grabbing, dragged = true, false
			grabX, grabScroll, lastX, vel = p.X, scroll, p.X, 0
		}
		if grabbing && in.Down {
			if math.Abs(p.X-grabX) > 10 {
				dragged = true
			}
			if dragged {
				scroll = clampScroll(grabScroll - (p.X - grabX))
				target = scroll
				if dt > 0 {
					vel = -(p.X - lastX) / dt
				}
			}
			lastX = p.X
		}
		if !in.Down && grabbing {
			grabbing = false
			if !dragged {
				vel = 0
			}
		}
		if !grabbing {
			if math.Abs(vel) > 5 { // fling, slowing down
				scroll = clampScroll(scroll + vel*dt)
				target = scroll
				vel *= math.Pow(0.04, dt)
			} else {
				vel = 0
				scroll += (target - scroll) * math.Min(1, dt*8)
			}
		}
		if in.Pressed && !(p.Y > rtTop && p.Y < rtBottom) {
			dragged = false
		}
		place()
	}
	return s
}
