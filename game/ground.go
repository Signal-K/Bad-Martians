package game

import (
	"math"
	"sync"

	"github.com/signal-k/haphazard-martians/engine"
	"github.com/signal-k/haphazard-martians/physics"
)

// Ground is the cross-section under a level's surface: what a drill would hit
// going down at any x. The surface crust takes its stretch's class (soil,
// sand, bedrock, boulders), and under it run layers shared by the whole level:
// a pebble bed, packed soil, then rock. Layer depths wander smoothly along x,
// and the crust colour fades from one stretch into the next, so there are no
// hard vertical cuts between classes.
//
// It is presentation and level data only: physics never reads it, so the
// drive handles exactly as before. Ground is deterministic per terrain.
type Ground struct {
	// Per terrain sample (len(t.H)): crust colour and the depth, in metres
	// below the surface, where each layer ends.
	Crust               []engine.Color
	CrustD, PebD, RockD []float64
	// Finds are interesting rocks set into the ground, for sampling tasks.
	Finds []Find
}

// Layer is one band of the ground.
type Layer uint8

const (
	LayerCrust Layer = iota
	LayerPebbles
	LayerSoil
	LayerRock
)

// Find is an interesting rock: a mineral-veined stone in the crust or pebble
// bed that a drill or arm could sample.
type Find struct {
	X, Depth float64 // position: metres along the level, metres below the surface
	Size     float64 // radius in metres
	Layer    Layer
	Vein     engine.Color
}

var (
	colPebbleBed = engine.RGB(150, 104, 76)
	colPacked    = engine.RGB(132, 84, 58)
	colRock      = engine.RGB(104, 88, 88)
	findVeins    = []engine.Color{engine.RGB(120, 214, 220), engine.RGB(236, 206, 92), engine.RGB(214, 120, 230)}
)

// Per class: crust thickness and how deep rock starts, in metres.
var classGround = [NumClasses]struct{ crust, rock float64 }{
	Soil:    {0.55, 2.8},
	Bedrock: {0.12, 0.2},
	Sand:    {1.15, 3.4},
	BigRock: {0.35, 1.3},
}

// blendHalf is how many samples either side of a point feed its crust colour
// and depths: 6 × 0.5 m = a 3 m fade each way.
const blendHalf = 6

var groundCache sync.Map // *physics.Terrain → *Ground

// GroundOf returns the (cached) ground under t.
func GroundOf(t *physics.Terrain) *Ground {
	if g, ok := groundCache.Load(t); ok {
		return g.(*Ground)
	}
	g := BuildGround(t)
	groundCache.Store(t, g)
	return g
}

// BuildGround computes the ground for a terrain.
func BuildGround(t *physics.Terrain) *Ground {
	n := t.Len()
	g := &Ground{
		Crust:  make([]engine.Color, n),
		CrustD: make([]float64, n),
		PebD:   make([]float64, n),
		RockD:  make([]float64, n),
	}
	cls := func(i int) Class {
		if len(t.Seg) == 0 {
			return Soil
		}
		i = max(0, min(i, len(t.Seg)-1))
		return Class(t.Seg[i])
	}
	seed := uint64(n)*977 + uint64(math.Float64bits(t.X0)>>40)
	for i := 0; i < n; i++ {
		x := t.X0 + float64(i)*t.DX
		var r, gg, b, crust, rock, wsum float64
		for k := -blendHalf; k <= blendHalf; k++ {
			// smooth bell: full weight at the centre, zero at the window edge
			f := float64(k) / float64(blendHalf+1)
			w := (1 - f*f) * (1 - f*f)
			c := cls(i + k)
			col := Classes[c].Color
			r += float64(col.R) * w
			gg += float64(col.G) * w
			b += float64(col.B) * w
			crust += classGround[c].crust * w
			rock += classGround[c].rock * w
			wsum += w
		}
		g.Crust[i] = engine.RGB(uint8(r/wsum+0.5), uint8(gg/wsum+0.5), uint8(b/wsum+0.5))
		cd := crust/wsum + 0.18*(noise1(x*0.35, seed)-0.5)
		cd = math.Max(0.08, cd)
		rd := rock/wsum + 1.2*(noise1(x*0.06, seed+1)-0.5)
		pd := cd + 0.2 + 0.6*noise1(x*0.09, seed+2)
		rd = math.Max(rd, cd+0.05)
		pd = math.Min(pd, rd)
		g.CrustD[i], g.PebD[i], g.RockD[i] = cd, pd, rd
	}
	g.Finds = placeFinds(t, g, seed)
	return g
}

// LayerAt is the layer at depth d metres below the surface at sample i.
func (g *Ground) LayerAt(i int, d float64) Layer {
	switch {
	case d < g.CrustD[i]:
		return LayerCrust
	case d < g.PebD[i]:
		return LayerPebbles
	case d < g.RockD[i]:
		return LayerSoil
	}
	return LayerRock
}

// placeFinds sets two or three interesting rocks along the playable stretch,
// aimed at the crust or pebble bed (rock where those are thin under
// bedrock), never in the first or last 10 m.
func placeFinds(t *physics.Terrain, g *Ground, seed uint64) []Find {
	rg := newRng(seed ^ 0xF1D5)
	x0 := t.X0 + 12 + 10
	x1 := t.X0 + float64(t.Len()-1)*t.DX - 12 - 10
	if x1 <= x0 {
		return nil
	}
	count := 2 + rg.intn(2)
	var out []Find
	for k := 0; k < count; k++ {
		x := x0 + (x1-x0)*(float64(k)+rg.rangef(0.2, 0.8))/float64(count)
		i := int((x - t.X0) / t.DX)
		top, bot := g.CrustD[i], g.PebD[i]
		if bot-top < 0.3 || rg.float() < 0.4 {
			top, bot = 0.1, g.CrustD[i]
		}
		size := rg.rangef(0.22, 0.3)
		d := top + (bot-top)*0.5
		d = math.Max(d, size*0.6) // never poking out of the surface
		out = append(out, Find{X: x, Depth: d, Size: size, Layer: g.LayerAt(i, d), Vein: findVeins[rg.intn(len(findVeins))]})
	}
	return out
}

// noise1 is smooth 1D value noise in [0,1].
func noise1(x float64, seed uint64) float64 {
	i := math.Floor(x)
	f := x - i
	f = f * f * (3 - 2*f)
	h := func(k float64) float64 { return newRng(seed*0x2545F491 + uint64(int64(k))).float() }
	return h(i) + (h(i+1)-h(i))*f
}
