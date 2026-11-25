package runcheck_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/signal-k/haphazard-martians/game"
	_ "github.com/signal-k/haphazard-martians/migrations"
	"github.com/signal-k/haphazard-martians/runcheck"
)

const (
	userID     = "player00000001a"
	goodDesign = `[{"x":0,"y":0,"k":"wheel"},{"x":1,"y":0,"k":"frame"},{"x":2,"y":0,"k":"frame"},{"x":3,"y":0,"k":"motor"},{"x":1,"y":1,"k":"frame"},{"x":2,"y":1,"k":"mast"}]`
)

func run(score int) string {
	d := game.StarterDesign()
	return `{"user":"` + userID + `","level":1,"won":true,"stars":1,"time":30,"samples":1,"score":` +
		strconv.Itoa(score) + `,"cost":` + strconv.Itoa(d.Cost()) + `,"design":` + goodDesign + `}`
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
	score := game.ScoreFor(game.GetLevel(1), true, 1, 30).Total()
	post(t, "honest run is saved", run(score), http.StatusOK, 0, `"collectionName":"hm_runs"`)
	post(t, "forged score is refused", run(99999), http.StatusBadRequest, 0, "Run rejected")
	post(t, "flooding is refused", run(score), http.StatusTooManyRequests, runcheck.MaxPerMinute, "Too many runs")
}
