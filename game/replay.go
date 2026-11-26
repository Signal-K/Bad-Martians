package game

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math"

	"github.com/signal-k/haphazard-martians/engine"
)

// A drive is a pure function of the level, the rover design and the stick
// inputs fed to each fixed step, and the sim is bit-identical on every
// platform (TestLevelTerrainGolden, docs/PHYSICS.md). So a client that records
// its inputs lets the server drive the same run again and see whether it won.
//
// Inputs are quantised to 1/100 before the sim sees them, live and in replay,
// so both read exactly the same numbers. The log is run-length coded as
// (count, throttle, lean) bytes and sent as base64.

const (
	inputScale = 100
	// MaxInputsLen bounds the encoded log: a 120 s drive at DriveSpeed is about
	// 3200 steps, so even a log that changes every step fits well inside this.
	MaxInputsLen = 40000
)

func quantise(v float64) int8 {
	if v != v { // NaN
		return 0
	}
	return int8(math.Round(math.Max(-1, math.Min(1, v)) * inputScale))
}

// InputRecorder turns each step's stick values into the run-length log.
type InputRecorder struct {
	buf      []byte
	th, lean int8
	n        int
}

// Next records one step and returns the quantised throttle and lean the sim
// must use for it.
func (r *InputRecorder) Next(th, lean float64) (float64, float64) {
	t, l := quantise(th), quantise(lean)
	if r.n > 0 && t == r.th && l == r.lean && r.n < 255 {
		r.n++
	} else {
		r.flush()
		r.th, r.lean, r.n = t, l, 1
	}
	return float64(t) / inputScale, float64(l) / inputScale
}

func (r *InputRecorder) flush() {
	if r.n > 0 {
		r.buf = append(r.buf, byte(r.n), byte(r.th), byte(r.lean))
	}
}

// Encode returns the log so far. It can be called repeatedly.
func (r *InputRecorder) Encode() string {
	out := append([]byte(nil), r.buf...)
	if r.n > 0 {
		out = append(out, byte(r.n), byte(r.th), byte(r.lean))
	}
	return base64.StdEncoding.EncodeToString(out)
}

// Replay drives design on l with the recorded inputs, exactly as the drive
// scene does, and returns the finished run. Inputs left over after the run
// ends are ignored: the scene keeps recording through the end-of-run pause.
func Replay(l *Level, d Design, log string) (*Run, error) {
	if log == "" {
		return nil, errors.New("no recorded inputs")
	}
	if len(log) > MaxInputsLen {
		return nil, errors.New("recorded inputs too long")
	}
	raw, err := base64.StdEncoding.DecodeString(log)
	if err != nil || len(raw) == 0 || len(raw)%3 != 0 {
		return nil, errors.New("recorded inputs are not a valid log")
	}
	run := NewRun(l, d)
	run.Speed = DriveSpeed
	for i := 0; i < len(raw) && run.Outcome == Running; i += 3 {
		n, th, lean := int(raw[i]), int8(raw[i+1]), int8(raw[i+2])
		if n == 0 || th < -inputScale || th > inputScale || lean < -inputScale || lean > inputScale {
			return nil, fmt.Errorf("recorded inputs out of range at byte %d", i)
		}
		for j := 0; j < n && run.Outcome == Running; j++ {
			run.Lean = float64(lean) / inputScale
			run.Step(engine.FixedDT, float64(th)/inputScale)
		}
	}
	return run, nil
}
