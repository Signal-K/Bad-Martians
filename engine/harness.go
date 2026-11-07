package engine

import (
	"encoding/binary"
	"hash/fnv"
	"math"
)

// Recorder is a headless Renderer. It hashes every draw call, so two runs of a
// scene can be compared bit for bit, and it counts what was drawn.
type Recorder struct {
	Ops, Texts int
	Strings    []string
	sum        interface {
		Write([]byte) (int, error)
		Sum64() uint64
	}
}

func NewRecorder() *Recorder { return &Recorder{sum: fnv.New64a()} }

func (r *Recorder) f(op byte, vals ...float64) {
	r.Ops++
	b := make([]byte, 1+8*len(vals))
	b[0] = op
	for i, v := range vals {
		binary.LittleEndian.PutUint64(b[1+8*i:], math.Float64bits(v))
	}
	r.sum.Write(b)
}

func col(c Color) float64 {
	return float64(uint32(c.R)<<24 | uint32(c.G)<<16 | uint32(c.B)<<8 | uint32(c.A))
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}

func (r *Recorder) Line(a, b Vec2, w float64, c Color) { r.f(1, a.X, a.Y, b.X, b.Y, w, col(c)) }
func (r *Recorder) Circle(p Vec2, rad float64, c Color, fill bool) {
	r.f(2, p.X, p.Y, rad, col(c), b2f(fill))
}
func (r *Recorder) Rect(x, y, w, h float64, c Color, fill bool) {
	r.f(3, x, y, w, h, col(c), b2f(fill))
}
func (r *Recorder) Poly(pts []Vec2, c Color) {
	r.f(4, float64(len(pts)), col(c))
	for _, p := range pts {
		r.f(5, p.X, p.Y)
	}
}
func (r *Recorder) Text(s string, x, y, size float64, c Color, a Align) {
	r.Texts++
	r.Strings = append(r.Strings, s)
	r.f(6, x, y, size, col(c), float64(a), float64(len(s)))
	r.sum.Write([]byte(s))
}

// Hash is the digest of everything drawn so far.
func (r *Recorder) Hash() uint64 { return r.sum.Sum64() }

// InputScript returns the input for a given frame of a headless run.
type InputScript func(frame int) Input

// RunFrames ticks the game for n 60 Hz frames, drawing each one into rec, and
// returns the digest of the whole run.
func RunFrames(g *Game, rec *Recorder, n int, script InputScript) uint64 {
	for i := 0; i < n; i++ {
		var in Input
		if script != nil {
			in = script(i)
		}
		g.Tick(FixedDT, in)
		g.Draw(rec)
	}
	return rec.Hash()
}
