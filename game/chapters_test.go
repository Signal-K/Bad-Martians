package game

import (
	"testing"

	"github.com/signal-k/haphazard-martians/engine"
)

// Chapters tile the campaign: every mission belongs to exactly one, in order,
// and a chapter on a moon holds only missions set on that moon.
func TestChaptersTileTheCampaign(t *testing.T) {
	next := 1
	for _, c := range Chapters {
		if c.First != next || c.Last < c.First {
			t.Fatalf("chapter %s covers %d–%d, want it to start at %d", c.Key, c.First, c.Last, next)
		}
		for _, d := range c.Levels() {
			if d.Body != c.Body {
				t.Errorf("mission %d is on %q but chapter %s is on %q", d.ID, d.Body, c.Key, c.Body)
			}
			if ChapterIndex(d.ID) != chapterByKey(c.Key) {
				t.Errorf("mission %d maps to the wrong chapter", d.ID)
			}
		}
		next = c.Last + 1
	}
	if next != len(LevelDefs)+1 {
		t.Fatalf("chapters end at mission %d but there are %d missions", next-1, len(LevelDefs))
	}
}

func chapterByKey(key string) int {
	for i, c := range Chapters {
		if c.Key == key {
			return i
		}
	}
	return -1
}

// The ice caps and each moon are chapters of their own, apart from Gale.
func TestIceCapsAndMoonsAreSeparateChapters(t *testing.T) {
	if ChapterIndex(11) == ChapterIndex(10) || ChapterIndex(15) != ChapterIndex(11) {
		t.Fatal("the ice caps should be their own chapter, starting at mission 11")
	}
	if ChapterIndex(21) == ChapterIndex(22) || ChapterIndex(21) == ChapterIndex(20) {
		t.Fatal("Phobos and Deimos should each be a chapter of their own")
	}
}

// The map opens on the furthest chapter reached, shows later ones locked, and
// opens a chapter's own route.
func TestMarsMapOpensChapters(t *testing.T) {
	defer func() { selMission, selChapter = 0, -1 }()
	selMission, selChapter = 0, -1
	h := newHarness(t)
	h.g.Load(SceneMap, nil)
	h.frames(2, engine.Input{})
	if !h.button("Enter chapter").Primary || h.button("Enter chapter").Disabled {
		t.Fatal("a new player can enter the first chapter")
	}
	// a locked chapter can be read about but not entered
	h.clickButton("Polar Ice Caps")
	h.frames(2, engine.Input{})
	if !h.button("Enter chapter").Disabled {
		t.Fatal("the ice caps are locked until mission 10 is won")
	}
	h.clickButton("Enter chapter")
	if h.g.Scene().Name != SceneMap {
		t.Fatalf("a locked chapter must not open, got %s", h.g.Scene().Name)
	}
	// win the whole of Gale and the caps open
	p := LoadProgress(h.h)
	for id := 1; id <= 10; id++ {
		p.Record(id, 1, 30, 100)
	}
	p.Save(h.h)
	h.g.Load(SceneMap, nil)
	h.frames(2, engine.Input{})
	if CurrentChapter(LoadProgress(h.h)) != 1 {
		t.Fatal("the rover should be parked at the ice caps")
	}
	h.clickButton("Polar Ice Caps") // already picked: tapping a picked place enters it
	h.frames(2, engine.Input{})
	if h.g.Scene().Name != SceneMenu {
		h.clickButton("Enter chapter")
	}
	if h.g.Scene().Name != SceneMenu {
		t.Fatalf("expected the chapter route, got %s", h.g.Scene().Name)
	}
	h.frames(2, engine.Input{})
	if h.menuTile(11).Disabled || h.menuTile(11).Hidden {
		t.Fatal("the ice caps' first stop should be open and on screen")
	}
	if b := h.button("Mars map"); b == nil {
		t.Fatal("a route leads back to the map")
	}
	h.clickButton("Mars map")
	if h.g.Scene().Name != SceneMap {
		t.Fatalf("expected the Mars map, got %s", h.g.Scene().Name)
	}
}
