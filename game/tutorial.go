package game

import (
	"math"
	"sort"

	"github.com/signal-k/haphazard-martians/engine"
)

// Tutorial guides a first-time player through building and driving on the
// first mission. It rides on the Session; nil means no tutorial.
type Tutorial struct {
	Tested bool    // the player has tried a test drop
	Gas    float64 // seconds of gas held on the drive
	Done   bool
}

// Build steps, in order. The current step is the first one not yet done, so a
// player who drops the Starter rover skips straight past the parts steps.
const (
	tutBeam = iota
	tutWheel
	tutMotor
	tutMast
	tutTest
	tutGo
)

// tutGasSeconds is how long the player holds gas before the drive tip goes.
const tutGasSeconds = 1.5

// BuildStep is the build step the player is on.
func (t *Tutorial) BuildStep(d Design) int {
	n := map[Kind]int{}
	for _, c := range d.Cells {
		n[c.K]++
	}
	switch {
	case n[KFrame] < 2:
		return tutBeam
	case n[KWheel] < 1:
		return tutWheel
	case n[KMotor] < 1:
		return tutMotor
	case n[KMast] < 1:
		return tutMast
	case !t.Tested:
		return tutTest
	}
	return tutGo
}

var tutBuildText = [...]string{
	tutBeam:  "Start with a frame. Drag the Beam from Shapes onto the glowing cells.",
	tutWheel: "Wheels lift the frame off the rocks. Drag a Wheel under the front end.",
	tutMotor: "A Motor wheel drives the rover. Put one under the back end.",
	tutMast:  "Every rover carries one Navcam mast. Stand it on top of the frame.",
	tutTest:  "Looks like a rover! Tap Test drop to see how it sits.",
	tutGo:    "Steady on its wheels? Press GO to drive it.",
}

// hintCells are the empty cells where the next part of the tutorial should go.
func (t *Tutorial) hintCells(d Design, step int) [][2]int {
	var out [][2]int
	add := func(x, y int) {
		if x < 0 || x >= GridW || y < 0 || y >= GridH {
			return
		}
		if _, ok := d.At(x, y); ok {
			return
		}
		for _, o := range out {
			if o == [2]int{x, y} {
				return
			}
		}
		out = append(out, [2]int{x, y})
	}
	switch step {
	case tutBeam: // one row up, so wheels fit underneath
		for x := 3; x <= 5; x++ {
			add(x, 1)
		}
	case tutWheel, tutMotor: // under the front (right) or back (left) end of the frame
		y, lo, hi := GridH, GridW, -1
		for _, c := range d.Cells {
			if c.K == KFrame && c.Y < y {
				y = c.Y
			}
		}
		for _, c := range d.Cells {
			if c.K == KFrame && c.Y == y {
				lo, hi = min(lo, c.X), max(hi, c.X)
			}
		}
		if hi < 0 {
			break
		}
		ends := [2]int{hi, lo}
		if step == tutMotor {
			ends = [2]int{lo, hi}
		}
		for _, x := range ends {
			if y > 0 {
				add(x, y-1)
			} else { // frame on the floor: beside it is all that is left
				add(x+map[bool]int{true: 1, false: -1}[x == hi], y)
			}
			if len(out) > 0 {
				break
			}
		}
	case tutMast: // on top of the frame, middle first
		var tops [][2]int
		for _, c := range d.Cells {
			if c.K == KFrame {
				if _, ok := d.At(c.X, c.Y+1); !ok {
					tops = append(tops, [2]int{c.X, c.Y + 1})
				}
			}
		}
		sort.Slice(tops, func(i, j int) bool { return tops[i][0] < tops[j][0] })
		if n := len(tops); n > 0 {
			add(tops[n/2][0], tops[n/2][1])
		}
	}
	return out
}

// drawCoach draws the tutorial bubble with its text, and a pulsing frame and
// arrow on the thing to touch next (when target.W > 0).
func drawCoach(r engine.Renderer, clock float64, box engine.Rect, msg string, target engine.Rect) {
	pulse := 0.5 + 0.5*math.Sin(clock*5)
	if target.W > 0 {
		g := 3 + 3*pulse
		outline(r, target.X-g, target.Y-g, target.W+2*g, target.H+2*g, colNeon.Alpha(uint8(150+100*pulse)))
		bob := 6 * pulse
		if target.X+target.W < 160 { // left edge: arrow on the right, pointing left
			ax, ay := target.X+target.W+14+bob, target.Y+target.H/2
			r.Poly([]engine.Vec2{engine.V(ax, ay), engine.V(ax+18, ay-12), engine.V(ax+18, ay+12)}, colNeon)
		} else { // arrow above, pointing down
			ax, ay := target.X+target.W/2, target.Y-12-bob
			r.Poly([]engine.Vec2{engine.V(ax, ay), engine.V(ax-12, ay-18), engine.V(ax+12, ay-18)}, colNeon)
		}
	}
	drawPanel(r, box.X, box.Y, box.W, box.H, "Tutorial")
	if msg == "" {
		return
	}
	lines := wrap(msg, sizeBody, box.W-36)
	y := box.Y + box.H/2 - float64(len(lines))*11 - 2
	for _, l := range lines {
		r.Text(l, box.X+18, y, sizeBody, colInk, engine.AlignLeft)
		y += 22
	}
}

// wrap breaks s into lines no wider than w at the given text size.
func wrap(s string, size, w float64) []string {
	var lines []string
	line := ""
	word := ""
	flush := func() {
		if word == "" {
			return
		}
		switch {
		case line == "":
			line = word
		case textW(line+" "+word, size) <= w:
			line += " " + word
		default:
			lines = append(lines, line)
			line = word
		}
		word = ""
	}
	for _, ch := range s {
		if ch == ' ' {
			flush()
			continue
		}
		word += string(ch)
	}
	flush()
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

const tutorialDoneKey = "hm.tutorial"

// TutorialDone reports whether the player has finished or skipped the tutorial.
func TutorialDone(h engine.Host) bool { return string(h.Load(tutorialDoneKey)) == "done" }

// FinishTutorial remembers the tutorial is over so it never shows again.
func FinishTutorial(h engine.Host, t *Tutorial) {
	if t != nil {
		t.Done = true
	}
	h.Save(tutorialDoneKey, []byte("done"))
}
