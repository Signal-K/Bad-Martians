package game

import "github.com/signal-k/haphazard-martians/engine"

// Design tokens. Every colour, size and weight the UI uses lives here so all
// scenes look like one game. The look: black icon tiles with white glyphs,
// thick white outlines, neon tab labels, a dark nebula behind everything.
var (
	colSpace   = engine.RGB(6, 7, 16)
	colTile    = engine.RGB(9, 9, 14)
	colInk     = engine.RGB(246, 246, 248) // lines, glyphs, body text
	colMuted   = engine.RGB(150, 160, 178) // secondary text
	colNeon    = engine.RGB(86, 236, 160)  // tab labels, meters, primary action
	colChip    = engine.RGB(58, 98, 200)   // resource chips
	colChipLt  = engine.RGB(118, 176, 214) // light resource chips
	colWarn    = engine.RGB(255, 96, 96)
	colGood    = engine.RGB(110, 230, 150)
	colStar    = engine.RGB(255, 208, 70)
	colStarOff = engine.RGB(70, 72, 86)
	colMars    = engine.RGB(236, 120, 60) // Mars accent, used sparingly
)

// Aliases kept for the gameplay drawing code written before the theme existed.
var (
	colText   = colInk
	colDim    = colMuted
	colPanel  = engine.RGB(20, 22, 32)
	colPanel2 = engine.RGB(34, 38, 54)
	colAccent = colMars
	colBad    = colWarn
)

const (
	lineW    = 3.0  // outline weight
	cutC     = 14.0 // chamfer on panel corners
	touchMin = 40.0 // smallest tappable edge, in virtual pixels
)

// Type sizes.
const (
	sizeTitle = 30.0
	sizeBody  = 17.0
	sizeSmall = 14.0
	sizeTab   = 16.0
)

// Virtual screen size every scene is laid out for. Hosts scale it to fit.
const (
	ScreenW = 960.0
	ScreenH = 540.0
)
