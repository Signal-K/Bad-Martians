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
	Active    bool
	Scene     *Scene
	comps     []Component
	started   int
	destroyed bool
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
	Game    *Game
	Cam     Camera
	Objects []*GameObject
	// OnUpdate runs once per fixed step before components, for scene-level
	// logic (state machines, win conditions).
	OnUpdate func(dt float64)
	Time     float64
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
		for _, c := range o.comps {
			if u, ok := c.(Updater); ok {
				u.Update(o, dt)
			}
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
	for _, o := range order {
		for _, c := range o.comps {
			if d, ok := c.(Drawer); ok {
				d.Draw(o, s.Cam, r)
			}
		}
	}
}
