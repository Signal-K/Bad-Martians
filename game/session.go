package game

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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
	Lesson  int           // 1-based Shape school lesson; 0 outside the school

	// Last is why the last drive failed, shown in the bay with the parts to
	// blame (LastCells) until the player changes the design it was about.
	Last       Cause
	LastCells  []Cell
	lastDesign string
}

// NoteFailure keeps why run failed so the bay can explain it; a win clears it.
func (s *Session) NoteFailure(r *Run) {
	s.Last, s.LastCells, s.lastDesign = Cause{}, nil, ""
	if r.Outcome == Won || (r.Outcome == Running && !r.IsStuck()) {
		return
	}
	s.Last = r.Diagnose()
	for _, i := range s.Last.Culprits {
		s.LastCells = append(s.LastCells, r.Rover.Cells[i])
	}
	s.lastDesign = designKey(s.Design)
}

// LastFailure is the last drive's cause while the design is still the one
// that failed; empty once the player has changed it.
func (s *Session) LastFailure() (Cause, []Cell) {
	if s.Last.Empty() || designKey(s.Design) != s.lastDesign {
		return Cause{}, nil
	}
	return s.Last, s.LastCells
}

func designKey(d Design) string {
	c := append([]Cell(nil), d.Cells...)
	sort.Slice(c, func(i, j int) bool { return c[i].Y*GridW+c[i].X < c[j].Y*GridW+c[j].X })
	return fmt.Sprint(c)
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
		if i >= 0 && i < len(s.Level.Segments) && s.Level.Segments[i].Gold && s.Level.Segments[i].Class == c {
			b += 2
		}
	}
	return b
}

// Budget is the credits a rover may cost, including the shared expedition budget.
func (s *Session) Budget() int {
	if s.Exp != nil {
		return s.Exp.Credits
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

// CanAdd enforces both the credit budget and mission part limits.
func (s *Session) CanAdd(k Kind) int {
	n := CanAdd(s.Design, k, s.Budget())
	if m := s.Level.MaxParts; m > 0 {
		n = min(n, max(0, m-len(s.Design.Cells)))
	}
	if m := s.Level.MaxMotor; m > 0 && k == KMotor {
		n = min(n, max(0, m-KitOf(s.Design)[KMotor]))
	}
	return n
}

func (s *Session) Fits() bool { return s.Design.Cost() <= s.Budget() }

func (s *Session) Problems() []string {
	errs := s.Design.Validate(s.Budget())
	return append(errs, RuleProblems(s.Level.LevelDef, s.Design)...)
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

// Span is how many grid columns a design covers.
func (d *Design) Span() int {
	if len(d.Cells) == 0 {
		return 0
	}
	lo, hi := GridW, -1
	for _, c := range d.Cells {
		lo, hi = min(lo, c.X), max(hi, c.X)
	}
	return hi - lo + 1
}

// RuleProblems is what breaks a level's lander rules: the bay width and the
// parts it can carry and the motors its power can drive.
func RuleProblems(def LevelDef, d Design) []string {
	var errs []string
	if w := def.BayWidth(); w < GridW {
		for _, c := range d.Cells {
			if c.X >= w {
				errs = append(errs, fmt.Sprintf("The lander bay is only %d columns wide", w))
				break
			}
		}
	}
	if def.MaxParts > 0 && len(d.Cells) > def.MaxParts {
		errs = append(errs, fmt.Sprintf("The lander carries %d parts; this rover has %d", def.MaxParts, len(d.Cells)))
	}
	if n := KitOf(d)[KMotor]; def.MaxMotor > 0 && n > def.MaxMotor {
		errs = append(errs, fmt.Sprintf("Power for %d %s; this rover has %d", def.MaxMotor, partNoun(KMotor, def.MaxMotor), n))
	}
	return errs
}

// RuleLine is the level's lander rules for the build screen, or "".
func RuleLine(def LevelDef) string {
	var p []string
	if def.MaxParts > 0 {
		p = append(p, fmt.Sprintf("%d parts at most", def.MaxParts))
	}
	if def.MaxMotor > 0 {
		p = append(p, fmt.Sprintf("power for %d %s", def.MaxMotor, partNoun(KMotor, def.MaxMotor)))
	}
	if w := def.BayWidth(); w < GridW {
		p = append(p, fmt.Sprintf("bay %d columns wide", w))
	}
	if def.Gaps > 0 {
		w := def.GapWidth
		if w <= 0 {
			w = 2.5
		}
		n := "crevasse"
		if def.Gaps > 1 {
			n = "crevasses"
		}
		p = append(p, fmt.Sprintf("%d %s %.1f m wide", def.Gaps, n, w))
	}
	if len(p) == 0 {
		return ""
	}
	return "Lander: " + strings.Join(p, " · ")
}
