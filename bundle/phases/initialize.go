package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/interpolation"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/bundle/deploy/metadata"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/bundle/python"
	"github.com/databricks/cli/bundle/scripts"
)

// The initialize phase fills in defaults and connects to the workspace.
// Interpolation of fields referring to the "bundle" and "workspace" keys
// happens upon completion of this phase.
func Initialize() bundle.Mutator {
	return newPhase(
		"initialize",
		[]bundle.Mutator{
			mutator.InitializeWorkspaceClient(),
			mutator.PopulateCurrentUser(),
			mutator.DefineDefaultWorkspaceRoot(),
			mutator.ExpandWorkspaceRoot(),
			mutator.DefineDefaultWorkspacePaths(),
			mutator.SetVariables(),
			mutator.ResolveResourceReferences(),
			interpolation.Interpolate(
				interpolation.IncludeLookupsInPath("bundle"),
				interpolation.IncludeLookupsInPath("workspace"),
				interpolation.IncludeLookupsInPath(variable.VariableReferencePrefix),
			),
			mutator.SetRunAs(),
			mutator.OverrideCompute(),
			mutator.ProcessTargetMode(),
			mutator.ExpandPipelineGlobPaths(),
			mutator.TranslatePaths(),
			python.WrapperWarning(),
			permissions.ApplyBundlePermissions(),
			metadata.AnnotateJobs(),
			terraform.Initialize(),
			scripts.Execute(config.ScriptPostInit),
		},
	)
}
