package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// hm_runs.inputs holds the recorded stick inputs of a won drive, base64 of
// run-length (count, throttle, lean) bytes. The server replays them (MAR-62).
func init() {
	m.Register(func(app core.App) error {
		runs, err := app.FindCollectionByNameOrId("hm_runs")
		if err != nil {
			return err
		}
		if runs.Fields.GetByName("inputs") == nil {
			runs.Fields.Add(&core.TextField{Name: "inputs", Max: 40000})
		}
		return app.Save(runs)
	}, func(app core.App) error {
		runs, err := app.FindCollectionByNameOrId("hm_runs")
		if err != nil {
			return nil
		}
		runs.Fields.RemoveByName("inputs")
		return app.Save(runs)
	})
}
