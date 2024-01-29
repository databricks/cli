package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/deploy"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/metadata"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/bundle/python"
	"github.com/databricks/cli/bundle/scripts"
)

// The deploy phase deploys artifacts and resources.
func Deploy() bundle.Mutator {
	deployMutator := bundle.Seq(
		scripts.Execute(config.ScriptPreDeploy),
		lock.Acquire(),
		bundle.Defer(
			bundle.Seq(
				deploy.CheckRunningResource(),
				mutator.ValidateGitDetails(),
				libraries.MatchWithArtifacts(),
				artifacts.CleanUp(),
				artifacts.UploadAll(),
				python.TransformWheelTask(),
				files.Upload(),
				permissions.ApplyWorkspaceRootPermissions(),
				terraform.Interpolate(),
				terraform.Write(),
				terraform.StatePull(),
				bundle.Defer(
					terraform.Apply(),
					bundle.Seq(
						terraform.StatePush(),
						terraform.Load(),
						metadata.Compute(),
						metadata.Upload(),
					),
				),
			),
			lock.Release(lock.GoalDeploy),
		),
		scripts.Execute(config.ScriptPostDeploy),
		bundle.LogString("Deployment complete!"),
	)

	return newPhase(
		"deploy",
		[]bundle.Mutator{deployMutator},
	)
}
