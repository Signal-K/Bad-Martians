// Package runcheck guards hm_runs against forged leaderboard entries (MAR-62).
// Call Register on the PocketBase app that hosts the hm_ collections: the
// standalone cmd/server does, and the shared Star Sailors backend can import it.
package runcheck

import (
	"net/http"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/signal-k/haphazard-martians/game"
)

// MaxPerMinute caps how many runs one player can submit in a minute. A real
// drive takes far longer than two seconds, so this is generous.
const MaxPerMinute = 20

// Register rejects implausible or too-frequent hm_runs creates.
func Register(app core.App) {
	app.OnRecordCreateRequest("hm_runs").BindFunc(func(e *core.RecordRequestEvent) error {
		r := e.Record
		if e.Auth != nil {
			n, err := e.App.CountRecords("hm_runs",
				dbx.HashExp{"user": e.Auth.Id},
				dbx.NewExp("created > {:t}", dbx.Params{"t": types.NowDateTime().Add(-time.Minute).String()}))
			if err == nil && n >= MaxPerMinute {
				return apis.NewApiError(http.StatusTooManyRequests, "Too many runs; slow down.", nil)
			}
		}
		if why := game.CheckRun(game.RunClaim{
			Level: r.GetInt("level"), Won: r.GetBool("won"), Stars: r.GetInt("stars"),
			Samples: r.GetInt("samples"), Score: r.GetInt("score"), Cost: r.GetInt("cost"),
			Time: r.GetFloat("time"), Design: []byte(r.GetString("design")),
		}); why != "" {
			e.App.Logger().Warn("hm_runs rejected", "user", r.GetString("user"), "why", why)
			return apis.NewBadRequestError("Run rejected: "+why, nil)
		}
		return e.Next()
	})
}
