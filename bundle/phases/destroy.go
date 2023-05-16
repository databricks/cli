package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
)

// The destroy phase deletes artifacts and resources.
func Destroy() bundle.Mutator {
	return newPhase(
		"destroy",
		[]bundle.Mutator{
			lock.Acquire(),
			terraform.StatePull(),
			terraform.Plan(terraform.PlanGoal("destroy")),
			terraform.Destroy(),
			terraform.StatePush(),
			lock.Release(),
			files.Delete(),
		},
	)
}
