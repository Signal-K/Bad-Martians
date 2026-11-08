package game

import (
	"fmt"
	"sort"

	"github.com/signal-k/haphazard-martians/engine"
	"github.com/signal-k/haphazard-martians/physics"
)

type Kind string

const (
	KFrame Kind = "frame"
	KWheel Kind = "wheel"
	KMotor Kind = "motor"
	KMast  Kind = "mast" // the Navcam mast: every rover must carry exactly one
)

type PartDef struct {
	Kind   Kind
	Name   string
	Cost   int
	Radius float64
	Mass   float64
	Wheel  bool
	Motor  bool
	Color  engine.Color
}

var Parts = []PartDef{
	{KFrame, "Frame", 1, 0.3, 1.0, false, false, engine.RGB(214, 214, 220)},
	{KWheel, "Wheel", 2, 0.5, 1.0, true, false, engine.RGB(60, 60, 70)},
	{KMotor, "Motor wheel", 4, 0.5, 1.4, true, true, engine.RGB(236, 120, 60)},
	{KMast, "Navcam mast", 2, 0.3, 1.5, false, false, engine.RGB(90, 170, 230)},
}

func PartOf(k Kind) (PartDef, bool) {
	for _, p := range Parts {
		if p.Kind == k {
			return p, true
		}
	}
	return PartDef{}, false
}

const (
	GridW = 12
	GridH = 5
)

type Cell struct {
	X, Y int
	K    Kind
}

// Design is a rover blueprint: parts on a grid, +Y up, 1 m pitch.
type Design struct {
	Cells []Cell `json:"cells"`
}

func (d *Design) At(x, y int) (Kind, bool) {
	for _, c := range d.Cells {
		if c.X == x && c.Y == y {
			return c.K, true
		}
	}
	return "", false
}

// Set places (or with k=="" removes) a part. Only one mast is allowed: placing
// another moves it.
func (d *Design) Set(x, y int, k Kind) {
	out := d.Cells[:0:0]
	for _, c := range d.Cells {
		if c.X == x && c.Y == y {
			continue
		}
		if k == KMast && c.K == KMast {
			continue
		}
		out = append(out, c)
	}
	if k != "" {
		out = append(out, Cell{x, y, k})
	}
	d.Cells = out
}

func (d *Design) Cost() int {
	t := 0
	for _, c := range d.Cells {
		if p, ok := PartOf(c.K); ok {
			t += p.Cost
		}
	}
	return t
}

// Validate returns player-facing problems; empty means launchable.
func (d *Design) Validate(budget int) []string {
	var errs []string
	count := map[Kind]int{}
	seen := map[[2]int]bool{}
	for _, c := range d.Cells {
		if c.X < 0 || c.X >= GridW || c.Y < 0 || c.Y >= GridH {
			errs = append(errs, fmt.Sprintf("Part outside the build grid at %d,%d", c.X, c.Y))
		}
		if _, ok := PartOf(c.K); !ok {
			errs = append(errs, "Unknown part "+string(c.K))
		}
		if seen[[2]int{c.X, c.Y}] {
			errs = append(errs, "Two parts share a cell")
		}
		seen[[2]int{c.X, c.Y}] = true
		count[c.K]++
	}
	if count[KMast] != 1 {
		errs = append(errs, "Add one Navcam mast")
	}
	if count[KMotor] == 0 {
		errs = append(errs, "Add a motor wheel")
	}
	if c := d.Cost(); c > budget {
		errs = append(errs, fmt.Sprintf("Over budget by %d", c-budget))
	}
	if len(d.Cells) > 0 && !d.connected() {
		errs = append(errs, "Parts must all touch each other")
	}
	return errs
}

func (d *Design) connected() bool {
	idx := map[[2]int]bool{}
	for _, c := range d.Cells {
		idx[[2]int{c.X, c.Y}] = true
	}
	start := [2]int{d.Cells[0].X, d.Cells[0].Y}
	seen := map[[2]int]bool{start: true}
	q := [][2]int{start}
	for len(q) > 0 {
		p := q[0]
		q = q[1:]
		for dx := -1; dx <= 1; dx++ {
			for dy := -1; dy <= 1; dy++ {
				n := [2]int{p[0] + dx, p[1] + dy}
				if idx[n] && !seen[n] {
					seen[n] = true
					q = append(q, n)
				}
			}
		}
	}
	return len(seen) == len(idx)
}

// StarterDesign is a cheap rover that can finish the first level.
func StarterDesign() Design {
	return Design{Cells: []Cell{
		{0, 0, KWheel}, {1, 0, KFrame}, {2, 0, KFrame}, {3, 0, KMotor},
		{1, 1, KFrame}, {2, 1, KMast},
	}}
}

