package game

import (
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

func stickInRun(t *testing.T) (*harness, *Stick) {
	h := newHarness(t)
	ses := NewSessionFor(h.g.Host, GetLevel(1))
	ses.Design = StarterDesign()
	ses.Tut = nil
	h.g.Load(SceneRun, ses)
	h.frames(2, engine.Input{})
	st, ok := engine.Get[*Stick](h.g.Scene().Find("stick"))
	if !ok {
		t.Fatal("no stick in the drive scene")
	}
	return h, st
}

func touches(ts ...engine.Touch) engine.Input {
	in := engine.Input{Touches: ts, Down: len(ts) > 0}
	if len(ts) > 0 {
		in.Pointer = ts[0].Pos
	}
	return in
}

func TestStickSitsLeft(t *testing.T) {
	_, st := stickInRun(t)
	if st.Home.X > ScreenW/3 {
		t.Fatalf("stick should sit on the left, at x %.0f", st.Home.X)
	}
}

func TestStickFollowsItsOwnFinger(t *testing.T) {
	h, st := stickInRun(t)
	thumb := engine.Touch{ID: 7, Pos: st.Home.Add(engine.V(st.R, 0))}
	h.frames(2, touches(thumb))
	if !st.Held() || st.X < 0.9 {
		t.Fatalf("full right should drive: held %v x %.2f", st.Held(), st.X)
	}
	// a second finger lands elsewhere, listed first: the stick keeps the thumb
	other := engine.Touch{ID: 3, Pos: engine.V(ScreenW-80, 200)}
	h.frames(3, touches(other, thumb))
	if !st.Held() || st.X < 0.9 || st.Home.X > ScreenW/3 {
		t.Fatalf("second finger moved or dropped the stick: held %v x %.2f", st.Held(), st.X)
	}
	// the other finger lifts: still held
	h.frames(2, touches(thumb))
	if !st.Held() {
		t.Fatal("lifting another finger dropped the stick")
	}
	// dragging far past the rim does not move the base
	home := st.Home
	h.frames(2, touches(engine.Touch{ID: 7, Pos: home.Add(engine.V(400, 0))}))
	if st.Home != home || st.X != 1 {
		t.Fatalf("base drifted or knob unclamped: home %v x %.2f", st.Home, st.X)
	}
	// the thumb lifts: released, centred
	h.frames(2, touches(other))
	if st.Held() || st.X != 0 {
		t.Fatalf("lifting the thumb should release: held %v x %.2f", st.Held(), st.X)
	}
	// a finger already down that slides onto the stick does not grab it
	h.frames(2, touches(engine.Touch{ID: 3, Pos: home}))
	if st.Held() {
		t.Fatal("a sliding finger grabbed the stick")
	}
}
