package game

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

//go:embed data/levels/*.json data/missions/*.json
var dataFS embed.FS

// Biomes are the terrain families a level can declare.
var Biomes = []string{"plains", "dunes", "boulders", "ridge", "crater", "ice"}

type levelFile struct {
	ID       int                `json:"id"`
	Name     string             `json:"name"`
	Blurb    string             `json:"blurb"`
	Biome    string             `json:"biome"`
	Length   float64            `json:"length"`
	Seed     uint64             `json:"seed"`
	Credits  int                `json:"credits"`
	Par      float64            `json:"par"`
	Hill     float64            `json:"hill"`
	Traction float64            `json:"traction"`
	Mix      map[string]float64 `json:"mix"`
}

// Reward is what a mission pays out on completion.
type Reward struct {
	Credits   int `json:"credits"`
	FreeRetry int `json:"free_retry"`
}

// Mission is a hand-authored level with a date and a reward.
type Mission struct {
	Date  string
	Title string
	Story string
	Reward
	Level LevelDef
}

type missionFile struct {
	Date   string    `json:"date"`
	Title  string    `json:"title"`
	Story  string    `json:"story"`
	Reward Reward    `json:"reward"`
	Level  levelFile `json:"level"`
}

// ParseLevel decodes and validates one level file. Unknown fields are errors,
// so a typo in a hand-written file is caught rather than ignored.
func ParseLevel(data []byte) (LevelDef, error) {
	var f levelFile
	if err := strictJSON(data, &f); err != nil {
		return LevelDef{}, err
	}
	return f.def()
}

// ParseMission decodes and validates one mission file.
func ParseMission(data []byte) (Mission, error) {
	var f missionFile
	if err := strictJSON(data, &f); err != nil {
		return Mission{}, err
	}
	if _, err := time.Parse("2006-01-02", f.Date); err != nil {
		return Mission{}, fmt.Errorf("date %q: want YYYY-MM-DD", f.Date)
	}
	if strings.TrimSpace(f.Title) == "" {
		return Mission{}, fmt.Errorf("title is required")
	}
	if f.Reward.Credits < 0 || f.Reward.FreeRetry < 0 || f.Reward.Credits > 50 || f.Reward.FreeRetry > 5 {
		return Mission{}, fmt.Errorf("reward out of range: %+v", f.Reward)
	}
	def, err := f.Level.def()
	if err != nil {
		return Mission{}, fmt.Errorf("level: %w", err)
	}
	return Mission{Date: f.Date, Title: f.Title, Story: f.Story, Reward: f.Reward, Level: def}, nil
}

func strictJSON(data []byte, v any) error {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	if dec.More() {
		return fmt.Errorf("trailing data after the JSON object")
	}
	return nil
}

func (f levelFile) def() (LevelDef, error) {
	d := LevelDef{ID: f.ID, Name: f.Name, Blurb: f.Blurb, Biome: f.Biome, Length: f.Length,
		Seed: f.Seed, Credits: f.Credits, Par: f.Par, Hill: f.Hill, Traction: f.Traction}
	switch {
	case f.ID <= 0:
		return d, fmt.Errorf("id must be positive")
	case strings.TrimSpace(f.Name) == "":
		return d, fmt.Errorf("level %d: name is required", f.ID)
	case f.Length < 40 || f.Length > 400:
		return d, fmt.Errorf("level %d: length %.0f outside 40..400 m", f.ID, f.Length)
	case f.Credits < 4 || f.Credits > 60:
		return d, fmt.Errorf("level %d: credits %d outside 4..60", f.ID, f.Credits)
	case f.Par <= 0:
		return d, fmt.Errorf("level %d: par must be positive", f.ID)
	case f.Hill < 0 || f.Hill > 8:
		return d, fmt.Errorf("level %d: hill %.1f outside 0..8 m", f.ID, f.Hill)
	}
	if d.Traction == 0 {
		d.Traction = 1
	}
	if d.Traction < 0.3 || d.Traction > 2 {
		return d, fmt.Errorf("level %d: traction %.2f outside 0.3..2", f.ID, d.Traction)
	}
	ok := false
	for _, b := range Biomes {
		ok = ok || b == f.Biome
	}
	if !ok {
		return d, fmt.Errorf("level %d: biome %q not one of %v", f.ID, f.Biome, Biomes)
	}
	total := 0.0
	for k, w := range f.Mix {
		c, known := ClassFromKey(k)
		if !known {
			return d, fmt.Errorf("level %d: unknown class %q in mix", f.ID, k)
		}
		if w < 0 {
			return d, fmt.Errorf("level %d: negative weight for %q", f.ID, k)
		}
		d.Mix[c] = w
		total += w
	}
	if total <= 0 {
		return d, fmt.Errorf("level %d: mix needs at least one positive weight", f.ID)
	}
	return d, nil
}

