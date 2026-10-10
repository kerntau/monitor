package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		systems, err := app.FindCollectionByNameOrId("systems")
		if err != nil {
			return err
		}
		monitors, err := app.FindCollectionByNameOrId("network_monitors")
		if err != nil {
			return err
		}
		config := core.NewBaseCollection("chenv_config")
		config.Fields.Add(&core.JSONField{Name: "value", MaxSize: 16384})
		if err = app.Save(config); err != nil {
			return err
		}
		for _, item := range []struct {
			name   string
			target *core.Collection
			field  string
		}{
			{"chenv_systems", systems, "system"}, {"chenv_monitors", monitors, "monitor"},
		} {
			c := core.NewBaseCollection(item.name)
			c.Fields.Add(&core.RelationField{Name: item.field, CollectionId: item.target.Id, MaxSelect: 1, Required: true, CascadeDelete: true}, &core.TextField{Name: "name", Max: 120}, &core.BoolField{Name: "public"}, &core.NumberField{Name: "sort", OnlyInt: true})
			c.Indexes = []string{"CREATE UNIQUE INDEX idx_" + item.name + "_relation ON " + item.name + " (" + item.field + ")"}
			if err = app.Save(c); err != nil {
				return err
			}
		}
		r := core.NewRecord(config)
		r.Set("value", map[string]any{"name": "chenv", "favicon": "/static/chenv-avatar.png", "theme": "system", "hideAddresses": true})
		if err = app.Save(r); err != nil {
			return err
		}
		for _, item := range []struct{ source, dest, field string }{{"systems", "chenv_systems", "system"}, {"network_monitors", "chenv_monitors", "monitor"}} {
			rows, err := app.FindAllRecords(item.source)
			if err != nil {
				return err
			}
			c, err := app.FindCollectionByNameOrId(item.dest)
			if err != nil {
				return err
			}
			for i, row := range rows {
				r := core.NewRecord(c)
				r.Set(item.field, row.Id)
				r.Set("sort", i)
				if item.source == "systems" {
					r.Set("name", row.GetString("name"))
					r.Set("public", row.Id == "m9gh5qbz6fc3bw3")
				} else {
					r.Set("name", row.GetString("target"))
					r.Set("public", true)
				}
				if err = app.Save(r); err != nil {
					return err
				}
			}
		}
		return nil
	}, func(app core.App) error {
		for _, name := range []string{"chenv_monitors", "chenv_systems", "chenv_config"} {
			c, err := app.FindCollectionByNameOrId(name)
			if err != nil {
				return err
			}
			if err = app.Delete(c); err != nil {
				return err
			}
		}
		return nil
	})
}
