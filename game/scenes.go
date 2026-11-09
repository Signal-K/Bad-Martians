package game

import (
	"github.com/signal-k/haphazard-martians/engine"
)

// Scene names. Every scene below registers a type, a layout and a fixture.
const (
	SceneMenu   = "menu"
	SceneBuild  = "build"
	SceneRun    = "run"
	SceneResult = "result"
	SceneReview = "review"
	SceneStyle  = "style"
	// SceneBriefing hands out an expedition's kit before its first mission.
	SceneBriefing = "briefing"
)

// Setup registers every scene. Opening the menu is left to the host so any
// scene can be opened first (see OpenStandalone).
func Setup(g *engine.Game) {
	g.RegisterScene(engine.SceneMeta{
		Name: SceneMenu, Title: "Mission select", Type: engine.TypeMenu, Layout: LayoutCenter,
		Fixture: func(*engine.Game) any { return nil },
	}, menuScene)
	g.RegisterScene(engine.SceneMeta{
		Name: SceneBuild, Title: "Rover bay", Type: engine.TypeBuild, Layout: LayoutBay,
		Fixture: func(g *engine.Game) any { return NewSessionFor(g.Host, GetLevel(1)) },
	}, buildScene)
	g.RegisterScene(engine.SceneMeta{
		Name: SceneRun, Title: "Drive", Type: engine.TypePlay, Layout: LayoutHUD,
		Fixture: func(g *engine.Game) any { return NewSessionFor(g.Host, GetLevel(1)) },
	}, runScene)
	g.RegisterScene(engine.SceneMeta{
		Name: SceneResult, Title: "Mission report", Type: engine.TypeResults, Layout: LayoutCenter,
		Fixture: func(g *engine.Game) any { return sampleResult(g) },
	}, resultScene)
	g.RegisterScene(engine.SceneMeta{
		Name: SceneReview, Title: "Review the accuracy", Type: engine.TypeActivity, Layout: LayoutCenter,
		Fixture: func(g *engine.Game) any {
			return &ReviewArg{Session: NewSessionFor(g.Host, GetLevel(2)), Return: SceneMenu}
		},
	}, reviewScene)
	g.RegisterScene(engine.SceneMeta{
		Name: SceneBriefing, Title: "Expedition briefing", Type: engine.TypeMenu, Layout: LayoutCenter,
		Fixture: func(*engine.Game) any { return NewExpedition() },
	}, briefingScene)
	g.RegisterScene(engine.SceneMeta{
		Name: SceneStyle, Title: "Style sheet", Type: engine.TypeDebug, Layout: LayoutCenter,
		Fixture: func(*engine.Game) any { return nil },
	}, styleScene)
}

func newScene() *engine.Scene {
	s := &engine.Scene{}
	s.Cam = engine.Camera{Zoom: 1, Pos: engine.V(ScreenW/2, ScreenH/2)}
	return s
}

// addBackdrop draws the nebula on layer 0.
func addBackdrop(s *engine.Scene) {
	AddDraw(s, 0, func(_ engine.Camera, r engine.Renderer) { drawBackdrop(r) })
}

// addTitle draws the scene title (and a one-line subtitle) in the title region.
func addTitle(s *engine.Scene, text, sub string) {
	AddDraw(s, 9, func(_ engine.Camera, r engine.Renderer) {
		r.Text(text, 28, 16, sizeTitle, colInk, engine.AlignLeft)
		if sub != "" {
			r.Text(sub, 28, 54, sizeBody, colMuted, engine.AlignLeft)
		}
	})
}

// addBack adds the standard bottom-left back tile.
func addBack(s *engine.Scene, onClick func()) *Button {
	b := AddButton(s, 24, 470, 60, 60, "Back", onClick)
	b.Icon, b.HideLabel = IconBack, true
	return b
}
