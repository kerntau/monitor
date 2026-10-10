package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("_superusers")
		if err != nil {
			return err
		}
		c := core.NewBaseCollection("chenv_universal_tokens")
		c.Fields.Add(&core.RelationField{Name: "user", CollectionId: users.Id, MaxSelect: 1, Required: true, CascadeDelete: true}, &core.TextField{Name: "token", Required: true, Max: 64})
		c.Indexes = []string{"CREATE UNIQUE INDEX idx_chenv_tokens_user ON chenv_universal_tokens (user)", "CREATE UNIQUE INDEX idx_chenv_tokens_token ON chenv_universal_tokens (token)"}
		return app.Save(c)
	}, func(app core.App) error {
		c, err := app.FindCollectionByNameOrId("chenv_universal_tokens")
		if err != nil {
			return err
		}
		return app.Delete(c)
	})
}
