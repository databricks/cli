package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
)

// The deploy phase deploys artifacts and resources.
func Deploy() bundle.Mutator {
	deployPhase := bundle.Defer([]bundle.Mutator{
		lock.Acquire(),
		files.Upload(),
		artifacts.UploadAll(),
		terraform.Interpolate(),
		terraform.Write(),
		terraform.StatePull(),
		terraform.Apply(),
		terraform.StatePush(),
	}, []bundle.Mutator{
		lock.Release(),
	})

	return newPhase(
		"deploy",
		deployPhase,
	)
}
