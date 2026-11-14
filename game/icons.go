package game

import "github.com/signal-k/haphazard-martians/engine"

// Icon names a glyph in the shared icon set. All icons are drawn from
// primitives in one line weight, so they stay consistent at any size and need
// no image assets.
type Icon string

const (
	IconNone    Icon = ""
	IconWheel   Icon = "wheel"
	IconMotor   Icon = "motor"
	IconFrame   Icon = "frame"
	IconMast    Icon = "mast"
	IconErase   Icon = "erase"
	IconCredit  Icon = "credit"
	IconBack    Icon = "back"
	IconRight   Icon = "right"
	IconLeft    Icon = "left"
	IconClock   Icon = "clock"
	IconSample  Icon = "sample"
	IconReview  Icon = "review"
	IconGear    Icon = "gear"
	IconRover   Icon = "rover"
	IconMass    Icon = "mass"
	IconStarIco Icon = "star"
	IconRetry   Icon = "retry"
	IconCamera  Icon = "camera"
)

// AllIcons is the full set, for the style sheet and tests.
var AllIcons = []Icon{IconWheel, IconMotor, IconFrame, IconMast, IconErase, IconCredit, IconBack,
	IconRight, IconLeft, IconClock, IconSample, IconReview, IconGear, IconRover, IconMass, IconStarIco, IconRetry, IconCamera}

func partIcon(k Kind) Icon {
	switch k {
	case KWheel:
		return IconWheel
	case KMotor, KGripMotor:
		return IconMotor
	case KMast:
		return IconMast
	case KFrame:
		return IconFrame
	}
	return IconErase
}

func ring(r engine.Renderer, c engine.Vec2, rad float64, col engine.Color) {
	r.Circle(c, rad, col, false)
	r.Circle(c, rad-1.2, col, false)
}

// drawIcon draws icon ic centred at (cx,cy) in a box of s x s pixels.
func drawIcon(r engine.Renderer, ic Icon, cx, cy, s float64, c engine.Color) {
	p := func(x, y float64) engine.Vec2 { return engine.V(cx+x*s, cy+y*s) } // unit coords, -0.5..0.5
	lw := s * 0.09
	if lw < 2 {
		lw = 2
	}
	switch ic {
	case IconWheel, IconMotor:
		ring(r, p(0, 0), s*0.42, c)
		n := 6
		if ic == IconMotor {
			n = 4
		}
		for i := 0; i < n; i++ {
			a := float64(i) * 6.2832 / float64(n)
			r.Line(p(0, 0), p(0.38*cosf(a), 0.38*sinf(a)), lw*0.8, c)
		}
		if ic == IconMotor {
			r.Poly([]engine.Vec2{p(0.04, -0.24), p(-0.14, 0.03), p(-0.01, 0.03), p(-0.05, 0.25), p(0.14, -0.04), p(0.01, -0.04)}, c)
		} else {
			r.Circle(p(0, 0), s*0.08, c, true)
		}
	case IconFrame:
		r.Rect(cx-s*0.3, cy-s*0.3, s*0.6, s*0.6, c, false)
		r.Line(p(-0.3, -0.3), p(0.3, 0.3), lw*0.8, c)
		r.Line(p(0.3, -0.3), p(-0.3, 0.3), lw*0.8, c)
	case IconMast:
		r.Line(p(0, 0.42), p(0, -0.05), lw, c)
		r.Line(p(-0.22, 0.42), p(0.22, 0.42), lw, c)
		r.Rect(cx-s*0.26, cy-s*0.38, s*0.52, s*0.3, c, false)
		r.Circle(p(0.06, -0.23), s*0.07, c, true)
	case IconErase:
		r.Line(p(-0.3, -0.3), p(0.3, 0.3), lw*1.3, c)
		r.Line(p(0.3, -0.3), p(-0.3, 0.3), lw*1.3, c)
	case IconCredit:
		var h []engine.Vec2
		for i := 0; i < 6; i++ {
			a := 6.2832*float64(i)/6 + 0.5236
			h = append(h, p(0.42*cosf(a), 0.42*sinf(a)))
		}
		for i := range h {
			r.Line(h[i], h[(i+1)%6], lw, c)
		}
		r.Circle(p(0, 0), s*0.12, c, true)
	case IconBack, IconLeft:
		r.Poly([]engine.Vec2{p(-0.42, 0), p(-0.02, -0.34), p(-0.02, -0.14), p(0.42, -0.14), p(0.42, 0.14), p(-0.02, 0.14), p(-0.02, 0.34)}, c)
	case IconRight:
		r.Poly([]engine.Vec2{p(0.42, 0), p(0.02, -0.34), p(0.02, -0.14), p(-0.42, -0.14), p(-0.42, 0.14), p(0.02, 0.14), p(0.02, 0.34)}, c)
	case IconClock:
		ring(r, p(0, 0), s*0.42, c)
		r.Line(p(0, 0), p(0, -0.28), lw, c)
		r.Line(p(0, 0), p(0.2, 0.08), lw, c)
	case IconSample:
		r.Poly([]engine.Vec2{p(0, -0.42), p(0.3, 0), p(0, 0.42), p(-0.3, 0)}, c)
	case IconReview:
		ring(r, p(-0.08, -0.08), s*0.28, c)
		r.Line(p(0.12, 0.12), p(0.4, 0.4), lw*1.4, c)
	case IconGear:
		ring(r, p(0, 0), s*0.26, c)
		for i := 0; i < 8; i++ {
			a := float64(i) * 0.7854
			r.Line(p(0.3*cosf(a), 0.3*sinf(a)), p(0.44*cosf(a), 0.44*sinf(a)), lw*1.3, c)
		}
	case IconRover:
		r.Rect(cx-s*0.36, cy-s*0.2, s*0.72, s*0.22, c, false)
		r.Line(p(0, -0.2), p(0, -0.4), lw*0.8, c)
		r.Circle(p(0.04, -0.42), s*0.05, c, true)
		r.Circle(p(-0.24, 0.2), s*0.14, c, false)
		r.Circle(p(0.24, 0.2), s*0.14, c, false)
	case IconMass:
		r.Poly([]engine.Vec2{p(-0.2, -0.1), p(0.2, -0.1), p(0.4, 0.36), p(-0.4, 0.36)}, c)
		ring(r, p(0, -0.26), s*0.14, c)
	case IconStarIco:
		drawStar(r, cx, cy, s*0.45, c)
	case IconRetry: // an open ring with an arrowhead
		const n = 14
		prev := p(0.34*cosf(-0.9), 0.34*sinf(-0.9))
		for i := 1; i <= n; i++ {
			a := -0.9 + 4.9*float64(i)/n
			q := p(0.34*cosf(a), 0.34*sinf(a))
			r.Line(prev, q, lw, c)
			prev = q
		}
		a := -0.9
		tip := p(0.34*cosf(a), 0.34*sinf(a))
		r.Poly([]engine.Vec2{tip.Add(engine.V(-s*0.2, -s*0.06)), tip.Add(engine.V(s*0.1, -s*0.2)), tip.Add(engine.V(s*0.06, s*0.14))}, c)
	case IconCamera:
		r.Rect(cx-s*0.4, cy-s*0.22, s*0.8, s*0.54, c, false)
		r.Rect(cx-s*0.16, cy-s*0.34, s*0.32, s*0.12, c, true)
		ring(r, p(0, 0.05), s*0.16, c)
	}
}
