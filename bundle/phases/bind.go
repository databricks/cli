package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/lock"
	"github.com/databricks/cli/bundle/deploy/terraform"
)

func Bind(opts *terraform.BindOptions) bundle.Mutator {
	return newPhase(
		"bind",
		[]bundle.Mutator{
			lock.Acquire(),
			bundle.Defer(
				bundle.Seq(
					terraform.StatePull(),
					terraform.Interpolate(),
					terraform.Write(),
					terraform.Import(opts),
					terraform.StatePush(),
				),
				lock.Release(lock.GoalBind),
			),
		},
	)
}

func Unbind(resourceType, resourceKey string) bundle.Mutator {
	return newPhase(
		"unbind",
		[]bundle.Mutator{
			lock.Acquire(),
			bundle.Defer(
				bundle.Seq(
					terraform.StatePull(),
					terraform.Interpolate(),
					terraform.Write(),
					terraform.Unbind(resourceType, resourceKey),
					terraform.StatePush(),
				),
				lock.Release(lock.GoalUnbind),
			),
		},
	)
}
