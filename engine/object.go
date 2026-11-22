package engine

import "sort"

// Component is anything attached to a GameObject. Behaviour is opt-in through
// the Starter / Updater / Drawer interfaces below.
type Component any

type Starter interface{ Start(g *GameObject) }
type Updater interface {
	Update(g *GameObject, dt float64)
}
type Drawer interface {
	Draw(g *GameObject, cam Camera, r Renderer)
}

// GameObject is a named entity with a transform and components.
type GameObject struct {
	Name      string
	Tag       string
	Pos       Vec2
	Rot       float64
	Layer     int // draw order, low first
	// AX and AY pin the object to an edge of the display: -1 left/top, +1
	// right/bottom, 0 stays with the centred design area. The object is laid out
	// in design coordinates; the engine shifts drawing and the pointer by the
	// display's overscan so it follows its edge on any aspect ratio.
	AX, AY int
	Active    bool
	Scene     *Scene
	comps     []Component
	started   int
	destroyed bool
}

// Offset is how far the object's anchor moves it from its design position.
func (g *GameObject) Offset() Vec2 {
	return Vec2{float64(g.AX) * Overscan.X, float64(g.AY) * Overscan.Y}
}

func (g *GameObject) Add(c Component) Component {
	g.comps = append(g.comps, c)
	return c
}
func (g *GameObject) Destroy() { g.destroyed = true }

// Get returns the first component of type T.
func Get[T any](g *GameObject) (T, bool) {
	for _, c := range g.comps {
		if t, ok := c.(T); ok {
			return t, true
		}
	}
	var zero T
	return zero, false
}

// Scene owns GameObjects and a physics world. It is what Unity calls a Scene
// and Godot a SceneTree root.
type Scene struct {
	Name    string
	Meta    SceneMeta
	Game    *Game
	Cam     Camera
	Objects []*GameObject
	// OnUpdate runs once per fixed step before components, for scene-level
	// logic (state machines, win conditions).
	OnUpdate func(dt float64)
	Time     float64

	// Photos: objects on PhotoUI and above (default 10, the HUD) are left
	// out, and OnPhoto draws a caption over the shot instead.
	PhotoUI int
	OnPhoto func(r Renderer)
}

func (s *Scene) Spawn(name string) *GameObject {
	g := &GameObject{Name: name, Active: true, Scene: s}
	s.Objects = append(s.Objects, g)
	return g
}

func (s *Scene) Find(name string) *GameObject {
	for _, o := range s.Objects {
		if o.Name == name && !o.destroyed {
			return o
		}
	}
	return nil
}

func (s *Scene) FindTag(tag string) []*GameObject {
	var out []*GameObject
	for _, o := range s.Objects {
		if o.Tag == tag && !o.destroyed {
			out = append(out, o)
		}
	}
	return out
}

func (s *Scene) update(dt float64) {
	s.Time += dt
	if s.OnUpdate != nil {
		s.OnUpdate(dt)
	}
	// Index loop: components may Spawn during Update.
	for i := 0; i < len(s.Objects); i++ {
		o := s.Objects[i]
		if o.destroyed || !o.Active {
			continue
		}
		for o.started < len(o.comps) {
			c := o.comps[o.started]
			o.started++
			if st, ok := c.(Starter); ok {
				st.Start(o)
			}
		}
		off := o.Offset()
		var game *Game
		var saved Input
		if off != (Vec2{}) {
			game = s.Game
		}
		if game != nil { // the object sees touches in its own design space
			saved = game.In
			game.In.Pointer = saved.Pointer.Sub(off)
			if len(saved.Touches) > 0 {
				game.In.Touches = make([]Touch, len(saved.Touches))
				for i, t := range saved.Touches {
					t.Pos = t.Pos.Sub(off)
					game.In.Touches[i] = t
				}
			}
		}
		for _, c := range o.comps {
			if u, ok := c.(Updater); ok {
				u.Update(o, dt)
			}
		}
		if game != nil {
			game.In.Pointer, game.In.Touches = saved.Pointer, saved.Touches
		}
	}
	live := s.Objects[:0]
	for _, o := range s.Objects {
		if !o.destroyed {
			live = append(live, o)
		}
	}
	s.Objects = live
}

func (s *Scene) draw(r Renderer) {
	order := make([]*GameObject, 0, len(s.Objects))
	for _, o := range s.Objects {
		if o.Active && !o.destroyed {
			order = append(order, o)
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return order[i].Layer < order[j].Layer })
	snap := s.Game != nil && s.Game.Snapping
	hideFrom := s.PhotoUI
	if hideFrom == 0 {
		hideFrom = 10
	}
	for _, o := range order {
		if snap && o.Layer >= hideFrom {
			continue
		}
		rr := r
		if off := o.Offset(); off != (Vec2{}) {
			rr = &shiftRenderer{r, off}
		}
		for _, c := range o.comps {
			if d, ok := c.(Drawer); ok {
				d.Draw(o, s.Cam, rr)
			}
		}
	}
	if snap && s.OnPhoto != nil {
		s.OnPhoto(r)
	}
}

// Overscan is how far the visible area extends past the design area on each
// side, in design units: the display's extra width on the left and right, and
// extra height above and below. Hosts set it when the window changes shape.
var Overscan Vec2

// shiftRenderer draws everything displaced by off.
type shiftRenderer struct {
	Renderer
	off Vec2
}

func (s *shiftRenderer) Line(a, b Vec2, w float64, c Color) {
	s.Renderer.Line(a.Add(s.off), b.Add(s.off), w, c)
}
func (s *shiftRenderer) Circle(p Vec2, r float64, c Color, fill bool) {
	s.Renderer.Circle(p.Add(s.off), r, c, fill)
}
func (s *shiftRenderer) Rect(x, y, w, h float64, c Color, fill bool) {
	s.Renderer.Rect(x+s.off.X, y+s.off.Y, w, h, c, fill)
}
func (s *shiftRenderer) Poly(pts []Vec2, c Color) {
	q := make([]Vec2, len(pts))
	for i, p := range pts {
		q[i] = p.Add(s.off)
	}
	s.Renderer.Poly(q, c)
}
func (s *shiftRenderer) Text(t string, x, y, size float64, c Color, a Align) {
	s.Renderer.Text(t, x+s.off.X, y+s.off.Y, size, c, a)
}
