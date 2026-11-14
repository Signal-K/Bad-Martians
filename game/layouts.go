package game

import "github.com/signal-k/haphazard-martians/engine"

// Layouts are the shared bones of the UI. A scene picks one and arranges its
// widgets inside the named regions.
var (
	// LayoutTravel separates the journey window from its action strip.
	LayoutTravel = engine.Layout{Name: "travel", Regions: map[string]engine.Rect{
		"title":   {X: 0, Y: 0, W: ScreenW, H: 100},
		"journey": {X: 0, Y: 100, W: ScreenW, H: 360},
		"actions": {X: 0, Y: 460, W: ScreenW, H: 80},
	}}
	// LayoutCenter: a title band over one wide content area.
	LayoutCenter = engine.Layout{Name: "center", Regions: map[string]engine.Rect{
		"title": {X: 0, Y: 0, W: ScreenW, H: 100},
		"main":  {X: 0, Y: 100, W: ScreenW, H: 440},
	}}
	// LayoutBay: the rover bay. A slim top bar, the build grid standing in the
	// world over the mission's real terrain, and a parts tray along the bottom.
	LayoutBay = engine.Layout{Name: "bay", Regions: map[string]engine.Rect{
		"title": {X: 0, Y: 0, W: ScreenW, H: 64},
		"bay":   {X: 0, Y: 64, W: ScreenW, H: 356},
		"tray":  {X: 0, Y: 420, W: ScreenW, H: 120},
	}}
	// LayoutHUD: the world fills the screen; a slim top bar and a control strip.
	LayoutHUD = engine.Layout{Name: "hud", Regions: map[string]engine.Rect{
		"hud":      {X: 0, Y: 0, W: ScreenW, H: 64},
		"main":     {X: 0, Y: 64, W: ScreenW, H: 476},
		"controls": {X: 0, Y: 430, W: ScreenW, H: 110},
	}}
)
