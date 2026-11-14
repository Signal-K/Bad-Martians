package engine

import (
	"fmt"
	"sort"
)

// SceneType says what job a scene does. Every scene must declare one.
type SceneType string

const (
	TypeMenu     SceneType = "menu"     // navigation
	TypeBuild    SceneType = "build"    // construct something on a grid
	TypePlay     SceneType = "play"     // live physics simulation
	TypeActivity SceneType = "activity" // citizen-science task
	TypeResults  SceneType = "results"  // outcome and scoring
	TypeTravel   SceneType = "travel"   // movement between expedition stages
	TypeSandbox  SceneType = "sandbox"  // free-form authoring
	TypeDebug    SceneType = "debug"    // tooling, style sheets, test beds
)

var sceneTypes = map[SceneType]bool{
	TypeMenu: true, TypeBuild: true, TypePlay: true, TypeActivity: true,
	TypeResults: true, TypeTravel: true, TypeSandbox: true, TypeDebug: true,
}

// Rect is an axis-aligned screen region in virtual pixels.
type Rect struct{ X, Y, W, H float64 }

func (r Rect) Contains(o Rect) bool {
	return o.X >= r.X-0.5 && o.Y >= r.Y-0.5 && o.X+o.W <= r.X+r.W+0.5 && o.Y+o.H <= r.Y+r.H+0.5
}

// Layout names the screen regions a scene arranges its UI in. Scenes with the
// same layout share the same bones, which keeps the UI consistent.
type Layout struct {
	Name    string
	Regions map[string]Rect
}

// SceneMeta is the contract every scene registers with.
type SceneMeta struct {
	Name   string
	Title  string
	Type   SceneType
	Layout Layout
	// Fixture builds the argument the scene needs to boot on its own, so any
	// scene can be opened directly (?scene=name, -scene name) and tested alone.
	Fixture func(g *Game) any
}

func (m SceneMeta) validate(w, h float64) error {
	switch {
	case m.Name == "":
		return fmt.Errorf("scene has no name")
	case !sceneTypes[m.Type]:
		return fmt.Errorf("scene %q: missing or unknown type %q", m.Name, m.Type)
	case m.Layout.Name == "" || len(m.Layout.Regions) == 0:
		return fmt.Errorf("scene %q: missing layout", m.Name)
	case m.Fixture == nil:
		return fmt.Errorf("scene %q: missing fixture (it must run standalone)", m.Name)
	}
	screen := Rect{0, 0, w, h}
	for n, r := range m.Layout.Regions {
		if r.W <= 0 || r.H <= 0 || !screen.Contains(r) {
			return fmt.Errorf("scene %q: layout region %q is outside the screen", m.Name, n)
		}
	}
	return nil
}

// RegisterScene adds a scene. It panics if the contract is not met, so a scene
// without a type, layout or fixture cannot exist.
func (g *Game) RegisterScene(m SceneMeta, f SceneFactory) {
	if err := m.validate(g.W, g.H); err != nil {
		panic("engine: " + err.Error())
	}
	if _, dup := g.factories[m.Name]; dup {
		panic("engine: duplicate scene " + m.Name)
	}
	g.factories[m.Name] = f
	g.metas[m.Name] = m
	g.order = append(g.order, m.Name)
}

// Scenes lists registered scenes in registration order.
func (g *Game) Scenes() []SceneMeta {
	out := make([]SceneMeta, 0, len(g.order))
	for _, n := range g.order {
		out = append(out, g.metas[n])
	}
	return out
}

// OpenStandalone loads a scene with its own fixture, as if it were the only
// scene in the game.
func (g *Game) OpenStandalone(name string) bool {
	m, ok := g.metas[name]
	if !ok {
		return false
	}
	g.Load(name, m.Fixture(g))
	return true
}

// RegionNames returns a layout's region names, sorted.
func (l Layout) RegionNames() []string {
	var n []string
	for k := range l.Regions {
		n = append(n, k)
	}
	sort.Strings(n)
	return n
}
