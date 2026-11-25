package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Usernames and the leaderboard (first release). A profile is the only public
// thing about a player: the leaderboard shows usernames, never emails, and
// players without a username are not on it.
func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		p := core.NewBaseCollection("hm_profiles")
		p.Fields.Add(
			&core.RelationField{Name: "user", CollectionId: users.Id, Required: true, MaxSelect: 1, CascadeDelete: true},
			// 3 to 20 of a-z 0-9 _, matching game.ValidUsername.
			&core.TextField{Name: "username", Required: true, Min: 3, Max: 20, Pattern: `^[a-z0-9_]+$`},
			&core.AutodateField{Name: "created", OnCreate: true},
		)
		p.ListRule, p.ViewRule = types.Pointer(""), types.Pointer("")
		p.CreateRule = types.Pointer("@request.auth.id != '' && @request.body.user = @request.auth.id")
		p.UpdateRule = types.Pointer("user = @request.auth.id && (@request.body.user:isset = false || @request.body.user = @request.auth.id)")
		p.AddIndex("idx_hm_profiles_user", true, "user", "")
		p.AddIndex("idx_hm_profiles_username", true, "username COLLATE NOCASE", "")
		if err := app.Save(p); err != nil {
			return err
		}

		// Best result per player per mission, summed. A run only counts once it is won.
		board := core.NewViewCollection("hm_leaderboard")
		board.ViewQuery = `
SELECT p.user AS id,
       p.username AS username,
       CAST(SUM(b.score) AS INTEGER) AS score,
       CAST(SUM(b.stars) AS INTEGER) AS stars,
       CAST(COUNT(*) AS INTEGER) AS missions
FROM hm_profiles p
JOIN (SELECT user, level, MAX(score) AS score, MAX(stars) AS stars
      FROM hm_runs WHERE won = 1 GROUP BY user, level) b ON b.user = p.user
GROUP BY p.user`
		board.ListRule, board.ViewRule = types.Pointer(""), types.Pointer("")
		return app.Save(board)
	}, func(app core.App) error {
		for _, n := range []string{"hm_leaderboard", "hm_profiles"} {
			if c, err := app.FindCollectionByNameOrId(n); err == nil {
				if err := app.Delete(c); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
