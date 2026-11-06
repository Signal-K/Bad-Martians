// Command server is the Bad Martians backend: PocketBase (database, auth,
// admin UI at /_/) plus the PWA served from the same origin.
package main

import (
	"io/fs"
	"log"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"

	_ "github.com/signal-k/bad-martians/migrations"
	"github.com/signal-k/bad-martians/web"
)

func main() {
	app := pocketbase.New()
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{Automigrate: true})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// WEB_DIR serves the PWA from disk (live reload while developing);
		// otherwise the embedded copy is used.
		var site fs.FS = web.FS
		if dir := os.Getenv("WEB_DIR"); dir != "" {
			site = os.DirFS(dir)
		}
		se.Router.GET("/{path...}", apis.Static(site, false)).Bind(apis.Gzip())
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
