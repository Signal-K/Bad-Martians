package game

import (
	"strings"
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

// The Photo button asks the client for a snapshot; the snapshot frame leaves
// the HUD out and carries the caption instead.
func TestPhotoSnapsTheWorldWithoutTheHUD(t *testing.T) {
	h := newHarness(t)
	h.g.Load(SceneRun, NewSessionFor(h.g.Host, GetLevel(1)))
	h.frames(2, engine.Input{})
	h.clickButton("Photo")
	name := h.g.TakePhotoRequest()
	if !strings.HasPrefix(name, "haphazard-martians-mission-1-") || !strings.HasSuffix(name, ".png") {
		t.Fatalf("photo request %q", name)
	}
	if h.g.TakePhotoRequest() != "" {
		t.Fatal("a request is taken once")
	}
	has := func(rec *engine.Recorder, s string) bool {
		for _, x := range rec.Strings {
			if strings.Contains(x, s) {
				return true
			}
		}
		return false
	}
	normal := engine.NewRecorder()
	h.g.Draw(normal)
	h.g.Snapping = true
	snap := engine.NewRecorder()
	h.g.Draw(snap)
	h.g.Snapping = false
	if !has(normal, "Par ") || has(snap, "Par ") {
		t.Fatal("the HUD must show normally and be left out of the photo")
	}
	if !has(snap, "HAPHAZARD MARTIANS") || !has(snap, "Gale Warm-up") || has(normal, "HAPHAZARD MARTIANS") {
		t.Fatal("the caption belongs on the photo only")
	}
}
