package game

import (
	"encoding/json"
	"fmt"
	"math"
)

// RunClaim is what a client says about one finished attempt (the hm_runs record).
type RunClaim struct {
	Level   int
	Won     bool
	Stars   int
	Samples int
	Score   int
	Cost    int
	Time    float64
	Design  json.RawMessage // [{x, y, k}, ...]
	Inputs  string          // recorded stick inputs (InputRecorder.Encode); required to claim a win
}

// CheckRun is the server-side sanity check for a submitted run (MAR-62). It
// returns "" when the claim is plausible, or the first reason it is not.
//
// A claimed win must carry the recorded stick inputs, and the server drives the
// same design on the same level with them (Replay). The replay has to win, with
// the same samples, stars and time. Beyond that it catches a record that cannot
// have come from the real game: unknown level, a score the scoring rules cannot
// produce for those samples and that time, more samples than the level holds, or
// a rover that is illegal or does not cost what it says.
func CheckRun(c RunClaim) string {
	l := GetLevel(c.Level)
	if l == nil {
		return fmt.Sprintf("unknown level %d", c.Level)
	}
	if c.Stars < 0 || c.Stars > 3 {
		return "stars out of range"
	}
	if c.Samples < 0 || c.Samples > len(l.Samples) {
		return fmt.Sprintf("%d samples, level has %d", c.Samples, len(l.Samples))
	}
	if !c.Won {
		if c.Score != 0 || c.Stars != 0 {
			return "a lost run scores nothing"
		}
	} else {
		if c.Time <= 0 {
			return "won in no time"
		}
		// Time is sent rounded to 0.1 s, so allow the score either side of that.
		lo := ScoreFor(l, true, c.Samples, c.Time+0.05).Total()
		hi := ScoreFor(l, true, c.Samples, math.Max(0, c.Time-0.05)).Total()
		if c.Score < lo-1 || c.Score > hi+1 {
			return fmt.Sprintf("score %d not possible for %d samples in %.1fs (expected %d to %d)", c.Score, c.Samples, c.Time, lo, hi)
		}
	}
	var raw []struct {
		X int  `json:"x"`
		Y int  `json:"y"`
		K Kind `json:"k"`
	}
	if err := json.Unmarshal(c.Design, &raw); err != nil {
		return "design is not a list of parts"
	}
	d := Design{}
	for _, p := range raw {
		d.Cells = append(d.Cells, Cell{p.X, p.Y, p.K})
	}
	if errs := d.Validate(math.MaxInt32); len(errs) > 0 {
		return "illegal design: " + errs[0]
	}
	if d.Cost() != c.Cost {
		return fmt.Sprintf("design costs %d, claimed %d", d.Cost(), c.Cost)
	}
	if l.MaxParts > 0 && len(d.Cells) > l.MaxParts {
		return fmt.Sprintf("%d parts, lander carries %d", len(d.Cells), l.MaxParts)
	}
	if c.Won {
		run, err := Replay(l, d, c.Inputs)
		if err != nil {
			return err.Error()
		}
		switch {
		case run.Outcome != Won:
			return "the recorded drive does not reach the finish"
		case run.SamplesGot() != c.Samples:
			return fmt.Sprintf("the recorded drive collects %d samples, claimed %d", run.SamplesGot(), c.Samples)
		case run.Stars() != c.Stars:
			return fmt.Sprintf("the recorded drive earns %d stars, claimed %d", run.Stars(), c.Stars)
		case math.Abs(run.Time-c.Time) > 0.051: // sent rounded to 0.1 s
			return fmt.Sprintf("the recorded drive takes %.1fs, claimed %.1fs", run.Time, c.Time)
		}
	}
	return ""
}
