package game

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/signal-k/haphazard-martians/engine"
)

// A ghost is the player's fastest win on a level, replayed beside the live
// run. It is recorded as the rover's centre and tilt ten times a second (not
// the inputs, so a later physics retune cannot desync it) and lives only on
// this device, in the host's storage next to the progress.
const (
	ghostHz    = 10.0
	ghostScale = 100 // centimetres, and hundredths of a radian
)

type ghostPart struct {
	K Kind    `json:"k"`
	X float64 `json:"x"` // offset from the rover's centre as built
	Y float64 `json:"y"`
}

// Ghost is one recorded run.
type Ghost struct {
	Time  float64     `json:"t"` // finish time, seconds
	Parts []ghostPart `json:"p"`
	// S holds three numbers per sample (x, y, tilt), each stored as the change
	// from the previous sample so the JSON stays small.
	S []int32 `json:"s"`
}

func ghostKey(level int) string { return fmt.Sprintf("hm.ghost.%d", level) }

// GhostRecorder samples a Run as it plays.
type GhostRecorder struct {
	g          Ghost
	next       float64
	px, py, pa int32
	done       bool
}

func NewGhostRecorder(r *Run) *GhostRecorder {
	rec := &GhostRecorder{}
	for i, c := range r.Rover.Cells {
		o := r.built[i]
		rec.g.Parts = append(rec.g.Parts, ghostPart{c.K, math.Round(o.X*100) / 100, math.Round(o.Y*100) / 100})
	}
	return rec
}

// Observe takes a sample when one is due, and always on the step the run ends.
func (rec *GhostRecorder) Observe(r *Run) {
	if rec.done || (r.Time < rec.next && r.Outcome == Running) {
		return
	}
	rec.done = r.Outcome != Running
	rec.next = r.Time + 1/ghostHz
	c := r.Rover.Center()
	x, y := int32(math.Round(c.X*ghostScale)), int32(math.Round(c.Y*ghostScale))
	a := int32(math.Round(r.tilt() * math.Pi / 180 * ghostScale))
	rec.g.S = append(rec.g.S, x-rec.px, y-rec.py, a-rec.pa)
	rec.px, rec.py, rec.pa = x, y, a
}

// Ghost returns the recording, stamped with the finish time.
func (rec *GhostRecorder) Ghost(finish float64) *Ghost {
	g := rec.g
	g.Time = finish
	return &g
}

// at returns the centre and tilt at sim time t, holding the last pose after the finish.
func (g *Ghost) at(t float64) (c engine.Vec2, tilt float64) {
	n := len(g.S) / 3
	var x, y, a, px, py, pa float64
	f := t * ghostHz
	for i := 0; i < n; i++ {
		px, py, pa = x, y, a
		x += float64(g.S[3*i])
		y += float64(g.S[3*i+1])
		a += float64(g.S[3*i+2])
		if float64(i) >= f {
			k := 1 - (float64(i) - f)
			if i == 0 {
				k = 1
			}
			d := a - pa
			d -= 2 * math.Pi * ghostScale * math.Round(d/(2*math.Pi*ghostScale)) // shortest way round
			return engine.V((px+(x-px)*k)/ghostScale, (py+(y-py)*k)/ghostScale), (pa + d*k) / ghostScale
		}
	}
	return engine.V(x/ghostScale, y/ghostScale), a / ghostScale
}

func (g *Ghost) valid() bool { return g != nil && len(g.Parts) > 0 && len(g.S) >= 3 && len(g.S)%3 == 0 }

// Draw paints the ghost rover at sim time t as translucent discs.
func (g *Ghost) Draw(rd engine.Renderer, cam engine.Camera, t float64) {
	c, a := g.at(t)
	sin, cos := math.Sincos(a)
	for _, p := range g.Parts {
		def, ok := PartOf(p.K)
		if !ok {
			continue
		}
		w := engine.V(c.X+p.X*cos-p.Y*sin, c.Y+p.X*sin+p.Y*cos)
		rd.Circle(cam.ToScreen(w), def.Radius*cam.Zoom, def.Color.Alpha(95), true)
	}
}

// LoadGhost returns the saved ghost for a level, or nil.
func LoadGhost(h engine.Host, level int) *Ghost {
	b := h.Load(ghostKey(level))
	if b == nil {
		return nil
	}
	g := &Ghost{}
	if json.Unmarshal(b, g) != nil || !g.valid() {
		return nil
	}
	return g
}

// SaveGhost stores g as the level's ghost.
func SaveGhost(h engine.Host, level int, g *Ghost) {
	b, _ := json.Marshal(g)
	h.Save(ghostKey(level), b)
}
