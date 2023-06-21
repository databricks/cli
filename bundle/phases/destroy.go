package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/files"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
)

// The destroy phase deletes artifacts and resources.
func Destroy() bundle.Mutator {

	destroyMutator := bundle.Seq(
		lock.Acquire(),
		bundle.Defer(
			bundle.Seq(
				terraform.StatePull(),
				terraform.Plan(terraform.PlanGoal("destroy")),
				terraform.Destroy(),
				terraform.StatePush(),
				files.Delete(),
			),
			lock.Release(lock.GoalDestroy),
		),
	)

	return newPhase(
		"destroy",
		[]bundle.Mutator{destroyMutator},
	)
}
