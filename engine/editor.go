package engine

import (
	"fmt"
	"sort"
)

// Bounder lets a component report its screen-space rectangle so the editor can
// pick and outline it. Objects without one are picked by their world position.
type Bounder interface {
	Bounds(o *GameObject, cam Camera) (x, y, w, h float64)
}

// Mover lets a component that owns its own screen coordinates follow a drag.
type Mover interface {
	MoveBy(o *GameObject, cam Camera, dx, dy float64)
}

// Editor turns a running scene into an editable one: pause and step the
// simulation, pick any GameObject (in the viewport or the hierarchy list), read
// its properties, drag it, hide it or delete it. It lives in the engine so every
// scene gets it for free and it can be tested headless.
type Editor struct {
	On       bool
	Paused   bool
	Selected *GameObject

	steps    int
	dragging bool
	lastPtr  Vec2
	scroll   int // first visible row of the hierarchy
}

const (
	edToolbarH = 40.0
	edRowH     = 22.0
	edRows     = 14
	edListW    = 190.0
	edInspW    = 230.0
)

var edButtons = []string{"Pause", "Step", "Hide", "Delete", "Close"}

// Rect is a plain screen rectangle.
type edRect struct{ x, y, w, h float64 }

func (r edRect) has(p Vec2) bool { return p.X >= r.x && p.X < r.x+r.w && p.Y >= r.y && p.Y < r.y+r.h }

func edButtonRect(i int) edRect { return edRect{8 + float64(i)*88, 0, 80, edToolbarH - 4} }

func (g *Game) listRect() edRect {
	return edRect{0, edToolbarH, edListW, edRowH*edRows + 52}
}
func (g *Game) inspRect() edRect { return edRect{g.W - edInspW, edToolbarH, edInspW, 190} }

// SetEditor switches editor mode on or off. Entering it pauses the game.
func (g *Game) SetEditor(on bool) {
	g.Ed.On = on
	g.Ed.dragging = false
	if on {
		g.Ed.Paused = true
	} else {
		g.Ed.Paused = false
		g.Ed.steps = 0
	}
}

// Select makes o the selected object (nil clears the selection).
func (g *Game) Select(o *GameObject) { g.Ed.Selected = o }

// Hierarchy lists the live objects of the scene, back to front.
func (g *Game) Hierarchy() []*GameObject {
	if g.scene == nil {
		return nil
	}
	var out []*GameObject
	for _, o := range g.scene.Objects {
		if !o.destroyed {
			out = append(out, o)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Layer < out[j].Layer })
	return out
}

// Pick returns the topmost object under the screen point p, or nil.
func (g *Game) Pick(p Vec2) *GameObject {
	h := g.Hierarchy()
	for i := len(h) - 1; i >= 0; i-- {
		if x, y, w, hh, ok := g.screenBounds(h[i]); ok && (edRect{x, y, w, hh}).has(p) {
			return h[i]
		}
	}
	return nil
}

// screenBounds is the object's pickable rectangle in screen pixels.
func (g *Game) screenBounds(o *GameObject) (x, y, w, h float64, ok bool) {
	for _, c := range o.comps {
		if b, isB := c.(Bounder); isB {
			x, y, w, h = b.Bounds(o, g.scene.Cam)
			return x, y, w, h, true
		}
	}
	if o.Pos == (Vec2{}) {
		return 0, 0, 0, 0, false
	}
	p := g.scene.Cam.ToScreen(o.Pos)
	return p.X - 14, p.Y - 14, 28, 28, true
}

// Inspect returns the selected object's properties as label, value pairs.
func (g *Game) Inspect() [][2]string {
	o := g.Ed.Selected
	if o == nil {
		return nil
	}
	rows := [][2]string{
		{"Name", o.Name}, {"Tag", o.Tag},
		{"Pos", fmt.Sprintf("%.2f, %.2f", o.Pos.X, o.Pos.Y)},
		{"Rot", fmt.Sprintf("%.2f", o.Rot)},
		{"Layer", fmt.Sprint(o.Layer)}, {"Active", fmt.Sprint(o.Active)},
	}
	for _, c := range o.comps {
		rows = append(rows, [2]string{"Comp", fmt.Sprintf("%T", c)})
	}
	return rows
}

// Move shifts the selected object by a screen-space delta.
func (g *Game) moveSelected(dx, dy float64) {
	o := g.Ed.Selected
	if o == nil {
		return
	}
	for _, c := range o.comps {
		if m, ok := c.(Mover); ok {
			m.MoveBy(o, g.scene.Cam, dx, dy)
			return
		}
	}
	z := g.scene.Cam.Zoom
	o.Pos = o.Pos.Add(Vec2{dx / z, -dy / z})
}

