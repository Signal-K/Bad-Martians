package game

import (
	"encoding/json"
	"fmt"

	"github.com/signal-k/haphazard-martians/engine"
)

// Session is the data that travels between scenes for one attempt at a level.
type Session struct {
	Level   *Level
	Labels  map[int]Class // segment index -> player's review label
	Design  Design
	emitted map[int]Class // labels already sent to the host
	Reward  int           // extra parts credits from a mission's reward
	Exp     *Expedition   // set when this attempt is part of an expedition
	Tut     *Tutorial     // first-time guided build and drive; nil when off
	Name    string        // the rover's name, given at launch
	Spare   Kind          // part salvaged on the mission before: its credits add to the budget
}

func NewSession(l *Level) *Session {
	return &Session{Level: l, Labels: map[int]Class{}, emitted: map[int]Class{}}
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
	// Every mission's bay starts empty: the player builds the rover. A
	// first-time player is walked through it step by step.
	if l.ID == LevelDefs[0].ID && !TutorialDone(h) {
		s.Tut = &Tutorial{}
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

// Budget is the credits a rover may cost. Expeditions count parts instead
// (see Have), so their credit budget is unlimited.
func (s *Session) Budget() int {
	if s.Exp != nil {
		return 1 << 20
	}
	return s.Level.Credits + s.Reward + s.Bonus() + s.SpareCredits()
}

// SpareCredits is what the part salvaged on the previous mission is worth.
func (s *Session) SpareCredits() int {
	if s.Spare == "" {
		return 0
	}
	p, _ := PartOf(s.Spare)
	return p.Cost
}

// Have is how many of part k this build may use in an expedition: what is
// left in the kit, plus a spare frame for every review credit. -1 outside an
// expedition, where credits are the limit.
func (s *Session) Have(k Kind) int {
	if s.Exp == nil {
		return -1
	}
	n := s.Exp.Inv[k]
	if k == KFrame {
		n += s.Bonus()
	}
	return n
}

// CanAdd is how many more of part k this rover can take: the tray's number.
func (s *Session) CanAdd(k Kind) int {
	if s.Exp == nil {
		return CanAdd(s.Design, k, s.Budget())
	}
	n := max(0, s.Have(k)-KitOf(s.Design)[k])
	if k == KMast {
		n = min(n, CanAdd(s.Design, k, s.Budget()))
	}
	return n
}

// Fits reports whether the rover is within the credits or the kit.
func (s *Session) Fits() bool {
	if s.Design.Cost() > s.Budget() {
		return false
	}
	if s.Exp != nil {
		for k, n := range KitOf(s.Design) {
			if n > s.Have(k) {
				return false
			}
		}
	}
	return true
}

// Problems is what stops this rover launching, for the player.
func (s *Session) Problems() []string {
	errs := s.Design.Validate(s.Budget())
	if s.Exp != nil {
		used := KitOf(s.Design)
		for _, k := range []Kind{KFrame, KWheel, KMotor, KMast} {
			if have := s.Have(k); used[k] > have {
				errs = append(errs, fmt.Sprintf("Only %d %s left in the kit", have, partNoun(k, have)))
			}
		}
	}
	return errs
}

// SubmitReview sends the labels like SubmitLabels, then one "review" event
// with every label of the strip, so the host can keep the whole review with
// its photos (named with stamp, see reviewScene).
func (s *Session) SubmitReview(h engine.Host, stamp string) {
	s.SubmitLabels(h)
	type label struct {
		Segment int     `json:"segment"`
		X0      float64 `json:"x0"`
		X1      float64 `json:"x1"`
		Class   string  `json:"class"`
		Gold    bool    `json:"gold"`
		Correct bool    `json:"correct"`
	}
	labels := []label{}
	for i := range s.Level.Segments {
		c, ok := s.Labels[i]
		if !ok {
			continue
		}
		seg := s.Level.Segments[i]
		labels = append(labels, label{i, seg.X0, seg.X1, c.Info().Key, seg.Gold, seg.Gold && seg.Class == c})
	}
	p, _ := json.Marshal(map[string]any{
		"level": s.Level.ID, "seed": s.Level.Seed, "segments": len(s.Level.Segments),
		"labels": labels, "stamp": stamp,
	})
	h.Emit("review", p)
}

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

	BestExpedition int `json:"best_expedition"`
	Launched       int `json:"launched"` // rovers sent out; picks the next suggested name
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
