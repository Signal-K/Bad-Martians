package game

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/signal-k/haphazard-martians/engine"
)

// The account page (MAR release): sync state, username, daily mission history
// and the leaderboard. The browser host owns the network, so it publishes what
// the page shows under two host keys and the game only reads them:
//
//	"account"     who is signed in and how their save is syncing
//	"leaderboard" the top players, from the public hm_leaderboard view
//
// Daily history needs no network: it comes from the local campaign progress,
// which sync keeps merged across devices.

// SyncState is how the player's save relates to the backend.
type SyncState string

const (
	SyncGuest   SyncState = "guest"   // not signed in: progress stays in this browser
	SyncSynced  SyncState = "synced"  // this device matches the account
	SyncSyncing SyncState = "syncing" // a sync is running
	SyncPending SyncState = "pending" // the last sync failed; local save kept, will retry
	SyncDemo    SyncState = "demo"    // a build with no backend
)

// Account is what the host says about the signed-in player.
type Account struct {
	SignedIn bool      `json:"signed_in"`
	Email    string    `json:"email"`
	Username string    `json:"username"`
	Sync     SyncState `json:"sync"`
	LastSync int64     `json:"last_sync"` // unix milliseconds of the last good sync, 0 if none yet
	Queued   int       `json:"queued"`    // runs and labels waiting to be sent
	Error    string    `json:"error"`     // why the last username change failed
}

// LoadAccount reads the host's account blob. A host that publishes nothing
// (desktop, tests) reads as a guest.
func LoadAccount(h engine.Host) Account {
	a := Account{Sync: SyncGuest}
	if b := h.Load("account"); b != nil {
		_ = json.Unmarshal(b, &a)
	}
	if a.Sync == "" {
		a.Sync = SyncGuest
	}
	return a
}

// SyncLine is the plain-language sync state and whether it is good news.
func (a Account) SyncLine(now time.Time) (text string, good bool) {
	switch a.Sync {
	case SyncDemo:
		return "This build has no backend. Progress stays in this browser.", false
	case SyncGuest:
		return "Not signed in. Progress stays in this browser.", false
	case SyncSyncing:
		return "Syncing now…", true
	case SyncPending:
		if a.Queued > 0 {
			return fmt.Sprintf("Offline or retrying. %d waiting; your save is safe here.", a.Queued), false
		}
		return "Offline or retrying. Your save is safe here.", false
	}
	if a.LastSync > 0 {
		return "Synced " + ago(now, time.UnixMilli(a.LastSync)), true
	}
	return "Synced", true
}

// ago says how long before now t was: "just now", "5 min ago", "2 h ago".
func ago(now, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d < 45*time.Second:
		return "just now"
	case d < 90*time.Minute:
		return fmt.Sprintf("%d min ago", max(1, int(d.Minutes()+0.5)))
	case d < 36*time.Hour:
		return fmt.Sprintf("%d h ago", int(d.Hours()+0.5))
	}
	return fmt.Sprintf("%d days ago", int(d.Hours()/24))
}

// LeaderRow is one player on the leaderboard.
type LeaderRow struct {
	Username string `json:"username"`
	Score    int    `json:"score"`    // best score summed over missions won
	Stars    int    `json:"stars"`    // best stars summed over missions won
	Missions int    `json:"missions"` // missions won
}

// Leaderboard is the host's last copy of the board.
type Leaderboard struct {
	Updated int64       `json:"updated"` // unix milliseconds; 0 if never fetched
	Failed  bool        `json:"failed"`  // the last fetch failed
	Rows    []LeaderRow `json:"rows"`
}

func LoadLeaderboard(h engine.Host) Leaderboard {
	var l Leaderboard
	if b := h.Load("leaderboard"); b != nil {
		_ = json.Unmarshal(b, &l)
	}
	return l
}

// DailyEntry is the player's standing on one authored daily mission.
type DailyEntry struct {
	Mission Mission
	Played  bool // won at least once
	Stars   int
	Best    float64
	Score   int
	Claimed bool // the mission's free retry was taken
	Today   bool
}

