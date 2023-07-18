package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/libraries"
)

// The deploy phase deploys artifacts and resources.
func Deploy() bundle.Mutator {
	deployMutator := bundle.Seq(
		lock.Acquire(),
		bundle.Defer(
			bundle.Seq(
				files.Upload(),
				artifacts.UploadAll(),
				libraries.AttachToResources(),
				terraform.Interpolate(),
				terraform.Write(),
				terraform.StatePull(),
				terraform.Apply(),
				terraform.StatePush(),
			),
			lock.Release(lock.GoalDeploy),
		),
	)

	return newPhase(
		"deploy",
		[]bundle.Mutator{deployMutator},
	)
}
