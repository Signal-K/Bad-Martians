package engine

// FixedDT is the simulation step. Physics and gameplay are deterministic at
// this rate regardless of display refresh.
const FixedDT = 1.0 / 60.0

// SceneFactory builds a scene; arg carries data between scenes (e.g. a level).
type SceneFactory func(g *Game, arg any) *Scene

// Game runs one active Scene on a fixed timestep and handles scene switches.
type Game struct {
	W, H      float64
	Host      Host
	In        Input
	scene     *Scene
	factories map[string]SceneFactory
	metas     map[string]SceneMeta
	order     []string
	pending   *pendingLoad
	acc       float64
	prevDown  bool
	pressL    bool // latched edges, consumed by the next fixed step
	releaseL  bool

	// Ed is the in-game editor (see editor.go).
	Ed Editor
}

type pendingLoad struct {
	name string
	arg  any
}

func NewGame(w, h float64, host Host) *Game {
	if host == nil {
		host = NewMemHost()
	}
	return &Game{W: w, H: h, Host: host, factories: map[string]SceneFactory{}, metas: map[string]SceneMeta{}}
}

// Load switches scene at the next frame boundary (safe to call from inside an
// Update).
func (g *Game) Load(name string, arg any) { g.pending = &pendingLoad{name, arg} }

func (g *Game) Scene() *Scene { return g.scene }

func (g *Game) applyPending() {
	if g.pending == nil {
		return
	}
	p := g.pending
	g.pending = nil
	f, ok := g.factories[p.name]
	if !ok {
		panic("engine: unknown scene " + p.name)
	}
	s := f(g, p.arg)
	s.Name, s.Game, s.Meta = p.name, g, g.metas[p.name]
	s.Cam.Screen = Vec2{g.W, g.H}
	if s.Cam.Zoom == 0 {
		s.Cam.Zoom = 1
	}
	g.scene = s
	g.Ed.Selected = nil
	g.Host.Emit("scene", []byte(`{"name":"`+p.name+`","type":"`+string(s.Meta.Type)+`"}`)) // hosts may adapt their chrome
}

// Tick advances the game by frameDt seconds of wall time using the given
// input snapshot. It may run zero or several fixed steps.
func (g *Game) Tick(frameDt float64, in Input) {
	if frameDt > 0.1 {
		frameDt = 0.1 // tab was backgrounded; don't spiral
	}
	g.applyPending()
	pressed, released := in.Down && !g.prevDown, !in.Down && g.prevDown
	if in.ToggleEdit {
		g.SetEditor(!g.Ed.On)
	}
	if g.Ed.On && g.scene != nil {
		if in.Pause {
			g.Do("Pause")
		}
		if in.Step {
			g.Do("Step")
		}
		if in.Delete {
			g.Do("Delete")
		}
		g.editorPointer(in, pressed, released)
		// The scene sees an idle pointer while editing, so taps select instead of firing buttons.
		g.prevDown = in.Down
		g.pressL, g.releaseL = false, false
		g.In = Input{Pointer: in.Pointer}
		steps := g.Ed.steps
		g.Ed.steps = 0
		if !g.Ed.Paused {
			g.acc += frameDt
			for g.acc >= FixedDT {
				g.acc -= FixedDT
				steps++
			}
		}
		for ; steps > 0 && g.scene != nil; steps-- {
			g.scene.update(FixedDT)
			if g.pending != nil {
				break
			}
		}
		return
	}
	g.pressL = g.pressL || (in.Down && !g.prevDown)
	g.releaseL = g.releaseL || (!in.Down && g.prevDown)
	g.prevDown = in.Down
	g.In = in
	if g.scene == nil {
		return
	}
	g.acc += frameDt
	for g.acc >= FixedDT {
		g.acc -= FixedDT
		// Edge flags are delivered to exactly one fixed step.
		g.In.Pressed, g.In.Released = g.pressL, g.releaseL
		g.pressL, g.releaseL = false, false
		g.scene.update(FixedDT)
		g.applyPending()
		if g.pending != nil || g.scene == nil {
			break
		}
	}
}

func (g *Game) Draw(r Renderer) {
	if g.scene != nil {
		g.scene.draw(r)
		if g.Ed.On {
			g.drawEditor(r)
		}
	}
}
