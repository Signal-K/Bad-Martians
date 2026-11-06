package game

import (
	"encoding/json"

	"github.com/signal-k/bad-martians/engine"
)

// Session is the data that travels between scenes for one attempt at a level.
type Session struct {
	Level     *Level
	Labels    map[int]Class // segment index -> player's label
	Design    Design
	Submitted bool // labels already sent to the host
}

func NewSession(l *Level) *Session {
	return &Session{Level: l, Labels: map[int]Class{}, Design: StarterDesign()}
}

// Bonus is the extra parts budget earned by labelling: +1 per label, +2 more
// when a gold-standard segment is labelled correctly.
func (s *Session) Bonus() int {
	b := 0
	for i, c := range s.Labels {
		b++
		if s.Level.Segments[i].Gold && s.Level.Segments[i].Class == c {
			b += 2
		}
	}
	return b
}

func (s *Session) Budget() int { return s.Level.Credits + s.Bonus() }

// SubmitLabels emits one "label" event per labelled segment (once).
func (s *Session) SubmitLabels(h engine.Host) {
	if s.Submitted {
		return
	}
	s.Submitted = true
	for i, c := range s.Labels {
		seg := s.Level.Segments[i]
		p, _ := json.Marshal(map[string]any{
			"level": s.Level.ID, "segment": i, "x0": seg.X0, "x1": seg.X1,
			"class": c.Info().Key, "gold": seg.Gold,
			"correct": seg.Gold && seg.Class == c,
		})
		h.Emit("label", p)
	}
}

type LevelProgress struct {
	Stars int     `json:"stars"`
	Best  float64 `json:"best"`
}

// Progress is the player's saved campaign state.
type Progress struct {
	Levels map[int]LevelProgress `json:"levels"`
}

const progressKey = "bm.progress"

func LoadProgress(h engine.Host) *Progress {
	p := &Progress{Levels: map[int]LevelProgress{}}
	if b := h.Load(progressKey); b != nil {
		_ = json.Unmarshal(b, p)
		if p.Levels == nil {
			p.Levels = map[int]LevelProgress{}
		}
	}
	return p
}

func (p *Progress) Save(h engine.Host) {
	b, _ := json.Marshal(p)
	h.Save(progressKey, b)
}

func (p *Progress) Unlocked(id int) bool {
	return id == LevelDefs[0].ID || p.Levels[id-1].Stars >= 1
}

// Record keeps the best stars/time.
func (p *Progress) Record(id, stars int, t float64) {
	cur := p.Levels[id]
	if stars > cur.Stars {
		cur.Stars = stars
	}
	if stars > 0 && (cur.Best == 0 || t < cur.Best) {
		cur.Best = t
	}
	p.Levels[id] = cur
}
