package main

import "github.com/observatorium/statectl/pkg/extkingpin"

func registerPropose(app *extkingpin.App) {
	_ = app.Command("propose", "Propose change of cluster state.")

}
