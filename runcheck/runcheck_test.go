package runcheck_test

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/signal-k/haphazard-martians/engine"
	"github.com/signal-k/haphazard-martians/game"
	_ "github.com/signal-k/haphazard-martians/migrations"
	"github.com/signal-k/haphazard-martians/runcheck"
)

const (
	userID     = "player00000001a"
	goodDesign = `[{"x":0,"y":0,"k":"wheel"},{"x":1,"y":0,"k":"frame"},{"x":2,"y":0,"k":"frame"},{"x":3,"y":0,"k":"motor"},{"x":1,"y":1,"k":"frame"},{"x":2,"y":1,"k":"mast"}]`
)

// drive plays the starter rover through level 1 as the drive scene does and
// returns what the client would record: the inputs and the result.
func drive() (inputs string, tm float64, samples, stars, score int) {
	l, d := game.GetLevel(1), game.StarterDesign()
	r := game.NewRun(l, d)
	r.Speed = game.DriveSpeed
	rec := &game.InputRecorder{}
	for i := 0; i < 60*130 && r.Outcome == game.Running; i++ {
		th, lean := rec.Next(1, 0)
		r.Lean = lean
		r.Step(engine.FixedDT, th)
	}
	return rec.Encode(), math.Round(r.Time*10) / 10, r.SamplesGot(), r.Stars(), r.Score().Total()
}

func run(score int, inputs string, tm float64, samples, stars int) string {
	d := game.StarterDesign()
	return `{"user":"` + userID + `","level":1,"won":true,"stars":` + strconv.Itoa(stars) + `,"time":` + strconv.FormatFloat(tm, 'f', 1, 64) +
		`,"samples":` + strconv.Itoa(samples) + `,"score":` + strconv.Itoa(score) + `,"cost":` + strconv.Itoa(d.Cost()) +
		`,"inputs":"` + inputs + `","design":` + goodDesign + `}`
}

// post sends one run as the player; floodFirst pre-fills that many saved runs this minute.
func post(t *testing.T, name, body string, status int, floodFirst int, want string) {
	headers := map[string]string{}
	(&tests.ApiScenario{
		Name:    name,
		Method:  http.MethodPost,
		URL:     "/api/collections/hm_runs/records",
		Body:    strings.NewReader(body),
		Headers: headers,
		TestAppFactory: func(tb testing.TB) *tests.TestApp {
			app, err := tests.NewTestApp()
			if err != nil {
				tb.Fatal(err)
			}
			runcheck.Register(app)
			users, _ := app.FindCollectionByNameOrId("users")
			u := core.NewRecord(users)
			u.Id = userID
			u.SetEmail("a@example.com")
			u.SetPassword("password12345")
			if err := app.Save(u); err != nil {
				tb.Fatal(err)
			}
			tok, _ := u.NewAuthToken()
			headers["Authorization"] = tok
			runs, _ := app.FindCollectionByNameOrId("hm_runs")
			for i := 0; i < floodFirst; i++ {
				r := core.NewRecord(runs)
				r.Set("user", userID)
				r.Set("level", 1)
				if err := app.Save(r); err != nil {
					tb.Fatal(err)
				}
			}
			return app
		},
		ExpectedStatus:  status,
		ExpectedContent: []string{want},
	}).Test(t)
}

func TestRunsAreChecked(t *testing.T) {
	in, tm, samples, stars, score := drive()
	post(t, "honest run is saved", run(score, in, tm, samples, stars), http.StatusOK, 0, `"collectionName":"hm_runs"`)
	post(t, "forged score is refused", run(99999, in, tm, samples, stars), http.StatusBadRequest, 0, "Run rejected")
	post(t, "a win with no recorded drive is refused", run(score, "", tm, samples, stars), http.StatusBadRequest, 0, "no recorded inputs")
	post(t, "a win the drive does not back up is refused", run(score, "PGQA", tm, samples, stars), http.StatusBadRequest, 0, "does not reach the finish")
	post(t, "flooding is refused", run(score, in, tm, samples, stars), http.StatusTooManyRequests, runcheck.MaxPerMinute, "Too many runs")
}
