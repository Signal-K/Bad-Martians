package game

import (
	"reflect"
	"testing"
)

// The playtest harness must stay deterministic and play by the real rules.
func TestPlaytestIsDeterministicAndFollowsTheRules(t *testing.T) {
	pool := GenerateDesigns(8, 3)
	if len(pool) != 8 {
		t.Fatalf("pool %d", len(pool))
	}
	for _, d := range pool {
		if len(d.Validate(1<<30)) != 0 {
			t.Fatalf("invalid design %s", d.Key())
		}
	}
	lab := NewLab(ExpeditionLevels, pool)
	a := lab.Play(ShippedEconomy(), 20, 7)
	b := NewLab(ExpeditionLevels, GenerateDesigns(8, 3)).Play(ShippedEconomy(), 20, 7)
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("not deterministic:\n%+v\n%+v", a, b)
	}
	for kind, tl := range a {
		failed := tl.FailedAt[0] + tl.FailedAt[1] + tl.FailedAt[2]
		if tl.Runs != 20 || tl.Completed+failed != tl.Runs {
			t.Fatalf("%s tally does not add up: %+v", kind, tl)
		}
	}
	// No par reward and no salvage: a winner can never end with more than it started.
	flat := Economy{StartingCredits: 20}
	for i := uint64(0); i < 10; i++ {
		r := lab.PlayExpedition(Tinkerer, flat, i)
		if r.ParAwards != 0 || r.SalvageCredits != 0 || r.Exp.Credits > 20 {
			t.Fatalf("flat economy leaked credits: %+v %v", r.Exp, r.Log)
		}
	}
}
