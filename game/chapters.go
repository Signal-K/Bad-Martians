package game

import "github.com/signal-k/haphazard-martians/engine"

// A Chapter is one place on the campaign's journey: a run of missions on the
// same ground. Missions keep their numbers and unlock in order, so a chapter is
// a contiguous range of mission IDs.
//
// The Mars map (scene_map.go) shows the chapters as places on the planet, and
// each chapter opens its own route of stops (scene_select.go).
type Chapter struct {
	Key, Name, Blurb string
	First, Last      int // mission IDs, inclusive
	// Body is the world the chapter is on, "" for Mars's surface. Airless
	// bodies get a moon's sky.
	Body string
	// Pin places the chapter on the Mars map, in units of the disc's radius
	// from its centre. Moons sit outside the disc, on their orbits.
	Pin engine.Vec2
	// Climb makes the route rise towards its last stop.
	Climb bool
}

// Chapters lists the journey in order. Their ranges must tile LevelDefs.
var Chapters = []Chapter{
	{Key: "gale", Name: "Gale Crater", First: 1, Last: 10, Pin: engine.V(0.05, 0.02),
		Blurb: "Rust plains, boulder fields and the first crevasses. Learn what makes a rover drive."},
	{Key: "polar", Name: "Polar Ice Caps", First: 11, Last: 15, Pin: engine.V(-0.05, -0.72),
		Blurb: "Frosted dunes and frozen hills under a low sun. Short, heavy rovers on weak batteries."},
	{Key: "sharp", Name: "Mount Sharp", First: 16, Last: 20, Pin: engine.V(0.75, 0.42), Climb: true,
		Blurb: "Knife ridges, thin air and a last crevasse. The climb ends at the summit."},
	{Key: "phobos", Name: "Phobos", First: 21, Last: 21, Body: "phobos", Pin: engine.V(1.7, -0.4),
		Blurb: "The nearer moon: dark, grooved ground at a third of Mars's pull. Every bump is a long hop."},
	{Key: "deimos", Name: "Deimos", First: 22, Last: 22, Body: "deimos", Pin: engine.V(2.7, 0.1),
		Blurb: "The far moon: smooth pale dust and almost no grip. Keep the wheels on the ground."},
	{Key: "moon", Name: "The Moon", First: 23, Last: 23, Body: "moon", Pin: engine.V(3.55, -0.3),
		Blurb: "Earth's Moon: grey regolith and rilles under a blue Earth. The last stop of the expedition."},
}

// IsMoon reports whether the chapter is off Mars.
func (c *Chapter) IsMoon() bool { return c.Body != "" }

// Levels are the chapter's missions.
func (c *Chapter) Levels() []LevelDef { return LevelDefs[c.First-1 : c.Last] }

// Sky is the body to draw the chapter under: nil for Mars's own sky.
func (c *Chapter) Sky() *Body {
	if b := Bodies[c.Body]; b != nil && b.Airless {
		return b
	}
	return nil
}

// Unlocked reports whether the player has reached the chapter.
func (c *Chapter) Unlocked(p *Progress) bool { return p.Unlocked(c.First) }

// Done counts the chapter's missions that have been won.
func (c *Chapter) Done(p *Progress) int {
	n := 0
	for _, d := range c.Levels() {
		if p.Levels[d.ID].Stars > 0 {
			n++
		}
	}
	return n
}

// Stars is the stars earned in the chapter and the most it holds.
func (c *Chapter) Stars(p *Progress) (got, max int) {
	for _, d := range c.Levels() {
		got += p.Levels[d.ID].Stars
	}
	return got, 3 * len(c.Levels())
}

// Complete reports whether every mission in the chapter has been won.
func (c *Chapter) Complete(p *Progress) bool { return c.Done(p) == len(c.Levels()) }

// Metres is the distance of the chapter's missions that have been won.
func (c *Chapter) Metres(p *Progress) float64 {
	m := 0.0
	for _, d := range c.Levels() {
		if p.Levels[d.ID].Stars > 0 {
			m += d.Length
		}
	}
	return m
}

// Furthest is the index within the chapter of its furthest unlocked mission,
// or -1 when the chapter is still locked.
func (c *Chapter) Furthest(p *Progress) int {
	at := -1
	for i, d := range c.Levels() {
		if p.Unlocked(d.ID) {
			at = i
		}
	}
	return at
}

// ChapterIndex is the chapter a mission belongs to.
func ChapterIndex(missionID int) int {
	for i := range Chapters {
		if missionID >= Chapters[i].First && missionID <= Chapters[i].Last {
			return i
		}
	}
	return 0
}

// CurrentChapter is the furthest chapter the player has reached: where the
// rover is parked on the Mars map.
func CurrentChapter(p *Progress) int {
	at := 0
	for i := range Chapters {
		if Chapters[i].Unlocked(p) {
			at = i
		}
	}
	return at
}

// DistanceDriven is the metres of every mission won so far.
func DistanceDriven(p *Progress) float64 {
	m := 0.0
	for i := range Chapters {
		m += Chapters[i].Metres(p)
	}
	return m
}
