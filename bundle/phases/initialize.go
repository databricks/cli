package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/interpolation"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/bundle/deploy/terraform"
)

// The initialize phase fills in defaults and connects to the workspace.
// Interpolation of fields referring to the "bundle" and "workspace" keys
// happens upon completion of this phase.
func Initialize() bundle.Mutator {
	return newPhase(
		"initialize",
		[]bundle.Mutator{
			mutator.PopulateCurrentUser(),
			mutator.DefineDefaultWorkspaceRoot(),
			mutator.ExpandWorkspaceRoot(),
			mutator.DefineDefaultWorkspacePaths(),
			mutator.SetVariables(),
			interpolation.Interpolate(
				interpolation.IncludeLookupsInPath("bundle"),
				interpolation.IncludeLookupsInPath("workspace"),
				interpolation.IncludeLookupsInPath(variable.VariableReferencePrefix),
			),
			mutator.OverrideCompute(),
			mutator.ProcessTargetMode(),
			mutator.TranslatePaths(),
			terraform.Initialize(),
		},
	)
}
