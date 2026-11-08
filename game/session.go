package game

import (
	"encoding/json"

	"github.com/signal-k/haphazard-martians/engine"
)

// Session is the data that travels between scenes for one attempt at a level.
type Session struct {
	Level   *Level
	Labels  map[int]Class // segment index -> player's review label
	Design  Design
	emitted map[int]Class // labels already sent to the host
	Reward  int           // extra parts credits from a mission's reward
}

func NewSession(l *Level) *Session {
	return &Session{Level: l, Labels: map[int]Class{}, Design: StarterDesign(), emitted: map[int]Class{}}
}

// NewSessionFor starts a session that remembers the player's earlier reviews of
// this level, so credits earned by reviewing carry into new builds.
func NewSessionFor(h engine.Host, l *Level) *Session {
	s := NewSession(l)
	for _, m := range Missions {
		if m.Level.ID == l.ID {
			s.Reward = m.Reward.Credits
		}
	}
	for seg, c := range LoadProgress(h).Labels[l.ID] {
		s.Labels[seg], s.emitted[seg] = Class(c), Class(c)
	}
	return s
}

// Bonus is the extra parts budget earned by reviewing: +1 per label, +2 more
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

func (s *Session) Budget() int { return s.Level.Credits + s.Reward + s.Bonus() }

// SubmitLabels sends each new or changed label to the host and remembers the
// labels in the player's progress.
func (s *Session) SubmitLabels(h engine.Host) {
	for i, c := range s.Labels {
		if prev, ok := s.emitted[i]; ok && prev == c {
			continue
		}
		s.emitted[i] = c
		seg := s.Level.Segments[i]
		p, _ := json.Marshal(map[string]any{
			"level": s.Level.ID, "segment": i, "x0": seg.X0, "x1": seg.X1,
			"class": c.Info().Key, "gold": seg.Gold,
			"correct": seg.Gold && seg.Class == c,
		})
		h.Emit("label", p)
	}
	prog := LoadProgress(h)
	if prog.Labels == nil {
		prog.Labels = map[int]map[int]int{}
	}
	m := map[int]int{}
	for i, c := range s.Labels {
		m[i] = int(c)
	}
	prog.Labels[s.Level.ID] = m
	prog.Save(h)
}

type LevelProgress struct {
	Stars int     `json:"stars"`
	Best  float64 `json:"best"`  // fastest winning time
	Score int     `json:"score"` // best score
}

// Progress is the player's saved campaign state.
type Progress struct {
	Levels map[int]LevelProgress `json:"levels"`
	Labels map[int]map[int]int   `json:"labels"` // level -> segment -> class
}

const progressKey = "hm.progress"

func LoadProgress(h engine.Host) *Progress {
	p := &Progress{Levels: map[int]LevelProgress{}, Labels: map[int]map[int]int{}}
	if b := h.Load(progressKey); b != nil {
		_ = json.Unmarshal(b, p)
		if p.Levels == nil {
			p.Levels = map[int]LevelProgress{}
		}
		if p.Labels == nil {
			p.Labels = map[int]map[int]int{}
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

// Record keeps the best stars, time and score.
func (p *Progress) Record(id, stars int, t float64, score int) {
	cur := p.Levels[id]
	if stars > cur.Stars {
		cur.Stars = stars
	}
	if stars > 0 && (cur.Best == 0 || t < cur.Best) {
		cur.Best = t
	}
	if score > cur.Score {
		cur.Score = score
	}
	p.Levels[id] = cur
}
