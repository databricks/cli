package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/deploy/terraform"
)

type BindOptions struct {
	ResourceType string
	ResourceKey  string
	ResourceId   string
}

func Bind(opts *BindOptions) bundle.Mutator {
	return newPhase(
		"bind",
		[]bundle.Mutator{
			terraform.Interpolate(),
			terraform.Write(),
			terraform.StatePull(),
			terraform.Import(opts.ResourceType, opts.ResourceKey, opts.ResourceId),
		},
	)
}