// DailyHistory lists the daily missions that have started (newest first, at most
// limit) with the player's best result on each. Missions in the future are not
// shown: they are secret until their day.
func DailyHistory(p *Progress, today string, limit int) []DailyEntry {
	var out []DailyEntry
	for i := len(Missions) - 1; i >= 0 && len(out) < limit; i-- {
		m := Missions[i]
		if m.Date > today {
			continue
		}
		lp := p.Levels[m.Level.ID]
		out = append(out, DailyEntry{Mission: m, Played: lp.Stars > 0, Stars: lp.Stars, Best: lp.Best,
			Score: lp.Score, Claimed: p.ClaimedMissions[m.Date], Today: m.Date == today})
	}
	return out
}

// DailyStreak is how many days in a row, ending today or yesterday, the player
// has won the daily mission.
func DailyStreak(p *Progress, today string) int {
	day, err := time.Parse("2006-01-02", today)
	if err != nil {
		return 0
	}
	won := func(d time.Time) bool {
		m, ok := MissionFor(d.Format("2006-01-02"))
		return ok && p.Levels[m.Level.ID].Stars > 0
	}
	if !won(day) {
		day = day.AddDate(0, 0, -1) // today is still open: don't break the streak yet
	}
	n := 0
	for won(day) {
		n++
		day = day.AddDate(0, 0, -1)
	}
	return n
}

// ValidUsername is the rule the backend enforces too: 3 to 20 of a-z 0-9 _.
func ValidUsername(s string) bool {
	if len(s) < 3 || len(s) > 20 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
			return false
		}
	}
	return true
}

// You is the leaderboard row to highlight for username, or -1.
func (l Leaderboard) You(username string) int {
	if username == "" {
		return -1
	}
	for i, r := range l.Rows {
		if strings.EqualFold(r.Username, username) {
			return i
		}
	}
	return -1
}

