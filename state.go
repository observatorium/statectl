package main

import (
	"os"

	"github.com/observatorium/statectl/pkg/extkingpin"
	"github.com/observatorium/statectl/pkg/project"
	"github.com/pkg/errors"
)

func registerPropose(app *extkingpin.App) {
	cmd := app.Command("propose", "Propose change of cluster state.")

	//svc := cmd.Flag("svc", "Service to propose cluster state change to.").Short('s').Required().String()
	//cluster := cmd.Flag("cluster", "Cluster to propose service state change to. Cluster has to be defined in project configuration.").Short('c').Required().String()
	//tag := cmd.Flag("tag", "Commit sha, branch or tag that identifies the version of the configuration repo you want to propose to deploy.").Short('t').Default("master").Required().String()

	cmd.Run(func(prj *project.Project) error {
		return errors.New("not implemented")
	})
}

func registerDiff(app *extkingpin.App) {
	cmd := app.Command("diff", "Print diff between states.")
	baseTag := cmd.Arg("base-state-sha", "Git commit or tag of state git repository to use as base.").Required().String()
	newTag := cmd.Arg("new-state-sha", "Git commit or tag of state git repository to compare with.").Required().String()

	cmd.Run(func(prj *project.Project) error {
		return prj.DiffState(os.Stdout, project.Ref(*baseTag), project.Ref(*newTag))
	})
}
