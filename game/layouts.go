package game

import "github.com/signal-k/haphazard-martians/engine"

// Layouts are the shared bones of the UI. A scene picks one and arranges its
// widgets inside the named regions.
var (
	// LayoutCenter: a title band over one wide content area.
	LayoutCenter = engine.Layout{Name: "center", Regions: map[string]engine.Rect{
		"title": {X: 0, Y: 0, W: ScreenW, H: 100},
		"main":  {X: 0, Y: 100, W: ScreenW, H: 440},
	}}
	// LayoutCargo: the Out There inventory look. A board on the left, a rail
	// of panels on the right, a footer for navigation and the primary action.
	LayoutCargo = engine.Layout{Name: "cargo", Regions: map[string]engine.Rect{
		"title":  {X: 0, Y: 0, W: ScreenW, H: 64},
		"board":  {X: 24, Y: 72, W: 610, H: 380},
		"rail":   {X: 650, Y: 72, W: 286, H: 380},
		"footer": {X: 0, Y: 460, W: ScreenW, H: 80},
	}}
	// LayoutHUD: the world fills the screen; a slim top bar and a control strip.
	LayoutHUD = engine.Layout{Name: "hud", Regions: map[string]engine.Rect{
		"hud":      {X: 0, Y: 0, W: ScreenW, H: 64},
		"main":     {X: 0, Y: 64, W: ScreenW, H: 476},
		"controls": {X: 0, Y: 430, W: ScreenW, H: 110},
	}}
)