func accountScene(g *engine.Game, _ any) *engine.Scene {
	s := newScene()
	addBackdrop(s)
	addTitle(s, "Account", "Your save, your daily missions and the leaderboard.")
	prog := LoadProgress(g.Host)
	acct := LoadAccount(g.Host)
	board := LoadLeaderboard(g.Host)
	today := Today()
	history := DailyHistory(prog, today, 7)
	streak := DailyStreak(prog, today)
	var clock, reload float64
	Track(g.Host, EvAccountOpened, map[string]any{"sync": string(acct.Sync), "has_username": acct.Username != ""})
	g.Host.Emit("account_open", []byte(`{}`)) // the host refreshes the profile and the board

	const top, panelH = 112.0, 340.0
	AddDraw(s, 2, func(_ engine.Camera, r engine.Renderer) {
		// Profile and sync
		drawPanel(r, 28, top, 292, panelH, "Profile")
		who := "Guest"
		if acct.SignedIn {
			who = acct.Email
		}
		r.Text(clip(who, 24), 48, top+22, sizeSmall, colMuted, engine.AlignLeft)
		name, col := acct.Username, colInk
		if name == "" {
			name, col = "No username yet", colMuted
		}
		r.Text(clip(name, 16), 48, top+44, 26, col, engine.AlignLeft)
		if acct.Error != "" {
			for i, ln := range wrapText(acct.Error, 12, 250) {
				r.Text(ln, 48, top+132+float64(i)*14, 12, colWarn, engine.AlignLeft)
			}
		}
		line, good := acct.SyncLine(time.Now())
		dot := colWarn
		if good {
			dot = colGood
		}
		if acct.Sync == SyncSyncing {
			dot = lerpColor(colChipLt, colGood, 0.5+0.5*sinf(clock*6))
		}
		r.Text("Sync", 48, top+178, sizeSmall, colNeon, engine.AlignLeft)
		r.Circle(engine.V(56, top+210), 6, dot, true)
		for i, ln := range wrapText(line, 13, 232) {
			r.Text(ln, 70, top+202+float64(i)*16, 13, colInk, engine.AlignLeft)
		}

		// Daily mission history
		drawPanel(r, 340, top, 296, panelH, "Daily missions")
		r.Text(fmt.Sprintf("Streak: %d day%s", streak, plural(streak)), 360, top+18, sizeSmall, colChipLt, engine.AlignLeft)
		if len(history) == 0 {
			r.Text("The first daily mission is coming.", 360, top+60, sizeSmall, colMuted, engine.AlignLeft)
		}
		for i, e := range history {
			y := top + 48 + float64(i)*40
			title := clip(e.Mission.Title, 20)
			r.Text(e.Mission.Date[5:], 360, y, 12, colMuted, engine.AlignLeft)
			ink := colInk
			if !e.Played {
				ink = colMuted
			}
			r.Text(title, 410, y-2, 14, ink, engine.AlignLeft)
			switch {
			case e.Played:
				drawStars(r, 560, y+6, 12, e.Stars)
				r.Text(fmt.Sprintf("%d pts · %s", e.Score, fmtTime(e.Best)), 410, y+16, 11, colMuted, engine.AlignLeft)
			case e.Today:
				r.Text("Today: not won yet", 410, y+16, 11, colNeon, engine.AlignLeft)
			default:
				r.Text("Missed", 410, y+16, 11, colMuted, engine.AlignLeft)
			}
		}

		// Leaderboard
		drawPanel(r, 656, top, 276, panelH, "Leaderboard")
		you := board.You(acct.Username)
		switch {
		case len(board.Rows) == 0 && board.Failed:
			r.Text("Could not load the board.", 676, top+30, sizeSmall, colMuted, engine.AlignLeft)
			r.Text("It will retry when you are online.", 676, top+50, 12, colMuted, engine.AlignLeft)
		case len(board.Rows) == 0:
			r.Text("No scores yet. Set a username,", 676, top+30, sizeSmall, colMuted, engine.AlignLeft)
			r.Text("win a mission and you are first.", 676, top+50, sizeSmall, colMuted, engine.AlignLeft)
		}
		for i, row := range board.Rows {
			if i >= 10 {
				break
			}
			y := top + 20 + float64(i)*28
			ink := colInk
			if i == you {
				r.Rect(668, y-4, 252, 24, colNeon.Alpha(60), true)
				ink = colNeon
			}
			r.Text(fmt.Sprint(i+1), 676, y, 14, colMuted, engine.AlignLeft)
			r.Text(clip(row.Username, 14), 706, y, 14, ink, engine.AlignLeft)
			r.Text(fmt.Sprint(row.Score), 914, y, 14, ink, engine.AlignRight)
		}
		if you >= 10 {
			r.Text(fmt.Sprintf("You are #%d", you+1), 676, top+panelH-22, 12, colNeon, engine.AlignLeft)
		} else if acct.SignedIn && acct.Username == "" {
			r.Text("Set a username to appear here.", 676, top+panelH-22, 12, colMuted, engine.AlignLeft)
		}
	})

	addBack(s, func() { g.Load(SceneMap, nil) })
	label := "Set username"
	if acct.Username != "" {
		label = "Change username"
	}
	un := AddButton(s, 48, top+84, 252, 40, label, func() { g.Host.Emit("account_username", []byte(`{}`)) })
	un.Disabled = !acct.SignedIn
	sync := AddButton(s, 48, top+270, 120, 40, "Sync now", func() { g.Host.Emit("account_sync", []byte(`{}`)) })
	sync.Disabled = !acct.SignedIn || acct.Sync == SyncSyncing
	sign := AddButton(s, 180, top+270, 120, 40, "Sign out", func() { g.Host.Emit("account_toggle", []byte(`{}`)) })
	if !acct.SignedIn {
		sign.Label, sign.Primary = "Sign in", true
		sign.Disabled = acct.Sync == SyncDemo
		sync.Hidden, un.Hidden = acct.Sync == SyncDemo, acct.Sync == SyncDemo
	}

	s.OnUpdate = func(dt float64) {
		clock += dt
		// the host answers asynchronously: pick up its newer copy about once a second
		if reload += dt; reload >= 1 {
			reload = 0
			acct, board = LoadAccount(g.Host), LoadLeaderboard(g.Host)
			un.Disabled = !acct.SignedIn
			sync.Disabled = !acct.SignedIn || acct.Sync == SyncSyncing
			sign.Label, sign.Primary = "Sign out", false
			if !acct.SignedIn {
				sign.Label, sign.Primary = "Sign in", true
			}
			un.Label = "Set username"
			if acct.Username != "" {
				un.Label = "Change username"
			}
		}
	}
	return s
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// clip shortens s to at most n runes, ending in an ellipsis when cut.
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
