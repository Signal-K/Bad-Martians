package game

import (
	"fmt"

	"github.com/signal-k/haphazard-martians/engine"
)

// ExpeditionLevels are the missions in one expedition, in order.
var ExpeditionLevels = []int{1, 2, 3}

// ExpeditionKit is every part the player gets for the whole expedition. There
// are no supply drops: this has to last all three missions.
var ExpeditionKit = Kit{KFrame: 24, KWheel: 18, KMotor: 9, KMast: 4}

// Kit counts parts by kind.
type Kit map[Kind]int

// KitOf counts the parts in a design.
func KitOf(d Design) Kit {
	k := Kit{}
	for _, c := range d.Cells {
		k[c.K]++
	}
	return k
}

// Total is the number of parts in the kit.
func (k Kit) Total() int {
	n := 0
	for _, v := range k {
		n += v
	}
	return n
}

// Expedition is one roguelike run through several missions. The rover's parts
// come from one kit handed out at the start: a finished mission keeps the
// rover, and a crash (or scrapping a stuck rover) loses every part on it. The
// expedition ends when the kit cannot build a rover that drives: no motor
// wheel or no Navcam mast left.
type Expedition struct {
	Index  int // current mission, an index into ExpeditionLevels
	Inv    Kit // parts left
	Score  int // total score of finished missions
	Done   bool
	Failed bool
	Lost   Kit // parts lost by the most recent crash

	Salvaged Kind   // part the most recent mission gave back, if any
	Name     string // the rover's name; it carries on while the rover survives
}

// NewExpedition hands out the full kit.
func NewExpedition() *Expedition {
	e := &Expedition{Inv: Kit{}}
	for k, n := range ExpeditionKit {
		e.Inv[k] = n
	}
	return e
}

// MinRover is the cost of the cheapest rover that can drive.
func MinRover() int {
	d := StarterDesign()
	return d.Cost()
}

func (e *Expedition) Level() *Level { return GetLevel(ExpeditionLevels[e.Index]) }

// Finish applies the outcome of an attempt with rover d.
func (e *Expedition) Finish(won bool, d Design, score int) {
	e.Lost, e.Salvaged = nil, ""
	if won {
		e.Score += score
		if e.Index+1 >= len(ExpeditionLevels) {
			e.Done = true
			return
		}
		e.Index++
		return
	}
	e.Lost = Kit{}
	for k, n := range KitOf(d) {
		n = min(n, e.Inv[k])
		e.Lost[k], e.Inv[k] = n, e.Inv[k]-n
	}
	if e.Inv[KMotor] < 1 || e.Inv[KMast] < 1 {
		e.Failed = true
	}
}

// Salvage puts one extra part from an undamaged rover back in the kit: its
// most valuable part other than the mast. Call it after a winning Finish.
func (e *Expedition) Salvage(d Design) Kind {
	k, ok := SalvagePart(d)
	if !ok {
		return ""
	}
	e.Inv[k]++
	e.Salvaged = k
	return k
}

// Over reports whether the expedition has ended, in success or failure.
func (e *Expedition) Over() bool { return e.Done || e.Failed }

// kitText lists a kit as "3 frames, 1 wheel" in tray order.
func kitText(k Kit) string {
	s := ""
	for _, kind := range []Kind{KFrame, KWheel, KMotor, KMast} {
		if k[kind] == 0 {
			continue
		}
		if s != "" {
			s += ", "
		}
		s += fmt.Sprintf("%d %s", k[kind], partNoun(kind, k[kind]))
	}
	return s
}

// partNoun is a part's short name, plural when n != 1.
func partNoun(k Kind, n int) string {
	w := map[Kind]string{KFrame: "frame", KWheel: "wheel", KMotor: "motor wheel", KMast: "mast"}[k]
	if n != 1 {
		w += "s"
	}
	return w
}

// ExpeditionSession starts the current mission of an expedition.
func ExpeditionSession(h engine.Host, e *Expedition) *Session {
	s := NewSessionFor(h, e.Level())
	s.Exp, s.Tut = e, nil
	s.Name = e.Name
	return s
}