// Rover is the component holding a spawned design's physics bodies.
type Rover struct {
	Cells  []Cell
	Bodies []*physics.Body
	Links  [][2]int // pairs of body indices joined by beams
	index  map[[2]int]int
}

// SpawnRover builds physics bodies for d with its lowest point resting just
// above groundY at world x = originX.
func SpawnRover(w *physics.World, d Design, originX, groundY float64) *Rover {
	cells := append([]Cell(nil), d.Cells...)
	sort.Slice(cells, func(i, j int) bool { // stable, order-independent build
		if cells[i].Y != cells[j].Y {
			return cells[i].Y < cells[j].Y
		}
		return cells[i].X < cells[j].X
	})
	r := &Rover{Cells: cells, index: map[[2]int]int{}}
	minBottom := 1e9
	for _, c := range cells {
		p, _ := PartOf(c.K)
		if b := float64(c.Y) - p.Radius; b < minBottom {
			minBottom = b
		}
	}
	for i, c := range cells {
		p, _ := PartOf(c.K)
		b := &physics.Body{
			Pos:     engine.V(originX+float64(c.X), groundY+0.05+float64(c.Y)-minBottom),
			Radius:  p.Radius,
			InvMass: 1 / p.Mass,
			Wheel:   p.Wheel, Motor: p.Motor,
			Friction: 0.4,
		}
		if p.Wheel {
			b.Friction = 1
		}
		w.AddBody(b)
		r.Bodies = append(r.Bodies, b)
		r.index[[2]int{c.X, c.Y}] = i
	}
	for i, c := range cells {
		for _, o := range [][2]int{{1, 0}, {0, 1}, {1, 1}, {-1, 1}} { // each pair once
			if j, ok := r.index[[2]int{c.X + o[0], c.Y + o[1]}]; ok {
				w.Connect(r.Bodies[i], r.Bodies[j], 1)
				r.Links = append(r.Links, [2]int{i, j})
			}
		}
	}
	return r
}

func (r *Rover) SetThrottle(t float64) {
	for _, b := range r.Bodies {
		if b.Motor {
			b.Throttle = t
		}
	}
}

func (r *Rover) Center() engine.Vec2 { return physics.Center(r.Bodies) }

// Mast returns the Navcam mast body.
func (r *Rover) Mast() *physics.Body {
	for i, c := range r.Cells {
		if c.K == KMast {
			return r.Bodies[i]
		}
	}
	return nil
}

func (r *Rover) Draw(g *engine.GameObject, cam engine.Camera, rd engine.Renderer) {
	z := cam.Zoom
	for _, l := range r.Links {
		drawBeam(rd, cam.ToScreen(r.Bodies[l[0]].Pos), cam.ToScreen(r.Bodies[l[1]].Pos), z)
	}
	for i, b := range r.Bodies {
		drawPart(rd, r.Cells[i].K, cam.ToScreen(b.Pos), z, b.Angle)
	}
}

func drawBeam(rd engine.Renderer, a, b engine.Vec2, z float64) {
	rd.Line(a, b, 0.16*z, engine.RGB(150, 156, 170))
}

// drawPart draws one part centred at c with z pixels per metre.
func drawPart(rd engine.Renderer, k Kind, c engine.Vec2, z, angle float64) {
	p, _ := PartOf(k)
	switch {
	case p.Wheel:
		rd.Circle(c, p.Radius*z, engine.RGB(36, 36, 44), true)
		rd.Circle(c, p.Radius*z*0.62, p.Color, true)
		for i := 0; i < 3; i++ {
			a := angle + float64(i)*2.0944
			d := engine.V(cosf(a), -sinf(a)).Mul(p.Radius * z * 0.9)
			rd.Line(c, c.Add(d), 0.07*z, engine.RGB(20, 20, 26))
		}
	case p.Kind == KMast:
		rd.Circle(c, p.Radius*z, p.Color, true)
		top := c.Add(engine.V(0, -0.5*z))
		rd.Line(c, top, 0.1*z, engine.RGB(190, 195, 205))
		rd.Rect(top.X-0.18*z, top.Y-0.22*z, 0.36*z, 0.22*z, engine.RGB(40, 44, 54), true)
		rd.Circle(top.Add(engine.V(0.06*z, -0.11*z)), 0.07*z, engine.RGB(120, 220, 255), true)
	default:
		rd.Circle(c, p.Radius*z*1.15, p.Color, true)
	}
}
