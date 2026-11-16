package game

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/signal-k/haphazard-martians/engine"
)

// Playtest is a headless balance harness (KES-517, KES-518). It drives the real
// Run physics with simple pilots over a pool of generated rovers, then plays
// whole expeditions with a few player types through the real Expedition rules
// (Finish, Salvage), so economy numbers can be compared without a person at
// the controls. Nothing in the game uses it; cmd/playtest prints the report.

// Pilot is a driving policy: the throttle and lean for the next step.
type Pilot struct {
	Name  string
	Drive func(r *Run) (throttle, lean float64)
}

// Pilots are deliberately plain: a player holding the stick forward, and one
// who drives at 70% and leans against a tipping rover.
var Pilots = []Pilot{
	{"full", func(*Run) (float64, float64) { return 1, 0 }},
	{"steady", func(r *Run) (float64, float64) {
		switch t := r.tilt(); {
		case t > 20:
			return 0.7, -1
		case t < -20:
			return 0.7, 1
		}
		return 0.7, 0
	}},
}

// Attempt is one drive of one design on one level.
type Attempt struct {
	Outcome Outcome
	Time    float64
	Samples int
	Damaged bool
	Score   int
}

func (a Attempt) Won() bool { return a.Outcome == Won }

// Drive plays one attempt to the end at the drive scene's speed. A stuck rover
// is scrapped, as a player pressing Restart would, so it costs its parts.
func Drive(l *Level, d Design, p Pilot) Attempt {
	r := NewRun(l, d)
	r.Speed = DriveSpeed
	for r.Outcome == Running {
		t, lean := p.Drive(r)
		r.Lean = lean
		r.Step(engine.FixedDT, t)
		if r.IsStuck() {
			r.Scrap()
		}
	}
	return Attempt{Outcome: r.Outcome, Time: r.Time, Samples: r.SamplesGot(), Damaged: r.Damaged(), Score: r.Score().Total()}
}

// Key is a design's stable identity for caching and de-duplication.
func (d Design) Key() string {
	cs := append([]Cell(nil), d.Cells...)
	sort.Slice(cs, func(i, j int) bool {
		if cs[i].Y != cs[j].Y {
			return cs[i].Y < cs[j].Y
		}
		return cs[i].X < cs[j].X
	})
	var b strings.Builder
	for _, c := range cs {
		fmt.Fprintf(&b, "%d,%d,%s;", c.X, c.Y, c.K)
	}
	return b.String()
}

// GenerateDesigns returns n distinct launchable designs (ignoring budget and
// lander rules, which each level checks), deterministic for a seed. They range
// from four-part landers to long two-row rovers, the way players build.
func GenerateDesigns(n int, seed uint64) []Design {
	rg := newRng(seed)
	seen := map[string]bool{StarterDesign().Key(): true}
	out := []Design{StarterDesign()}
	for tries := 0; len(out) < n && tries < n*200; tries++ {
		d := randomDesign(rg)
		if len(d.Validate(math.MaxInt32)) != 0 || seen[d.Key()] {
			continue
		}
		seen[d.Key()] = true
		out = append(out, d)
	}
	return out
}

func randomDesign(rg *rng) Design {
	wheel := func() Kind {
		switch v := rg.intn(10); {
		case v < 4:
			return KWheel
		case v < 8:
			return KMotor
		default:
			return KGripMotor
		}
	}
	if rg.intn(5) == 0 { // a four-part lander
		d := Design{Cells: []Cell{{0, 0, wheel()}, {1, 0, KFrame}, {2, 0, wheel()}, {1, 1, KMast}}}
		if motorCount(d) == 0 {
			d.Cells[2].K = KMotor
		}
		return d
	}
	var d Design
	cols := 2 + rg.intn(6) // 2..7 wide
	for x := 0; x < cols; x++ {
		k := KFrame
		if x == 0 || x == cols-1 || rg.intn(3) == 0 {
			k = wheel()
		}
		d.Set(x, 0, k)
	}
	if cols > 2 && rg.intn(3) > 0 { // frames over part of it, mast on top
		a := rg.intn(cols - 1)
		b := a + 1 + rg.intn(cols-a-1)
		for x := a; x <= b; x++ {
			d.Set(x, 1, KFrame)
		}
		d.Set(a+rg.intn(b-a+1), 2, KMast)
	} else {
		d.Set(rg.intn(cols), 1, KMast)
	}
	return d
}

