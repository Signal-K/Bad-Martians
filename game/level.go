package game

import (
	"math"

	"github.com/signal-k/haphazard-martians/engine"
	"github.com/signal-k/haphazard-martians/physics"
)

// Segment is a stretch of terrain with one AI4Mars class. Gold segments have
// known ground truth and are used to score (and calibrate) labellers; the rest
// are collected for crowd consensus.
type Segment struct {
	X0, X1 float64
	Class  Class
	Gold   bool
}

type LevelDef struct {
	ID       int
	Name     string
	Blurb    string
	Biome    string
	Length   float64
	Seed     uint64
	Credits  int     // base parts budget
	Par      float64 // seconds for the time star
	Hill     float64 // hill amplitude (m)
	Traction float64 // multiplies every class's friction; 1 is as labelled
	Mix      [NumClasses]float64
}

type Level struct {
	LevelDef
	Terrain  *physics.Terrain
	Segments []Segment
	Samples  []engine.Vec2
	StartX   float64
	GoalX    float64
}

const (
	terrainDX  = 0.5
	terrainPad = 12.0
)

// bump is a Gaussian-like hill (about exp(-d²)) built from multiplications
// only. math.Exp has a different implementation on amd64 than on wasm, which
// made the same level differ in its last bits between platforms; this does not.
// The explicit float64 conversion stops fused multiply-add on arm64.
func bump(d float64) float64 {
	u := 1 - float64(d*d)/8
	if u <= 0 {
		return 0
	}
	u = float64(u * u)
	u = float64(u * u)
	return float64(u * u)
}

func smooth(t float64) float64 {
	t = math.Max(0, math.Min(1, t))
	return t * t * (3 - 2*t)
}

// BuildLevel deterministically generates a level from its definition.
func BuildLevel(def LevelDef) *Level {
	r := newRng(def.Seed)
	L := &Level{LevelDef: def, StartX: 0, GoalX: def.Length}

	// Segments: 8 stretches, first/last safe.
	const nSeg = 8
	cuts := make([]float64, nSeg+1)
	cuts[0], cuts[nSeg] = -terrainPad, def.Length+terrainPad
	span := def.Length + 2*terrainPad
	for i := 1; i < nSeg; i++ {
		base := -terrainPad + span*float64(i)/nSeg
		cuts[i] = math.Round((base+r.rangef(-3, 3))/terrainDX) * terrainDX
	}
	total := 0.0
	for _, m := range def.Mix {
		total += m
	}
	prev := Class(255)
	for i := 0; i < nSeg; i++ {
		var c Class
		switch {
		case i == 0:
			c = Soil
		case i == nSeg-1:
			c = Bedrock
		default:
			for tries := 0; ; tries++ {
				x := r.float() * total
				for k, m := range def.Mix {
					if x < m {
						c = Class(k)
						break
					}
					x -= m
				}
				if c != prev || tries > 8 {
					break
				}
			}
		}
		prev = c
		L.Segments = append(L.Segments, Segment{X0: cuts[i], X1: cuts[i+1], Class: c, Gold: i%3 == 1})
	}

	// Heightfield.
	n := int(span/terrainDX) + 1
	h := make([]float64, n)
	p1, p2, p3 := r.rangef(0, 6.28), r.rangef(0, 6.28), r.rangef(0, 6.28)
	for i := range h {
		x := -terrainPad + float64(i)*terrainDX
		ramp := smooth((x-12)/22) * (1 - smooth((x-(def.Length-18))/14))
		h[i] = ramp * def.Hill * (math.Sin(x*0.07+p1) + 0.5*math.Sin(x*0.13+p2) + 0.25*math.Sin(x*0.29+p3))
	}
	// Class-specific micro-relief, added on top (never in the safe ends).
	for _, s := range L.Segments {
		switch s.Class {
		case Sand:
			for i := range h {
				x := -terrainPad + float64(i)*terrainDX
				if x > s.X0 && x < s.X1 {
					edge := math.Min(x-s.X0, s.X1-x)
					h[i] += 0.12 * math.Sin(x*1.1) * smooth(edge/3)
				}
			}
		case BigRock:
			for x := s.X0 + 3; x < s.X1-3; x += r.rangef(3, 5.5) {
				ph, pw := r.rangef(0.3, 0.7), r.rangef(0.6, 1.1)
				for i := range h {
					xx := -terrainPad + float64(i)*terrainDX
					d := (xx - x) / pw
					h[i] += ph * bump(d)
				}
			}
		}
	}

	seg := make([]uint8, n-1)
	for i := range seg {
		x := -terrainPad + (float64(i)+0.5)*terrainDX
		for _, s := range L.Segments {
			if x >= s.X0 && x < s.X1 {
				seg[i] = uint8(s.Class)
			}
		}
	}
	mats := make([]physics.Material, NumClasses)
	for i := range mats {
		mats[i] = Classes[i].Mat
		if def.Traction > 0 {
			mats[i].Friction *= def.Traction
		}
	}
	L.Terrain = &physics.Terrain{X0: -terrainPad, DX: terrainDX, H: h, Seg: seg, Materials: mats}

	// Samples hang at different heights: the high one rewards a taller rover.
	for i, f := range []float64{0.3, 0.55, 0.8} {
		x := def.Length * f
		L.Samples = append(L.Samples, engine.V(x, L.Terrain.HeightAt(x)+[]float64{1.4, 3.2, 1.4}[i]))
	}
	return L
}

func GetLevel(id int) *Level {
	for _, d := range LevelDefs {
		if d.ID == id {
			return BuildLevel(d)
		}
	}
	for _, m := range Missions {
		if m.Level.ID == id {
			return BuildLevel(m.Level)
		}
	}
	return nil
}

// SegmentAt returns the index of the segment containing x.
func (l *Level) SegmentAt(x float64) int {
	for i, s := range l.Segments {
		if x >= s.X0 && x < s.X1 {
			return i
		}
	}
	return len(l.Segments) - 1
}
