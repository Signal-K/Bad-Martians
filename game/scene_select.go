package game

import (
	"fmt"
	"math"

	"github.com/signal-k/haphazard-martians/engine"
)

// Mission select is one chapter's traverse: its missions are stops along a
// route, left to right. Swipe the route to look along it, tap a stop to read
// its brief, then build a rover for it. The Mars map (scene_map.go) is where
// chapters are picked.

// Route geometry, in route pixels (the route scrolls under the screen).
const (
	rtStart  = 150.0 // route x of a chapter's first stop, when it fills the screen
	rtStep   = 140.0 // between stops
	rtTop    = 64.0  // the map sits between the top bar and the brief
	rtBottom = 392.0
	rtBadge  = 30.0 // stop badge radius
	rtPost   = 58.0 // badge centre above the ground
)

// selMission is the stop the player last picked; 0 picks the furthest
// unlocked one. It outlives the scene so Back from the bay returns to it.
var selMission = 0

// route is a chapter's geometry. A short chapter is centred rather than
// pinned to the left edge.
type route struct {
	ch    *Chapter
	defs  []LevelDef
	start float64
}

func newRoute(ch *Chapter) route {
	defs := ch.Levels()
	span := rtStep * float64(len(defs)-1)
	return route{ch: ch, defs: defs, start: math.Max(rtStart, (ScreenW-span)/2)}
}

func (rt route) width() float64 {
	return rt.start + rtStep*float64(len(rt.defs)-1) + math.Min(rt.start, rtStart)
}

func (rt route) stopX(i int) float64 { return rt.start + rtStep*float64(i) }

// ground is the screen y of the ground at route x: rolling plains, rising
// towards the last stop when the chapter climbs.
func (rt route) ground(x float64) float64 {
	climb := 0.0
	if rt.ch.Climb && len(rt.defs) > 1 {
		climb = smooth((x - rt.stopX(0)) / (rt.stopX(len(rt.defs)-1) - rt.stopX(0)))
	}
	return 290 + 12*math.Sin(x*0.011) + 7*math.Sin(x*0.029+1) - 110*climb
}

var biomeGround = map[string]engine.Color{
	"plains":   engine.RGB(184, 98, 60),
	"dunes":    engine.RGB(214, 150, 86),
	"boulders": engine.RGB(150, 84, 58),
	"ridge":    engine.RGB(128, 74, 58),
	"crater":   engine.RGB(160, 92, 74),
	"ice":      engine.RGB(196, 196, 206),
}

func groundAt(d LevelDef) engine.Color {
	if b := BodyOf(d); b.TintMix > 0 {
		return b.ground(biomeGround[d.Biome])
	}
	if c, ok := biomeGround[d.Biome]; ok {
		return c
	}
	return biomeGround["plains"]
}

// chapterFor is the chapter the route opens on: the one asked for, else the
// one holding the mission last picked, else the furthest the player has reached.
func chapterFor(arg any, prog *Progress) *Chapter {
	if c, ok := arg.(*Chapter); ok && c != nil {
		return c
	}
	if selMission != 0 {
		return &Chapters[ChapterIndex(selMission)]
	}
	return &Chapters[CurrentChapter(prog)]
}