// LevelReport is how the design pool fares on one level.
type LevelReport struct {
	Level                    *Level
	Legal, Winners, UnderPar int
	CheapestWin, CheapestPar int                           // credits; 0 = none
	FastestWin               float64                       // seconds; 0 = none
	Results                  map[string]map[string]Attempt // design key -> pilot -> attempt
}

// Lab holds every attempt of every legal design, so expeditions replay cheaply.
type Lab struct {
	Designs []Design
	Levels  []*Level
	Reports []*LevelReport
}

// NewLab drives every legal design with every pilot on each level.
func NewLab(levelIDs []int, designs []Design) *Lab {
	lab := &Lab{Designs: designs}
	for _, id := range levelIDs {
		l := GetLevel(id)
		rep := &LevelReport{Level: l, Results: map[string]map[string]Attempt{}}
		for _, d := range designs {
			if len(RuleProblems(l.LevelDef, d)) != 0 {
				continue
			}
			rep.Legal++
			res := map[string]Attempt{}
			won, under := false, false
			for _, p := range Pilots {
				a := Drive(l, d, p)
				res[p.Name] = a
				if a.Won() {
					won, under = true, under || a.Time < l.Par
					if rep.FastestWin == 0 || a.Time < rep.FastestWin {
						rep.FastestWin = a.Time
					}
				}
			}
			rep.Results[d.Key()] = res
			c := d.Cost()
			if won {
				rep.Winners++
				if rep.CheapestWin == 0 || c < rep.CheapestWin {
					rep.CheapestWin = c
				}
			}
			if under {
				rep.UnderPar++
				if rep.CheapestPar == 0 || c < rep.CheapestPar {
					rep.CheapestPar = c
				}
			}
		}
		lab.Levels = append(lab.Levels, l)
		lab.Reports = append(lab.Reports, rep)
	}
	return lab
}

// Economy is the set of numbers under test. ShippedEconomy is what the game uses.
type Economy struct {
	StartingCredits int
	UnderParCredits int
	Salvage         bool // an undamaged win salvages its dearest part's credits
}

func ShippedEconomy() Economy {
	return Economy{StartingCredits: ExpeditionStartingCredits, UnderParCredits: UnderParCredits, Salvage: ExpeditionSalvage}
}

func (ec Economy) String() string {
	return fmt.Sprintf("start %d, par +%d, salvage %v", ec.StartingCredits, ec.UnderParCredits, ec.Salvage)
}

// Player types: how a player picks a rover for the current stage.
const (
	// Planner knows which rovers work and builds the cheapest one that does.
	Planner = "planner"
	// Racer builds the fastest affordable winner, chasing par.
	Racer = "racer"
	// Tinkerer builds something affordable at random; after each crash there
	// is an even chance they copy a rover that works (learning from the debrief).
	Tinkerer = "tinkerer"
)

var PlayerTypes = []string{Planner, Racer, Tinkerer}

type choice struct {
	d  Design
	p  Pilot
	a  Attempt
	ok bool
}

func (lab *Lab) pick(kind string, stage, budget int, rg *rng, learned bool, keep Design) choice {
	rep, par := lab.Reports[stage], lab.Levels[stage].Par
	// A surviving rover that is legal here and works is kept (rebuilding is free,
	// so a sensible player only changes a rover that fails).
	if res, ok := rep.Results[keep.Key()]; ok && len(keep.Cells) > 0 && kind != Tinkerer && keep.Cost() <= budget {
		for _, p := range Pilots {
			if a := res[p.Name]; a.Won() && (kind != Racer || a.Time < par) {
				return choice{keep, p, a, true}
			}
		}
	}
	var all, winners []choice
	for _, d := range lab.Designs {
		res, ok := rep.Results[d.Key()]
		if !ok || d.Cost() > budget {
			continue
		}
		for _, p := range Pilots {
			c := choice{d, p, res[p.Name], true}
			all = append(all, c)
			if c.a.Won() {
				winners = append(winners, c)
			}
		}
	}
	if len(all) == 0 {
		return choice{}
	}
	if len(winners) == 0 || (kind == Tinkerer && !learned) {
		return all[rg.intn(len(all))]
	}
	if kind == Racer {
		sort.SliceStable(winners, func(i, j int) bool { return winners[i].a.Time < winners[j].a.Time })
	} else {
		sort.SliceStable(winners, func(i, j int) bool { return winners[i].d.Cost() < winners[j].d.Cost() })
	}
	return winners[0]
}

