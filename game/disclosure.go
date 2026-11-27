package game

import (
	"strings"

	"github.com/signal-k/haphazard-martians/engine"
)

// Progressive disclosure. Playtest feedback (docs/playtest/feedback-2026-11-26.md)
// said the game opens with too much text and too many systems before the
// player reaches the mechanics. Screens therefore show only what the player
// has earned the right to need: the tier comes from missions won.
const (
	TierFirst    = iota // no mission won: build, drive, nothing else
	TierLearning        // first win: stars, daily mission, account
	TierFull            // three wins: the expedition and its economy
)

// tierLearningWins and tierFullWins are the missions won that open each tier.
const (
	tierLearningWins = 1
	tierFullWins     = 3
)

// WonMissions counts missions finished with at least one star.
func WonMissions(p *Progress) int {
	n := 0
	for _, lp := range p.Levels {
		if lp.Stars > 0 {
			n++
		}
	}
	return n
}

// DisclosureTier is how much of the game the player has been shown.
func DisclosureTier(p *Progress) int {
	switch won := WonMissions(p); {
	case won >= tierFullWins:
		return TierFull
	case won >= tierLearningWins:
		return TierLearning
	}
	return TierFirst
}

// FirstOpen reports a brand-new player: the game skips the map and opens in
// the first mission's rover bay, where the one-line tutorial takes over.
func FirstOpen(h engine.Host) bool {
	return !TutorialDone(h) && WonMissions(LoadProgress(h)) == 0
}

// FirstMissionSession is the session a first-time player starts in.
func FirstMissionSession(h engine.Host) *Session {
	return NewSessionFor(h, GetLevel(LevelDefs[0].ID))
}

// firstSentence is s up to and including its first full stop.
func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i >= 0 {
		return s[:i+1]
	}
	return s
}
