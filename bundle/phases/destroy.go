package phases

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/deploy/files"
	"github.com/databricks/bricks/bundle/deploy/lock"
	"github.com/databricks/bricks/bundle/deploy/terraform"
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
