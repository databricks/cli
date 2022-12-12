package phases

import (
	"github.com/databricks/bricks/bundle"
	"github.com/databricks/bricks/bundle/artifacts"
	"github.com/databricks/bricks/bundle/config/interpolation"
)

// The build phase builds artifacts.
func Build() bundle.Mutator {
	return newPhase(
		"build",
		[]bundle.Mutator{
			artifacts.BuildAll(),
			interpolation.Interpolate(
				interpolation.IncludeLookupsInPath("artifacts"),
			),
		},
	)
}
