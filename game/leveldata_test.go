package game

import (
	"strings"
	"testing"
	"testing/fstest"
)

const goodLevel = `{"id":1,"name":"A","blurb":"b","biome":"plains","length":80,"seed":1,"credits":12,"par":14,"hill":1,"traction":1,"mix":{"soil":1}}`

func TestShippedDataLoads(t *testing.T) {
	if len(LevelDefs) != 20 {
		t.Fatalf("want 20 campaign levels, got %d", len(LevelDefs))
	}
	for i, d := range LevelDefs {
		if d.ID != i+1 || d.Biome == "" || d.Traction <= 0 {
			t.Errorf("level %d: %+v", i+1, d)
		}
	}
	// The last level is harder than the first.
	if LevelDefs[len(LevelDefs)-1].Difficulty() <= LevelDefs[0].Difficulty() {
		t.Error("final level should be harder than the first")
	}
	m, ok := MissionFor("2026-11-08")
	if !ok || m.Level.ID < 100 || m.Reward.FreeRetry != 1 {
		t.Fatalf("sample mission: %+v ok=%v", m, ok)
	}
	if GetLevel(m.Level.ID) == nil {
		t.Fatal("mission level cannot be built by id")
	}
}

func TestSampleMissionBeatable(t *testing.T) {
	m, _ := MissionFor("2026-11-08")
	d := referenceDesign()
	if d.Cost() > m.Level.Credits+8 {
		t.Fatal("reference design unaffordable")
	}
	if r := drive(BuildLevel(m.Level), d, 120); r.Outcome != Won {
		t.Errorf("mission not beatable by the reference rover: %s", r.Outcome.Message())
	}
}

func TestParseLevelRejectsBadFiles(t *testing.T) {
	cases := map[string]struct{ from, to, want string }{
		"typo field":    {`"hill":1`, `"hil":1`, "unknown field"},
		"bad biome":     {`"plains"`, `"swamp"`, "biome"},
		"bad class":     {`"soil":1`, `"mud":1`, "unknown class"},
		"empty mix":     {`"soil":1`, `"soil":0`, "positive weight"},
		"short length":  {`"length":80`, `"length":10`, "length"},
		"huge traction": {`"traction":1`, `"traction":5`, "traction"},
		"no name":       {`"name":"A"`, `"name":""`, "name"},
		"no par":        {`"par":14`, `"par":0`, "par"},
		"zero id":       {`"id":1`, `"id":0`, "id"},
		"many gaps":     {`"traction":1`, `"traction":1,"gaps":9`, "gaps"},
		"wide bay":      {`"traction":1`, `"traction":1,"max_width":13`, "max_width"},
		"two parts":     {`"traction":1`, `"traction":1,"max_parts":2`, "max_parts"},
		"neg motors":    {`"traction":1`, `"traction":1,"max_motors":-1`, "max_motors"},
	}
	if _, err := ParseLevel([]byte(goodLevel)); err != nil {
		t.Fatalf("good level rejected: %v", err)
	}
	for name, c := range cases {
		_, err := ParseLevel([]byte(strings.Replace(goodLevel, c.from, c.to, 1)))
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want it to mention %q", name, err, c.want)
		}
	}
	if _, err := ParseLevel([]byte(goodLevel + goodLevel)); err == nil {
		t.Error("trailing data accepted")
	}
}

func TestLoadLevelsRequiresContiguousIDs(t *testing.T) {
	two := strings.Replace(goodLevel, `"id":1`, `"id":3`, 1)
	_, err := LoadLevels(fstest.MapFS{
		"data/levels/1.json": {Data: []byte(goodLevel)},
		"data/levels/3.json": {Data: []byte(two)},
	})
	if err == nil || !strings.Contains(err.Error(), "without gaps") {
		t.Fatalf("err = %v", err)
	}
}

func TestLoadMissionsValidation(t *testing.T) {
	mission := func(date string, id int) string {
		lvl := strings.Replace(goodLevel, `"id":1`, `"id":`+itoa(id), 1)
		return `{"date":"` + date + `","title":"T","story":"s","reward":{"credits":2,"free_retry":1},"level":` + lvl + `}`
	}
	cases := map[string]struct {
		files map[string]string
		want  string
	}{
		"bad date":       {map[string]string{"a": mission("11/09/2026", 101)}, "YYYY-MM-DD"},
		"campaign id":    {map[string]string{"a": mission("2026-11-08", 5)}, "100 or more"},
		"duplicate date": {map[string]string{"a": mission("2026-11-08", 101), "b": mission("2026-11-08", 102)}, "date"},
		"duplicate id":   {map[string]string{"a": mission("2026-11-08", 101), "b": mission("2026-11-10", 101)}, "level id"},
		"big reward":     {map[string]string{"a": strings.Replace(mission("2026-11-08", 101), `"credits":2`, `"credits":99`, 1)}, "reward"},
	}
	for name, c := range cases {
		fsys := fstest.MapFS{}
		for k, v := range c.files {
			fsys["data/missions/"+k+".json"] = &fstest.MapFile{Data: []byte(v)}
		}
		if _, err := LoadMissions(fsys); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want %q", name, err, c.want)
		}
	}
}

func TestTractionScalesFriction(t *testing.T) {
	d := LevelDefs[0]
	base := BuildLevel(d).Terrain.Materials[Soil].Friction
	d.Traction = 0.5
	if got := BuildLevel(d).Terrain.Materials[Soil].Friction; got != base*0.5 {
		t.Errorf("friction %v, want %v", got, base*0.5)
	}
}

func itoa(n int) string {
	s := ""
	for ; n > 0; n /= 10 {
		s = string(rune('0'+n%10)) + s
	}
	return s
}
