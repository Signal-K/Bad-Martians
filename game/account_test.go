package game

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/signal-k/haphazard-martians/engine"
)

func TestValidUsername(t *testing.T) {
	for s, want := range map[string]bool{"ab": false, "abc": true, "rover_9": true, "Rover": false, "a b": false,
		"twenty_characters_ok": true, "twenty_one_characters": false, "é_é_é": false} {
		if ValidUsername(s) != want {
			t.Errorf("ValidUsername(%q) = %v, want %v", s, !want, want)
		}
	}
}

func TestAccountDefaultsToGuest(t *testing.T) {
	h := engine.NewMemHost()
	if a := LoadAccount(h); a.Sync != SyncGuest || a.SignedIn {
		t.Fatalf("empty host should read as a guest, got %+v", a)
	}
	b, _ := json.Marshal(Account{SignedIn: true, Username: "zed", Sync: SyncPending, Queued: 3})
	h.Save("account", b)
	a := LoadAccount(h)
	if text, good := a.SyncLine(time.Now()); good || text == "" {
		t.Fatalf("pending sync is not good news: %q %v", text, good)
	}
}

func TestSyncLineSynced(t *testing.T) {
	now := time.Now()
	a := Account{SignedIn: true, Sync: SyncSynced, LastSync: now.Add(-5 * time.Minute).UnixMilli()}
	if text, good := a.SyncLine(now); !good || text != "Synced 5 min ago" {
		t.Fatalf("got %q %v", text, good)
	}
}

func TestDailyHistoryHidesTheFutureAndShowsBest(t *testing.T) {
	if len(Missions) < 3 {
		t.Skip("needs authored missions")
	}
	mid := Missions[1]
	p := &Progress{Levels: map[int]LevelProgress{mid.Level.ID: {Stars: 2, Best: 31.5, Score: 700}}, ClaimedMissions: map[string]bool{mid.Date: true}}
	h := DailyHistory(p, mid.Date, 10)
	if len(h) != 2 || h[0].Mission.Date != mid.Date || !h[0].Today || !h[0].Played || h[0].Score != 700 || !h[0].Claimed {
		t.Fatalf("unexpected history %+v", h)
	}
	if h[1].Played {
		t.Fatal("an unplayed mission must not read as played")
	}
	if got := DailyHistory(p, mid.Date, 1); len(got) != 1 {
		t.Fatalf("limit ignored: %d", len(got))
	}
}

func TestDailyStreak(t *testing.T) {
	days := []string{"2026-11-23", "2026-11-24", "2026-11-25", "2026-11-26"}
	p := &Progress{Levels: map[int]LevelProgress{}}
	for _, d := range days[:3] {
		m, ok := MissionFor(d)
		if !ok {
			t.Skip("needs authored missions for " + d)
		}
		p.Levels[m.Level.ID] = LevelProgress{Stars: 1}
	}
	if n := DailyStreak(p, days[2]); n != 3 {
		t.Fatalf("streak %d, want 3", n)
	}
	// today not won yet: yesterday's streak still stands
	if n := DailyStreak(p, days[3]); n != 3 {
		t.Fatalf("open day broke the streak: %d", n)
	}
	m, _ := MissionFor(days[1])
	delete(p.Levels, m.Level.ID)
	if n := DailyStreak(p, days[2]); n != 1 {
		t.Fatalf("gap should end the streak, got %d", n)
	}
}

func TestLeaderboardYouIsCaseInsensitive(t *testing.T) {
	l := Leaderboard{Rows: []LeaderRow{{Username: "ann"}, {Username: "Bob"}}}
	if l.You("bob") != 1 || l.You("") != -1 || l.You("zed") != -1 {
		t.Fatal("You lookup wrong")
	}
}

func TestAccountSceneOpensAndRefreshes(t *testing.T) {
	hs := newHarness(t)
	hs.g.Load(SceneAccount, nil)
	hs.frames(2, engine.Input{})
	if hs.g.Scene().Name != SceneAccount {
		t.Fatal("account scene did not open")
	}
	if hs.count("account_open") != 1 || hs.count("track") != 1 {
		t.Fatal("opening the account page should ask the host to refresh and track once")
	}
	hs.frames(90, engine.Input{}) // a second of frames reloads the host's copy without error
	hs.clickButton("Sign in")
	if hs.count("account_toggle") != 1 {
		t.Fatal("Sign in should ask the host to start sign-in")
	}
}
