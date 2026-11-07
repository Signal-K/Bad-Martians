package game

import "github.com/signal-k/haphazard-martians/engine"

// paintTerrainSample draws a pseudo-photo of a terrain class in a panel. The
// look is deterministic per (level, segment) so players see a stable image.
func paintTerrainSample(r engine.Renderer, lvl *Level, idx int, x, y, w, h float64) {
	seg := lvl.Segments[idx]
	rg := newRng(lvl.Seed*131 + uint64(idx)*7919)
	base := Classes[seg.Class].Color
	j := func() uint8 { return uint8(rg.intn(24)) }
	base = engine.RGB(clamp8(int(base.R)+int(j())-12), clamp8(int(base.G)+int(j())-12), clamp8(int(base.B)+int(j())-12))
	if seg.Class == BigRock {
		base = Classes[Soil].Color
	}
	r.Rect(x, y, w, h, base, true)
	switch seg.Class {
	case Soil:
		for i := 0; i < 70; i++ {
			c := engine.RGB(clamp8(int(base.R)-30+rg.intn(60)), clamp8(int(base.G)-24+rg.intn(48)), clamp8(int(base.B)-20+rg.intn(40)))
			r.Circle(engine.V(x+rg.float()*w, y+rg.float()*h), 1+rg.float()*2.2, c, true)
		}
	case Sand:
		for i := 0; i < 9; i++ {
			yy := y + 8 + float64(i)*h/9
			prev := engine.V(x, yy)
			for xx := 6.0; xx <= w; xx += 6 {
				p := engine.V(x+xx, yy+3*sinf(xx*0.18+float64(i)*1.7))
				r.Line(prev, p, 1.4, lerpColor(base, engine.RGB(255, 226, 170), 0.5))
				prev = p
			}
		}
	case Bedrock:
		dark := lerpColor(base, engine.RGB(30, 24, 24), 0.55)
		for row := 0; row < 4; row++ {
			yy := y + float64(row)*h/4
			r.Line(engine.V(x, yy), engine.V(x+w, yy), 2, dark)
			off := rg.rangef(10, w-10)
			r.Line(engine.V(x+off, yy), engine.V(x+off+rg.rangef(-8, 8), yy+h/4), 2, dark)
		}
	case BigRock:
		for i := 0; i < 70; i++ {
			r.Circle(engine.V(x+rg.float()*w, y+rg.float()*h), 1+rg.float()*1.8, lerpColor(base, engine.RGB(40, 26, 20), 0.3), true)
		}
		for i := 0; i < 2+rg.intn(2); i++ {
			c := engine.V(x+14+rg.float()*(w-28), y+22+rg.float()*(h-44))
			rad := 12 + rg.float()*14
			r.Circle(c, rad, engine.RGB(70, 58, 58), true)
			r.Circle(c.Add(engine.V(-rad*0.25, -rad*0.3)), rad*0.55, engine.RGB(104, 90, 88), true)
		}
	}
}

func clamp8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}
