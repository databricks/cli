package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/config/interpolation"
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