// Outcome of one played expedition.
type ExpeditionRun struct {
	Exp            *Expedition
	Crashes        int
	ParAwards      int
	SalvageCredits int
	Log            []string
}

// PlayExpedition plays one expedition through the real Expedition rules. An
// economy other than the shipped one is applied on top: the starting purse
// directly, and the par reward by correcting what Finish awarded.
func (lab *Lab) PlayExpedition(kind string, ec Economy, seed uint64) ExpeditionRun {
	rg := newRng(seed)
	out := ExpeditionRun{Exp: NewExpedition()}
	e := out.Exp
	e.Credits = ec.StartingCredits
	learned := false
	for attempts := 0; !e.Over() && attempts < 30; attempts++ {
		stage := e.Index
		l := lab.Levels[stage]
		c := lab.pick(kind, stage, e.Credits, rg, learned, e.Rover)
		if !c.ok {
			e.Failed = true
			out.Log = append(out.Log, fmt.Sprintf("stage %d: nothing affordable with %d credits", stage+1, e.Credits))
			break
		}
		e.Finish(c.a.Won(), c.d, c.a.Score, c.a.Time, l.Par)
		if e.ParReward > 0 {
			e.Credits += ec.UnderParCredits - e.ParReward
			e.ParReward = ec.UnderParCredits
			if e.ParReward > 0 {
				out.ParAwards++
			}
		}
		salvaged := Kind("")
		if c.a.Won() && !c.a.Damaged && ec.Salvage && !e.Done {
			if salvaged = e.Salvage(c.d); salvaged != "" {
				p, _ := PartOf(salvaged)
				out.SalvageCredits += p.Cost
			}
		}
		if !c.a.Won() {
			out.Crashes++
			learned = learned || (kind == Tinkerer && rg.intn(2) == 0)
		}
		out.Log = append(out.Log, fmt.Sprintf("stage %d %s: %d-credit rover, %s pilot, %s in %.1fs (par %.0f) -> %d credits%s%s",
			stage+1, l.Name, c.d.Cost(), c.p.Name, outcomeWord(c.a.Outcome), c.a.Time, l.Par, e.Credits,
			ifs(e.ParReward > 0, fmt.Sprintf(", +%d par", e.ParReward)), ifs(salvaged != "", ", salvaged "+string(salvaged))))
	}
	return out
}

func outcomeWord(o Outcome) string {
	return map[Outcome]string{Won: "won", Fell: "fell", Stuck: "stuck", TimedOut: "timed out", Scrapped: "stuck, scrapped"}[o]
}

func ifs(ok bool, s string) string {
	if ok {
		return s
	}
	return ""
}

// Tally sums n expeditions of one player type under one economy.
type Tally struct {
	Runs, Completed, Crashes, ParAwards, SalvageCredits, Score, CreditsLeft int
	FailedAt                                                                [3]int
}

func (t Tally) Rate() float64 { return pct(t.Completed, t.Runs) }

func pct(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return 100 * float64(a) / float64(b)
}

// Play runs n seeded expeditions per player type.
func (lab *Lab) Play(ec Economy, n int, seed uint64) map[string]Tally {
	out := map[string]Tally{}
	for ti, kind := range PlayerTypes {
		var t Tally
		for i := 0; i < n; i++ {
			r := lab.PlayExpedition(kind, ec, seed+uint64(ti*100003+i))
			t.Runs++
			t.Crashes += r.Crashes
			t.ParAwards += r.ParAwards
			t.SalvageCredits += r.SalvageCredits
			t.Score += r.Exp.Score
			if r.Exp.Done {
				t.Completed++
				t.CreditsLeft += r.Exp.Credits
			} else {
				t.FailedAt[min(r.Exp.Index, 2)]++
			}
		}
		out[kind] = t
	}
	return out
}
