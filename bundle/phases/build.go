package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/artifacts/whl"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/scripts"
)

// The build phase builds artifacts.
func Build() bundle.Mutator {
	return newPhase(
		"build",
		[]bundle.Mutator{
			scripts.Execute(config.ScriptPreBuild),
			whl.DetectPackage(),
			artifacts.InferMissingProperties(),
			artifacts.PrepareAll(),
			artifacts.BuildAll(),
			scripts.Execute(config.ScriptPostBuild),
			mutator.ResolveVariableReferences(
				"artifacts",
			),
		},
	)
}