// Do runs a toolbar command by name; the on-screen buttons call it too.
func (g *Game) Do(cmd string) {
	e := &g.Ed
	switch cmd {
	case "Pause":
		e.Paused = !e.Paused
	case "Step":
		e.Paused = true
		e.steps++
	case "Hide":
		if e.Selected != nil {
			e.Selected.Active = !e.Selected.Active
		}
	case "Delete":
		if e.Selected != nil {
			e.Selected.Destroy()
			e.Selected = nil
		}
	case "Close":
		g.SetEditor(false)
	}
}

// editorPointer handles a pointer press, drag and release while editing.
func (g *Game) editorPointer(in Input, pressed, released bool) {
	e := &g.Ed
	p := in.Pointer
	if pressed {
		for i, name := range edButtons {
			if edButtonRect(i).has(p) {
				g.Do(name)
				return
			}
		}
		lr := g.listRect()
		if lr.has(p) {
			if p.Y >= lr.y+lr.h-32 { // scroll tiles
				if p.X < lr.w/2 {
					e.scroll = max(0, e.scroll-edRows)
				} else if e.scroll+edRows < len(g.Hierarchy()) {
					e.scroll += edRows
				}
				return
			}
			row := e.scroll + int((p.Y-lr.y-24)/edRowH)
			if h := g.Hierarchy(); row >= 0 && row < len(h) && p.Y >= lr.y+24 {
				e.Selected = h[row]
			}
			return
		}
		if g.inspRect().has(p) {
			return
		}
		e.Selected = g.Pick(p)
		e.dragging = e.Selected != nil
		e.lastPtr = p
		return
	}
	if e.dragging && in.Down {
		g.moveSelected(p.X-e.lastPtr.X, p.Y-e.lastPtr.Y)
		e.lastPtr = p
	}
	if released {
		e.dragging = false
	}
}

func (g *Game) drawEditor(r Renderer) {
	e := &g.Ed
	ink, dim, hot := RGB(235, 238, 245), RGB(150, 158, 175), RGB(90, 230, 150)
	panel := RGB(14, 18, 30).Alpha(225)
	// Outline every pickable object; the selection is brighter and thicker.
	for _, o := range g.Hierarchy() {
		x, y, w, h, ok := g.screenBounds(o)
		if !ok {
			continue
		}
		c, t := RGB(120, 170, 255).Alpha(110), 1.0
		if !o.Active {
			c = dim.Alpha(90)
		}
		if o == e.Selected {
			c, t = hot, 2
		}
		r.Line(V(x, y), V(x+w, y), t, c)
		r.Line(V(x+w, y), V(x+w, y+h), t, c)
		r.Line(V(x+w, y+h), V(x, y+h), t, c)
		r.Line(V(x, y+h), V(x, y), t, c)
	}
	// Toolbar.
	r.Rect(0, 0, g.W, edToolbarH, panel, true)
	for i, name := range edButtons {
		b := edButtonRect(i)
		label := name
		if name == "Pause" && e.Paused {
			label = "Play"
		}
		r.Rect(b.x, b.y+4, b.w, b.h-4, RGB(40, 52, 80), true)
		r.Text(label, b.x+b.w/2, b.y+12, 14, ink, AlignCenter)
	}
	state := "playing"
	if e.Paused {
		state = "paused"
	}
	r.Text("EDITOR · "+state, g.W-12, 12, 14, hot, AlignRight)

	// Hierarchy.
	lr := g.listRect()
	r.Rect(lr.x, lr.y, lr.w, lr.h, panel, true)
	r.Text(fmt.Sprintf("Hierarchy (%d)", len(g.Hierarchy())), lr.x+8, lr.y+4, 13, dim, AlignLeft)
	h := g.Hierarchy()
	for i := 0; i < edRows && e.scroll+i < len(h); i++ {
		o := h[e.scroll+i]
		y := lr.y + 24 + float64(i)*edRowH
		c := ink
		if o == e.Selected {
			r.Rect(lr.x+2, y-2, lr.w-4, edRowH, hot.Alpha(60), true)
			c = hot
		} else if !o.Active {
			c = dim
		}
		r.Text(fmt.Sprintf("%d  %s", o.Layer, o.Name), lr.x+8, y, 12, c, AlignLeft)
	}
	r.Text("< prev", lr.x+lr.w*0.25, lr.y+lr.h-24, 13, dim, AlignCenter)
	r.Text("next >", lr.x+lr.w*0.75, lr.y+lr.h-24, 13, dim, AlignCenter)

	// Inspector.
	ir := g.inspRect()
	r.Rect(ir.x, ir.y, ir.w, ir.h, panel, true)
	r.Text("Inspector", ir.x+8, ir.y+4, 13, dim, AlignLeft)
	rows := g.Inspect()
	if rows == nil {
		r.Text("Tap an object to select it.", ir.x+8, ir.y+28, 13, ink, AlignLeft)
	}
	for i, row := range rows {
		if i > 10 {
			break
		}
		y := ir.y + 24 + float64(i)*14
		r.Text(row[0], ir.x+8, y, 12, dim, AlignLeft)
		r.Text(trim(row[1], 26), ir.x+60, y, 12, ink, AlignLeft)
	}
}

func trim(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}
