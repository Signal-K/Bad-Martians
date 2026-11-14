package game

import (
	"encoding/json"

	"github.com/signal-k/haphazard-martians/engine"
)

// ExpeditionLevels are the missions in one expedition, in order.
var ExpeditionLevels = []int{1, 2, 3}

// ExpeditionStartingCredits funds builds and retries across all three missions.
const ExpeditionStartingCredits = 36

// UnderParCredits is awarded only when a non-final stage is finished faster than par.
const UnderParCredits = 2

// Expedition tracks the total value available, including the surviving rover.
// Editing a surviving rover returns its parts to the budget at full value.
type Expedition struct {
	Index        int
	Credits      int
	Score        int
	Done, Failed bool
	Lost         int
	Salvaged     Kind
	ParReward    int
	Name         string
	Rover        Design
}

func NewExpedition() *Expedition { return &Expedition{Credits: ExpeditionStartingCredits} }

const expeditionKey = "expedition-v1"

// SaveExpedition stores the last completed-attempt checkpoint on this device.
func SaveExpedition(h engine.Host, e *Expedition) {
	if e == nil || e.Over() {
		h.Save(expeditionKey, nil)
		return
	}
	b, err := json.Marshal(e)
	if err == nil {
		h.Save(expeditionKey, b)
	}
}

func LoadExpedition(h engine.Host) *Expedition {
	var e Expedition
	if json.Unmarshal(h.Load(expeditionKey), &e) != nil || e.Over() || e.Index < 0 || e.Index >= len(ExpeditionLevels) || e.Credits < MinRover() {
		return nil
	}
	if len(e.Rover.Cells) > 0 && len(e.Rover.Validate(e.Credits)) != 0 {
		return nil
	}
	return &e
}

// MinRover is the minimum valid connected design: one motor wheel and one mast.
func MinRover() int {
	motor, _ := PartOf(KMotor)
	mast, _ := PartOf(KMast)
	return motor.Cost + mast.Cost
}

func (e *Expedition) Level() *Level { return GetLevel(ExpeditionLevels[e.Index]) }

// Finish applies a single attempt. A winning rover can be rebuilt at no extra
// cost; a crash consumes its full value. Time and par are in seconds.
func (e *Expedition) Finish(won bool, d Design, score int, elapsed, par float64) {
	if e.Over() {
		return
	}
	e.Lost, e.Salvaged, e.ParReward = 0, "", 0
	if won {
		e.Score += score
		e.Rover = Design{Cells: append([]Cell(nil), d.Cells...)}
		if e.Index+1 >= len(ExpeditionLevels) {
			e.Done = true
			return
		}
		if elapsed > 0 && par > 0 && elapsed < par {
			e.ParReward = UnderParCredits
			e.Credits += e.ParReward
		}
		e.Index++
		return
	}
	e.Lost = min(d.Cost(), e.Credits)
	e.Credits -= e.Lost
	e.Rover = Design{}
	e.Failed = e.Credits < MinRover()
}

// Salvage converts the recovered part to credits for a later build.
func (e *Expedition) Salvage(d Design) Kind {
	k, ok := SalvagePart(d)
	if !ok || e.Done {
		return ""
	}
	p, _ := PartOf(k)
	e.Credits += p.Cost
	e.Salvaged = k
	return k
}

func (e *Expedition) Over() bool { return e.Done || e.Failed }

// partNoun is a part's short name, plural when n != 1.
func partNoun(k Kind, n int) string {
	w := map[Kind]string{KFrame: "frame", KWheel: "wheel", KMotor: "motor wheel", KGripMotor: "grip motor", KMast: "mast"}[k]
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
	s.Design = Design{Cells: append([]Cell(nil), e.Rover.Cells...)}
	return s
}

// KitOf counts parts for mission-specific lander rules.
func KitOf(d Design) map[Kind]int {
	counts := map[Kind]int{}
	for _, c := range d.Cells {
		counts[c.K]++
	}
	return counts
}
