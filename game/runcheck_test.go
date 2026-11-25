package game

import (
	"encoding/json"
	"testing"
)

func claimFor(t *testing.T, won bool, samples int, tm float64) RunClaim {
	t.Helper()
	l := GetLevel(LevelDefs[0].ID)
	d := StarterDesign()
	var cells []map[string]any
	for _, c := range d.Cells {
		cells = append(cells, map[string]any{"x": c.X, "y": c.Y, "k": string(c.K)})
	}
	b, _ := json.Marshal(cells)
	c := RunClaim{Level: l.ID, Won: won, Samples: samples, Time: tm, Cost: d.Cost(), Design: b}
	if won {
		c.Score = ScoreFor(l, true, samples, tm).Total()
		c.Stars = 1
	}
	return c
}

func TestCheckRunAcceptsHonestRuns(t *testing.T) {
	for _, c := range []RunClaim{claimFor(t, true, 2, 31.4), claimFor(t, true, 0, 200), claimFor(t, false, 0, 12)} {
		if why := CheckRun(c); why != "" {
			t.Errorf("honest run rejected: %s", why)
		}
	}
}

func TestCheckRunRejectsForgeries(t *testing.T) {
	cases := map[string]func(c *RunClaim){
		"unknown level":    func(c *RunClaim) { c.Level = 99999 },
		"inflated score":   func(c *RunClaim) { c.Score += 1000 },
		"too many samples": func(c *RunClaim) { c.Samples = 50 },
		"zero time win":    func(c *RunClaim) { c.Time = 0 },
		"cost mismatch":    func(c *RunClaim) { c.Cost-- },
		"bad stars":        func(c *RunClaim) { c.Stars = 9 },
		"junk design":      func(c *RunClaim) { c.Design = json.RawMessage(`"x"`) },
		"no mast":          func(c *RunClaim) { c.Design = json.RawMessage(`[{"x":0,"y":0,"k":"wheel"}]`) },
	}
	for name, f := range cases {
		c := claimFor(t, true, 2, 31.4)
		f(&c)
		if CheckRun(c) == "" {
			t.Errorf("%s was accepted", name)
		}
	}
	lost := claimFor(t, false, 0, 10)
	lost.Score = 300
	if CheckRun(lost) == "" {
		t.Error("a lost run with a score was accepted")
	}
}
