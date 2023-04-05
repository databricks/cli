package phases

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/deploy/files"
	"github.com/databricks/bricks/bundle/deploy/lock"
	"github.com/databricks/bricks/bundle/deploy/terraform"
)

// The destroy phase deletes artifacts and resources.
// TODO: force lock workaround. Error message on lock acquisition is misleading
func Destroy() bundle.Mutator {
	return newPhase(
		"destroy",
		[]bundle.Mutator{
			lock.Acquire(),
			terraform.StatePull(),
			terraform.Plan(true),
			terraform.Destroy(),
			terraform.StatePush(),
			lock.Release(),
			files.Delete(),
		},
	)
}
