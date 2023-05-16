package phases

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/deploy/files"
	"github.com/databricks/bricks/bundle/deploy/lock"
	"github.com/databricks/bricks/bundle/deploy/terraform"
)

// The destroy phase deletes artifacts and resources.
func Destroy() bundle.Mutator {
	destroyPhase := bundle.Defer([]bundle.Mutator{
		lock.Acquire(),
		terraform.StatePull(),
		terraform.Plan(terraform.PlanGoal("destroy")),
		terraform.Destroy(),
		terraform.StatePush(),
		files.Delete(),
	}, []bundle.Mutator{
		lock.Release(),
	})

	return newPhase(
		"destroy",
		destroyPhase,
	)
}