func menuScene(g *engine.Game, arg any) *engine.Scene {
	s := newScene()
	prog := LoadProgress(g.Host)
	ch := chapterFor(arg, prog)
	rt := newRoute(ch)
	defs := rt.defs
	sel := max(0, ch.Furthest(prog))
	for i, d := range defs {
		if d.ID == selMission && prog.Unlocked(d.ID) {
			sel = i
		}
	}
	maxScroll := math.Max(0, rt.width()-ScreenW)
	clampScroll := func(v float64) float64 { return math.Max(0, math.Min(maxScroll, v)) }
	scroll := clampScroll(rt.stopX(sel) - ScreenW/2)
	target := scroll // eases towards a picked stop
	var vel, grabX, grabScroll, lastX float64
	grabbing, dragged := false, false
	clock := 0.0
	reached := max(0, ch.Furthest(prog))

	stops := make([]*Button, len(defs))
	var drawStop func(r engine.Renderer, i int, b *Button)

	// sky, route, stops' ground
	AddDraw(s, 0, func(_ engine.Camera, r engine.Renderer) {
		drawSky(r, scroll/40, ch.Sky())
		// the ground, its colour easing from one mission's terrain to the next
		lo := math.Floor((scroll+viewL())/4) * 4
		for x := lo; x < scroll+viewR()+4; x += 4 {
			f := (x + 5 - rt.start) / rtStep
			i := int(math.Max(0, math.Min(float64(len(defs)-1), math.Floor(f))))
			j := min(i+1, len(defs)-1)
			col := lerpColor(groundAt(defs[i]), groundAt(defs[j]), smooth((f-float64(i)-0.3)/0.4))
			a, b := engine.V(x-scroll, rt.ground(x)), engine.V(x+4-scroll, rt.ground(x+4))
			r.Poly([]engine.Vec2{a, engine.V(b.X+1, b.Y), engine.V(b.X+1, viewB()), engine.V(a.X, viewB())}, col)
			r.Line(a, b, 2, shade(col, 0.22))
		}
		for _, band := range []float64{34, 64} { // deeper rock reads darker
			pts := []engine.Vec2{engine.V(viewL()-10, viewB())}
			for x := viewL() - 10; x <= viewR()+10; x += 10 {
				pts = append(pts, engine.V(x, rt.ground(x+scroll)+band+6*math.Sin((x+scroll)*0.023)))
			}
			pts = append(pts, engine.V(viewR()+10, viewB()))
			r.Poly(pts, engine.RGB(40, 14, 10).Alpha(46))
		}
		// the traverse track: driven up to the furthest unlock, dashed beyond
		for i := 0; i < len(defs)-1; i++ {
			a, b := rt.stopX(i), rt.stopX(i+1)
			if b-scroll < viewL()-10 || a-scroll > viewR()+10 {
				continue
			}
			for x := a; x < b; x += 14 {
				p := engine.V(x-scroll, rt.ground(x)+10)
				q := engine.V(math.Min(x+8, b)-scroll, rt.ground(math.Min(x+8, b))+10)
				if i < reached {
					r.Line(p, q, 4, colNeon.Alpha(220))
				} else {
					r.Line(p, q, 3, colInk.Alpha(90))
				}
			}
		}
		// posts under the badges; names under the ground line
		for i, d := range defs {
			x := rt.stopX(i) - scroll
			if x < viewL()-80 || x > viewR()+80 {
				continue
			}
			gy := rt.ground(rt.stopX(i))
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
			if b.Hidden && b.X+b.W > viewL() && b.X < viewR() {
				drawStop(r, i, b)
			}
		}
		// the rover parked at the picked stop
		rx := rt.stopX(sel) - scroll + rtBadge + 18
		drawIcon(r, IconRover, rx, rt.ground(rt.stopX(sel)+rtBadge+18)-12, 30, colInk)
	})

	// stops: one button per mission, moved with the route every frame. A stop
	// only half on screen is hidden (it cannot be tapped) and the map layer
	// draws its badge instead, so stops slide in and out smoothly.
	drawStop = func(r engine.Renderer, i int, b *Button) {
		d, lp := defs[i], prog.Levels[defs[i].ID]
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

	for i, d := range defs {
		i, d := i, d
		b := AddButton(s, 0, 0, rtBadge*2+8, rtBadge*2+8, fmt.Sprintf("Mission %d", d.ID), func() {
			if dragged {
				return
			}
			sel, selMission = i, d.ID
			target = clampScroll(rt.stopX(i) - ScreenW/2)
		})
		b.Disabled = !prog.Unlocked(d.ID)
		b.Custom = func(b *Button, r engine.Renderer) { drawStop(r, i, b) }
		stops[i] = b
	}
	place := func() {
		for i, b := range stops {
			x := rt.stopX(i) - scroll
			b.X, b.Y = x-b.W/2, rt.ground(rt.stopX(i))-rtPost-b.H/2
			b.Hidden = b.X < viewL() || b.X+b.W > viewR()
		}
	}
	place()

	// top bar: the chapter, and the way back to the Mars map
	AddDrawAt(s, 9, 0, -1, func(_ engine.Camera, r engine.Renderer) {
		r.Rect(viewL(), 0, viewR()-viewL(), rtTop, colTile.Alpha(215), true)
		r.Rect(viewL(), rtTop-1, viewR()-viewL(), 2, colInk.Alpha(60), true)
		r.Text(ch.Name, 92, 6, 26, colInk, engine.AlignLeft)
		got, most := ch.Stars(prog)
		span := fmt.Sprintf("Missions %d–%d", ch.First, ch.Last)
		if ch.First == ch.Last {
			span = fmt.Sprintf("Mission %d", ch.First)
		}
		r.Text(fmt.Sprintf("%s · %d of %d stars", span, got, most), 93, 40, sizeSmall, colMuted, engine.AlignLeft)
		if prog.FreeRetries > 0 {
			r.Text(fmt.Sprintf("Free retries: %d", prog.FreeRetries), ScreenW-24, 10, sizeSmall, colGood, engine.AlignRight)
		}
		d := defs[sel]
		r.Text(fmt.Sprintf("Obstacles %.0f%% · difficulty %.0f%%", 100*d.Obstacle(), 100*d.Difficulty()), ScreenW-24, 40, sizeSmall, colMuted, engine.AlignRight)
	})
	mapBack := AddButton(s, 16, 10, 60, 44, "Mars map", func() { g.Load(SceneMap, nil) }).Pin(0, -1)
	mapBack.Icon, mapBack.HideLabel = IconBack, true

	// the brief, on the bottom edge with its button
	AddDrawAt(s, 9, 0, 1, func(_ engine.Camera, r engine.Renderer) {
		d := defs[sel]
		r.Rect(viewL(), rtBottom, viewR()-viewL(), viewB()-rtBottom, colTile.Alpha(235), true)
		r.Rect(viewL(), rtBottom, viewR()-viewL(), 2, colInk.Alpha(60), true)
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
			if lp.Best > 0 {
				best += " · " + fmtTime(lp.Best)
			}
		}
		r.Text(best, 640, rtBottom+60, sizeSmall, colMuted, engine.AlignCenter)
	})

	build := AddButton(s, ScreenW-24-200, rtBottom+24, 200, 64, "Build rover", func() {
		g.Load(SceneBuild, NewSessionFor(g.Host, BuildLevel(defs[sel])))
	}).Pin(0, 1)
	build.Primary, build.Icon = true, IconRight

	// swiping the route: drag anywhere on the map, with a little fling
	s.OnUpdate = func(dt float64) {
		clock += dt
		in := g.In
		p := in.Pointer
		// the map is what lies between the top bar and the brief, which hang
		// from the display's edges
		inMap := func(p engine.Vec2) bool {
			return p.Y > rtTop-engine.Overscan.Y && p.Y < rtBottom+engine.Overscan.Y
		}
		if in.Pressed && inMap(p) {
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
		if in.Pressed && !inMap(p) {
			dragged = false
		}
		place()
	}
	return s
}
