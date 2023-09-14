package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/interpolation"
	"github.com/databricks/cli/bundle/scripts"
)

// The build phase builds artifacts.
func Build() bundle.Mutator {
	return newPhase(
		"build",
		[]bundle.Mutator{
			scripts.Execute(config.ScriptPreBuild),
			artifacts.DetectPackages(),
			artifacts.InferMissingProperties(),
			artifacts.BuildAll(),
			scripts.Execute(config.ScriptPostBuild),
			interpolation.Interpolate(
				interpolation.IncludeLookupsInPath("artifacts"),
			),
		},
	)
}
