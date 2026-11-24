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
		c := core.NewBaseCollection("hm_saves")
		c.Fields.Add(
			&core.RelationField{Name: "user", CollectionId: users.Id, Required: true, MaxSelect: 1, CascadeDelete: true},
			&core.TextField{Name: "device", Required: true, Max: 64},
			&core.JSONField{Name: "state", MaxSize: 256 << 10},
		)
		c.ListRule = types.Pointer("user = @request.auth.id")
		c.ViewRule = c.ListRule
		c.CreateRule = types.Pointer("@request.auth.id != '' && @request.body.user = @request.auth.id")
		c.UpdateRule = types.Pointer("user = @request.auth.id && (@request.body.user:isset = false || @request.body.user = @request.auth.id)")
		c.AddIndex("idx_hm_saves_user_device", true, "user,device", "")
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("hm_saves")
		if err != nil {
			return nil
		}
		return app.Delete(c)
	})
}
