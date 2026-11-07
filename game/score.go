package game

import "math"

// Score is how well an attempt went. Finishing is the base; samples and speed
// add to it, so a slow, careful finish and a fast, greedy one both score, but
// the best runs do everything.
type Score struct {
	Finish  int
	Samples int
	Time    int
}

const (
	scoreFinish    = 500
	scorePerSample = 150
	scoreTimeMax   = 500
	timeFullAtPar  = 0.6 // at or under this fraction of par: full time bonus
	timeZeroAtPar  = 2.0 // at or over this multiple of par: no time bonus
)

func (s Score) Total() int { return s.Finish + s.Samples + s.Time }

// ScoreFor scores an attempt: won, samples collected, and the time taken.
func ScoreFor(l *Level, won bool, samples int, t float64) Score {
	if !won {
		return Score{}
	}
	f := (timeZeroAtPar*l.Par - t) / ((timeZeroAtPar - timeFullAtPar) * l.Par)
	f = math.Max(0, math.Min(1, f))
	return Score{
		Finish:  scoreFinish,
		Samples: scorePerSample * samples,
		Time:    int(math.Round(f * scoreTimeMax)),
	}
}

// Score is the live score of this run.
func (r *Run) Score() Score {
	return ScoreFor(r.Level, r.Outcome == Won, r.SamplesGot(), r.Time)
}
