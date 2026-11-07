// Package migrations creates the Haphazard Martians collections. Auth is PocketBase's
// built-in `users` collection for now (Clerk replaces it in staging/prod).
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		owner := func() *core.RelationField {
			return &core.RelationField{Name: "user", CollectionId: users.Id, Required: true, MaxSelect: 1, CascadeDelete: true}
		}
		createRule := types.Pointer("@request.auth.id != '' && @request.body.user = @request.auth.id")

		// One row per attempt. Public read: it doubles as the leaderboard.
		runs := core.NewBaseCollection("hm_runs")
		runs.Fields.Add(
			owner(),
			&core.NumberField{Name: "level", Required: true, OnlyInt: true, Min: types.Pointer(1.0)},
			&core.BoolField{Name: "won"},
			&core.NumberField{Name: "stars", OnlyInt: true, Min: types.Pointer(0.0), Max: types.Pointer(3.0)},
			&core.NumberField{Name: "time", Min: types.Pointer(0.0)},
			&core.NumberField{Name: "samples", OnlyInt: true, Min: types.Pointer(0.0)},
			&core.NumberField{Name: "score", OnlyInt: true, Min: types.Pointer(0.0)},
			&core.NumberField{Name: "cost", OnlyInt: true, Min: types.Pointer(0.0)},
			&core.JSONField{Name: "design", MaxSize: 64 << 10},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		runs.ListRule, runs.ViewRule, runs.CreateRule = types.Pointer(""), types.Pointer(""), createRule
		runs.AddIndex("idx_hm_runs_level", false, "level,won,score", "")
		if err := app.Save(runs); err != nil {
			return err
		}

		// AI4Mars-style terrain labels. Each player labels a segment once;
		// private to the player, aggregated server-side for consensus.
		labels := core.NewBaseCollection("hm_labels")
		labels.Fields.Add(
			owner(),
			&core.NumberField{Name: "level", Required: true, OnlyInt: true, Min: types.Pointer(1.0)},
			&core.NumberField{Name: "segment", OnlyInt: true, Min: types.Pointer(0.0)},
			&core.NumberField{Name: "x0"},
			&core.NumberField{Name: "x1"},
			&core.SelectField{Name: "class", Required: true, MaxSelect: 1, Values: []string{"soil", "bedrock", "sand", "big_rock"}},
			&core.BoolField{Name: "gold"},
			&core.BoolField{Name: "correct"},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		labels.ListRule = types.Pointer("user = @request.auth.id")
		labels.ViewRule = labels.ListRule
		labels.CreateRule = createRule
		labels.AddIndex("idx_hm_labels_unique", true, "user,level,segment", "")
		return app.Save(labels)
	}, func(app core.App) error {
		for _, n := range []string{"hm_labels", "hm_runs"} {
			if c, err := app.FindCollectionByNameOrId(n); err == nil {
				if err := app.Delete(c); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
