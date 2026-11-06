package engine

// Color is straight (non-premultiplied) RGBA.
type Color struct{ R, G, B, A uint8 }

func RGB(r, g, b uint8) Color       { return Color{r, g, b, 255} }
func (c Color) Alpha(a uint8) Color { c.A = a; return c }

type Align int

const (
	AlignLeft Align = iota
	AlignCenter
	AlignRight
)

// Renderer is everything a game needs to draw. Coordinates are screen pixels
// in the game's virtual resolution (Y down). Hosts implement it.
type Renderer interface {
	Line(a, b Vec2, width float64, c Color)
	Circle(center Vec2, r float64, c Color, fill bool)
	Rect(x, y, w, h float64, c Color, fill bool)
	Poly(pts []Vec2, c Color) // filled convex-or-simple polygon
	Text(s string, x, y, size float64, c Color, a Align)
}

// Camera maps world metres (Y up) to screen pixels (Y down).
type Camera struct {
	Pos    Vec2    // world point at screen centre
	Zoom   float64 // pixels per metre
	Screen Vec2    // virtual screen size
}

func (c Camera) ToScreen(p Vec2) Vec2 {
	return Vec2{c.Screen.X/2 + (p.X-c.Pos.X)*c.Zoom, c.Screen.Y/2 - (p.Y-c.Pos.Y)*c.Zoom}
}
func (c Camera) ToWorld(p Vec2) Vec2 {
	return Vec2{c.Pos.X + (p.X-c.Screen.X/2)/c.Zoom, c.Pos.Y - (p.Y-c.Screen.Y/2)/c.Zoom}
}
