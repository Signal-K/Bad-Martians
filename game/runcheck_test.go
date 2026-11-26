package game

import (
	"encoding/base64"
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/signal-k/haphazard-martians/engine"
)

// drivenClaim drives the starter rover on level 1 the way the drive scene does
// (inputs recorded and quantised, then stepped at DriveSpeed, with the scene's
// end-of-run pause) and returns what the client would send for it.
func drivenClaim(t *testing.T) RunClaim {
	t.Helper()
	l := GetLevel(LevelDefs[0].ID)
	d := StarterDesign()
	run := NewRun(l, d)
	run.Speed = DriveSpeed
	rec := &InputRecorder{}
	for i := 0; i < 60*130; i++ {
		th, lean := 1.0, 0.4*math.Sin(float64(i)/45) // a wobbling lean makes the log non-trivial
		th, lean = rec.Next(th, lean)
		run.Lean = lean
		run.Step(engine.FixedDT, th)
		if run.Outcome != Running && i > 0 {
			for j := 0; j < 100; j++ { // the scene records through its end pause
				rec.Next(th, lean)
			}
			break
		}
	}
	if run.Outcome != Won {
		t.Fatalf("starter should win level 1 with this drive: %s", run.Outcome.Message())
	}
	var cells []map[string]any
	for _, c := range d.Cells {
		cells = append(cells, map[string]any{"x": c.X, "y": c.Y, "k": string(c.K)})
	}
	b, _ := json.Marshal(cells)
	return RunClaim{Level: l.ID, Won: true, Stars: run.Stars(), Samples: run.SamplesGot(),
		Score: run.Score().Total(), Cost: d.Cost(), Time: math.Round(run.Time*10) / 10,
		Design: b, Inputs: rec.Encode()}
}

func lostClaim(t *testing.T) RunClaim {
	c := drivenClaim(t)
	c.Won, c.Score, c.Stars, c.Inputs = false, 0, 0, ""
	return c
}

func TestCheckRunAcceptsHonestRuns(t *testing.T) {
	start := time.Now()
	if why := CheckRun(drivenClaim(t)); why != "" {
		t.Errorf("honest win rejected: %s", why)
	}
	t.Logf("drive, replay and checks took %v", time.Since(start))
	if why := CheckRun(lostClaim(t)); why != "" {
		t.Errorf("honest loss rejected: %s", why)
	}
}

// What the client records is what the server replays, bit for bit.
func TestReplayMatchesTheLiveDrive(t *testing.T) {
	c := drivenClaim(t)
	run, err := Replay(GetLevel(c.Level), StarterDesign(), c.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	if run.Outcome != Won || run.SamplesGot() != c.Samples || math.Abs(run.Time-c.Time) > 0.051 {
		t.Fatalf("replay: %v samples=%d time=%.2f; claim samples=%d time=%.1f", run.Outcome.Message(), run.SamplesGot(), run.Time, c.Samples, c.Time)
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

		// The score is one the rules allow, but the drive does not back it up.
		"win with no inputs":       func(c *RunClaim) { c.Inputs = "" },
		"win with junk inputs":     func(c *RunClaim) { c.Inputs = "not base64!" },
		"inputs of the wrong size": func(c *RunClaim) { c.Inputs = base64.StdEncoding.EncodeToString([]byte{1, 2}) },
		"inputs out of range":      func(c *RunClaim) { c.Inputs = base64.StdEncoding.EncodeToString([]byte{200, 127, 0}) },
		"zero-length input run":    func(c *RunClaim) { c.Inputs = base64.StdEncoding.EncodeToString([]byte{0, 100, 0}) },
		"inputs that never finish": func(c *RunClaim) { c.Inputs = base64.StdEncoding.EncodeToString([]byte{60, 100, 0}) }, // 1 s of throttle
		"inputs that coast":        func(c *RunClaim) { c.Inputs = base64.StdEncoding.EncodeToString([]byte{255, 0, 0, 255, 0, 0}) },
		"faster time than driven":  func(c *RunClaim) { c.Time = math.Max(1, c.Time-8) },
		"faked time and its score": func(c *RunClaim) { // consistent with the scoring rules, not with the drive
			c.Time = math.Max(1, c.Time-8)
			c.Score = ScoreFor(GetLevel(c.Level), true, c.Samples, c.Time).Total()
		},
		"slower time than driven":   func(c *RunClaim) { c.Time += 8 },
		"fewer samples than driven": func(c *RunClaim) { c.Samples-- },
		"stars not earned":          func(c *RunClaim) { c.Stars = 3 },
		"another level's drive":     func(c *RunClaim) { c.Level = LevelDefs[1].ID },
	}
	for name, f := range cases {
		c := drivenClaim(t)
		f(&c)
		if why := CheckRun(c); why == "" {
			t.Errorf("%s was accepted", name)
		}
	}
	lost := lostClaim(t)
	lost.Score = 300
	if CheckRun(lost) == "" {
		t.Error("a lost run with a score was accepted")
	}
}

func TestInputRecorderQuantisesAndCodes(t *testing.T) {
	r := &InputRecorder{}
	th, lean := r.Next(0.123456, -2) // lean clamps to -1
	if th != 0.12 || lean != -1 {
		t.Fatalf("quantised to %v, %v", th, lean)
	}
	if th, _ = r.Next(math.NaN(), 0); th != 0 {
		t.Fatal("NaN should read as no input")
	}
	for i := 0; i < 600; i++ {
		r.Next(1, 0)
	}
	raw, _ := base64.StdEncoding.DecodeString(r.Encode())
	if len(raw)%3 != 0 || len(raw) > 3*6 { // 600 equal steps are three or four runs, not 600
		t.Fatalf("log is %d bytes", len(raw))
	}
}