// LoadLevels reads every level file in fsys (data/levels), sorted by id, and
// requires ids to run 1..n so progress can index by id.
func LoadLevels(fsys fs.FS) ([]LevelDef, error) {
	names, err := fs.Glob(fsys, "data/levels/*.json")
	if err != nil || len(names) == 0 {
		return nil, fmt.Errorf("no level files found")
	}
	var defs []LevelDef
	for _, n := range names {
		b, err := fs.ReadFile(fsys, n)
		if err != nil {
			return nil, err
		}
		d, err := ParseLevel(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		defs = append(defs, d)
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].ID < defs[j].ID })
	for i, d := range defs {
		if d.ID != i+1 {
			return nil, fmt.Errorf("level ids must run 1..%d without gaps; found %d at position %d", len(defs), d.ID, i+1)
		}
	}
	return defs, nil
}

// LoadMissions reads every mission file in fsys (data/missions), sorted by
// date. Dates and level ids must be unique, and mission level ids must not
// collide with the campaign (they start at 100).
func LoadMissions(fsys fs.FS) ([]Mission, error) {
	names, _ := fs.Glob(fsys, "data/missions/*.json")
	var ms []Mission
	dates, ids := map[string]string{}, map[int]string{}
	for _, n := range names {
		b, err := fs.ReadFile(fsys, n)
		if err != nil {
			return nil, err
		}
		m, err := ParseMission(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		if m.Level.ID < 100 {
			return nil, fmt.Errorf("%s: mission level id %d must be 100 or more", n, m.Level.ID)
		}
		if p, dup := dates[m.Date]; dup {
			return nil, fmt.Errorf("%s: date %s already used by %s", n, m.Date, p)
		}
		if p, dup := ids[m.Level.ID]; dup {
			return nil, fmt.Errorf("%s: level id %d already used by %s", n, m.Level.ID, p)
		}
		dates[m.Date], ids[m.Level.ID] = n, n
		ms = append(ms, m)
	}
	sort.Slice(ms, func(i, j int) bool { return ms[i].Date < ms[j].Date })
	return ms, nil
}

// MissionFor returns the mission for a date (YYYY-MM-DD), if one is authored.
func MissionFor(date string) (Mission, bool) {
	for _, m := range Missions {
		if m.Date == date {
			return m, true
		}
	}
	return Mission{}, false
}

// Obstacle is the visible obstacle load of a level, 0..1: how much of the
// course is rocky and how steep it is.
func (d LevelDef) Obstacle() float64 {
	total := 0.0
	for _, w := range d.Mix {
		total += w
	}
	rock := d.Mix[BigRock] / total
	sand := d.Mix[Sand] / total
	hill := d.Hill / 4
	if hill > 1 {
		hill = 1
	}
	return 0.4*hill + 0.35*rock + 0.25*sand
}

// Difficulty is the weighted average of obstacle load, length and slipperiness.
func (d LevelDef) Difficulty() float64 {
	length := d.Length / 180
	if length > 1 {
		length = 1
	}
	slip := 2 - d.Traction
	if slip < 0 {
		slip = 0
	}
	if slip > 1 {
		slip = 1
	}
	return 0.6*d.Obstacle() + 0.25*length + 0.15*slip
}

func mustLoad() ([]LevelDef, []Mission) {
	defs, err := LoadLevels(dataFS)
	if err != nil {
		panic("levels: " + err.Error())
	}
	ms, err := LoadMissions(dataFS)
	if err != nil {
		panic("missions: " + err.Error())
	}
	return defs, ms
}

// LevelDefs is the campaign, loaded from data/levels at startup. Missions are
// the dated, hand-authored extras from data/missions.
var LevelDefs, Missions = mustLoad()
