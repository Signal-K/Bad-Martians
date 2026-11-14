package game

import (
	"encoding/json"
	"fmt"

	"github.com/signal-k/haphazard-martians/engine"
)

// launchScene names the rover before it is sent out. The name is typed on a
// keyboard, entered through the host's text prompt (tap the name on a phone),
// or rolled from the generator. It travels with the rover: run HUD, mission
// report, run log, and on to the next mission if the rover survives.
func launchScene(g *engine.Game, arg any) *engine.Scene {
	ses := arg.(*Session)
	lvl := ses.Level
	s := newScene()
	prog := LoadProgress(g.Host)
	roll := prog.Launched
	if ses.Name == "" {
		ses.Name = SuggestName(roll)
	}
	name := []rune(ses.Name)
	editing := false // the player has started typing over the suggestion
	blink := 0.0

	addBackdrop(s)
	addTitle(s, "Name your rover", fmt.Sprintf("Mission %d · %s. It goes out under this name.", lvl.ID, lvl.Name))

	const fx, fy, fw, fh = 120.0, 150.0, 720.0, 96.0
	AddDraw(s, 2, func(_ engine.Camera, r engine.Renderer) {
		drawPanel(r, 96, 124, 768, 300, "Rover")
		// what is going out with it
		y := 300.0
		r.Text(fmt.Sprintf("%d parts · cost %d of %d credits", len(ses.Design.Cells), ses.Design.Cost(), ses.Budget()), 120, y, sizeBody, colInk, engine.AlignLeft)
		if ses.Spare != "" {
			r.Text(fmt.Sprintf("Salvaged %s from last mission: +%d credits", partName(ses.Spare), ses.SpareCredits()), 120, y+28, sizeSmall, colGood, engine.AlignLeft)
		}
		r.Text("Finish without scraping a frame or the mast to salvage a part.", 120, y+54, sizeSmall, colMuted, engine.AlignLeft)
		r.Text("Tap the name to type it. Shuffle for another.", 120, y+78, sizeSmall, colMuted, engine.AlignLeft)
	})

	// The name field is a button: a tap asks the host for a text box, which a
	// phone needs to show its keyboard. Desktop typing works without it.
	field := AddButton(s, fx, fy, fw, fh, "Rover name", func() {
		p, _ := json.Marshal(map[string]any{"field": "rover_name", "title": "Name your rover", "value": string(name), "max": MaxNameLen})
		g.Host.Emit("prompt", p)
		editing = true
	})
	field.Custom = func(b *Button, r engine.Renderer) {
		line := colInk
		if editing {
			line = colNeon
		}
		r.Rect(b.X, b.Y, b.W, b.H, colTile, true)
		outline(r, b.X, b.Y, b.W, b.H, line)
		txt := string(name)
		col := colInk
		if txt == "" {
			txt, col = "Type a name", colMuted
		}
		r.Text(txt, b.X+24, b.Y+30, 36, col, engine.AlignLeft)
		if int(blink*2)%2 == 0 {
			cx := fx + 24 + textW(string(name), 36) + 4
			if len(name) == 0 {
				cx = fx + 24
			}
			r.Rect(cx, b.Y+28, 3, 44, colNeon, true)
		}
		r.Text(fmt.Sprintf("%d/%d", len(name), MaxNameLen), b.X+b.W-16, b.Y+b.H-26, sizeSmall, colMuted, engine.AlignRight)
	}
	shuffle := AddButton(s, 640, 330, 200, 60, "Shuffle", func() {
		roll++
		name, editing = []rune(SuggestName(roll)), false
	})
	shuffle.Icon = IconGear

	launch := func() {
		n := CleanName(string(name))
		if n == "" {
			return
		}
		ses.Name = n
		p := LoadProgress(g.Host)
		p.Launched++
		p.Save(g.Host)
		g.Load(SceneRun, ses)
	}
	addBack(s, func() {
		ses.Name = CleanName(string(name))
		g.Load(SceneBuild, ses)
	})
	goBtn := AddButton(s, 730, 470, 206, 60, "Send it out", launch)
	goBtn.Primary, goBtn.Icon = true, IconRight

	// first-time players: one line of coaching, pointing at the launch button
	if tut := ses.Tut; tut != nil && !tut.Done {
		o := s.Spawn("tutorial")
		o.Layer = 30
		o.Add(DrawFunc(func(_ engine.Camera, r engine.Renderer) {
			drawCoach(r, blink, engine.Rect{X: 360, Y: 440, W: 350, H: 48}, "Name your rover, then send it out.",
				engine.Rect{X: goBtn.X, Y: goBtn.Y, W: goBtn.W, H: goBtn.H})
		}))
	}

	s.OnUpdate = func(dt float64) {
		blink += dt
		in := g.In
		if in.Replacing {
			name, editing = []rune(CleanName(in.Replace)), true
		}
		if len(in.Typed) > 0 {
			if !editing { // first keypress replaces the suggestion
				name, editing = nil, true
			}
			for _, c := range in.Typed {
				if len(name) < MaxNameLen && (nameRune(c) || c == ' ' && len(name) > 0 && name[len(name)-1] != ' ') {
					name = append(name, c)
				}
			}
		}
		if in.Backspace && len(name) > 0 {
			name, editing = name[:len(name)-1], true
		}
		goBtn.Disabled = CleanName(string(name)) == ""
		if in.Enter && !goBtn.Disabled {
			launch()
		}
	}
	return s
}
