package game

import "github.com/signal-k/haphazard-martians/engine"

// ExpeditionLevels are the missions in one expedition, in order.
var ExpeditionLevels = []int{1, 2, 3}

// Expedition is one roguelike run through several missions. The rover's parts
// are a shared inventory: every mission drops in some supplies, a finished
// mission keeps the rover, and a crash loses every part used in that attempt.
// The expedition ends when the inventory cannot pay for the cheapest rover.
type Expedition struct {
	Index  int // current mission, an index into ExpeditionLevels
	Parts  int // parts in the inventory
	Score  int // total score of finished missions
	Done   bool
	Failed bool
	Lost   int // parts lost by the most recent crash
}

// NewExpedition starts with the first mission's supplies in the inventory.
func NewExpedition() *Expedition {
	e := &Expedition{}
	e.Parts = GetLevel(ExpeditionLevels[0]).Credits
	return e
}

// MinRover is the cost of the cheapest rover that can drive.
func MinRover() int {
	d := StarterDesign()
	return d.Cost()
}

func (e *Expedition) Level() *Level { return GetLevel(ExpeditionLevels[e.Index]) }

// Finish applies the outcome of an attempt. cost is the parts the rover used.
func (e *Expedition) Finish(won bool, cost, score int) {
	e.Lost = 0
	if won {
		e.Score += score
		if e.Index+1 >= len(ExpeditionLevels) {
			e.Done = true
			return
		}
		e.Index++
		e.Parts += e.Level().Credits // supply drop at the next mission
		return
	}
	e.Lost = min(cost, e.Parts)
	e.Parts -= e.Lost
	if e.Parts < MinRover() {
		e.Failed = true
	}
}

// Over reports whether the expedition has ended, in success or failure.
func (e *Expedition) Over() bool { return e.Done || e.Failed }

// ExpeditionSession starts the current mission of an expedition.
func ExpeditionSession(h engine.Host, e *Expedition) *Session {
	s := NewSessionFor(h, e.Level())
	s.Exp = e
	return s
}
