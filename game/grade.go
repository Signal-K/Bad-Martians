package game

import "github.com/signal-k/haphazard-martians/engine"

// Grade is the debrief's one-letter verdict on an attempt.
type Grade struct {
	Letter string
}

// GradeFor grades an attempt by its score against the best possible score:
// S needs every star, F is a failed mission.
func GradeFor(l *Level, ra *ResultArg) Grade {
	if ra.Outcome != Won {
		return Grade{"F"}
	}
	best := ScoreFor(l, true, len(l.Samples), 0).Total()
	f := float64(ra.Score.Total()) / float64(best)
	switch {
	case ra.Stars >= 3 && f >= 0.9:
		return Grade{"S"}
	case f >= 0.75:
		return Grade{"A"}
	case f >= 0.55:
		return Grade{"B"}
	case f >= 0.4:
		return Grade{"C"}
	}
	return Grade{"D"}
}

func (g Grade) Color() engine.Color {
	switch g.Letter {
	case "S", "A":
		return colNeon
	case "B", "C":
		return colChipLt
	}
	return colWarn
}
