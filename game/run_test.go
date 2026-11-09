package game

import (
	"encoding/binary"
	"hash/fnv"
	"math"
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

func drive(l *Level, d Design, maxT float64) *Run {
	r := NewRun(l, d)
	for r.Time < maxT && r.Outcome == Running {
		r.Step(engine.FixedDT, 1)
	}
	return r
}

func TestStarterFinishesLevel1(t *testing.T) {
	r := drive(GetLevel(1), StarterDesign(), 90)
	t.Logf("outcome=%v t=%.1f x=%.1f samples=%d", r.Outcome, r.Time, r.Rover.Center().X, r.SamplesGot())
	if r.Outcome != Won {
		t.Fatalf("starter should win level 1: %v", r.Outcome.Message())
	}
}

func TestStarterIsValid(t *testing.T) {
	d := StarterDesign()
	if e := d.Validate(GetLevel(1).Credits); len(e) != 0 {
		t.Fatal(e)
	}
}

func TestLevelsDeterministic(t *testing.T) {
	a, b := GetLevel(3), GetLevel(3)
	for i := range a.Terrain.H {
		if a.Terrain.H[i] != b.Terrain.H[i] {
			t.Fatal("level generation not deterministic")
		}
	}
}

// A sturdy all-wheel-drive rover that every level must be beatable with.
func referenceDesign() Design {
	return Design{Cells: []Cell{
		{0, 0, KMotor}, {1, 0, KFrame}, {2, 0, KFrame}, {3, 0, KMotor},
		{0, 1, KFrame}, {1, 1, KFrame}, {2, 1, KMast}, {3, 1, KFrame},
	}}
}

func TestEveryLevelBeatable(t *testing.T) {
	d := referenceDesign()
	for _, def := range LevelDefs {
		l := BuildLevel(def)
		if d.Cost() > l.Credits+8 { // base budget plus a typical labelling bonus
			t.Fatalf("level %d reference design unaffordable", def.ID)
		}
		r := drive(l, d, 120)
		if r.Outcome != Won {
			t.Errorf("level %d (%s): %s x=%.1f", def.ID, def.Name, r.Outcome.Message(), r.Rover.Center().X)
		}
	}
}

func TestStarterCannotCoastThroughHardLevels(t *testing.T) {
	if r := drive(GetLevel(4), StarterDesign(), 120); r.Outcome == Won {
		t.Fatal("level 4 should require a better rover than the starter")
	}
}

func TestDesignRules(t *testing.T) {
	d := Design{}
	if len(d.Validate(99)) == 0 {
		t.Fatal("empty design must be invalid")
	}
	d = StarterDesign()
	d.Set(5, 5, KFrame) // floating, disconnected
	if len(d.Validate(99)) == 0 {
		t.Fatal("disconnected part must be invalid")
	}
	d = StarterDesign()
	d.Set(0, 1, KMast) // moves the mast, doesn't add a second
	n := 0
	for _, c := range d.Cells {
		if c.K == KMast {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("expected one mast, got %d", n)
	}
	if len(d.Validate(3)) == 0 {
		t.Fatal("over-budget must be invalid")
	}
}

// Levels must be bit-identical on every platform, or a run recorded on one
// device could not be checked on another. Hash values were taken on amd64 and
// must match under GOOS=js GOARCH=wasm too (see docs/PHYSICS.md).
func TestLevelTerrainGolden(t *testing.T) {
	for _, def := range LevelDefs {
		l := BuildLevel(def)
		h := fnv.New64a()
		var b [8]byte
		for _, v := range l.Terrain.H {
			binary.LittleEndian.PutUint64(b[:], math.Float64bits(v))
			h.Write(b[:])
		}
		t.Logf("level %d terrain %016x", def.ID, h.Sum64())
		if want, ok := goldenTerrain[def.ID]; ok && want != h.Sum64() {
			t.Errorf("level %d terrain changed: %016x, want %016x", def.ID, h.Sum64(), want)
		}
	}
}

var goldenTerrain = map[int]uint64{
	1: 0xb67c4e8e94c2a439, 2: 0x7c9b5d5a920caeac, 3: 0xa15b7fe52f477289,
	4: 0x8d6d0ebe36f97387, 5: 0x5e365f19704d38a6,
}

// A rover whose motor wheel never touches the ground goes nowhere: after a few
// seconds of trying the Restart offer appears, and scrapping it fails the run
// and costs an expedition the parts.
func TestStuckRoverOffersRestartAndScrapCostsParts(t *testing.T) {
	d := Design{Cells: []Cell{{0, 0, KFrame}, {1, 0, KFrame}, {2, 0, KFrame}, {1, 1, KMotor}, {2, 1, KMast}}}
	if errs := d.Validate(99); len(errs) > 0 {
		t.Fatal(errs)
	}
	r := NewRun(GetLevel(1), d)
	for i := 0; i < 60*3 && !r.IsStuck(); i++ {
		r.Step(1.0/60, 1)
	}
	if r.IsStuck() {
		t.Fatal("stuck offered too early")
	}
	for i := 0; i < 60*3 && !r.IsStuck(); i++ {
		r.Step(1.0/60, 1)
	}
	if !r.IsStuck() || r.Outcome != Running {
		t.Fatalf("want the Restart offer with the run still going, stuck=%v outcome=%v", r.IsStuck(), r.Outcome)
	}
	c0 := r.Rover.Center()
	r.Scrap()
	for i := 0; i < 30; i++ {
		r.Debris(1.0 / 60)
	}
	if r.Outcome != Scrapped || len(r.World.Cons) != 0 {
		t.Fatalf("scrap: outcome %v, %d joints left", r.Outcome, len(r.World.Cons))
	}
	spread := 0.0
	for _, b := range r.Rover.Bodies {
		spread = max(spread, b.Pos.Sub(c0).Len())
	}
	if spread < 2 {
		t.Fatalf("parts did not fly apart: max %.2f m", spread)
	}
	e := NewExpedition()
	before := e.Inv.Total()
	e.Finish(false, d, 0)
	if e.Inv.Total() != before-len(d.Cells) {
		t.Fatalf("expedition kept the scrapped parts: %d -> %d", before, e.Inv.Total())
	}
}
