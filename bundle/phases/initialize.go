package phases

import (
	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	pythonmutator "github.com/databricks/cli/bundle/config/mutator/python"
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
			mutator.RewriteSyncPaths(),
			mutator.MergeJobClusters(),
			mutator.MergeJobTasks(),
			mutator.MergePipelineClusters(),
			mutator.InitializeWorkspaceClient(),
			mutator.PopulateCurrentUser(),
			mutator.DefineDefaultWorkspaceRoot(),
			mutator.ExpandWorkspaceRoot(),
			mutator.DefineDefaultWorkspacePaths(),
			mutator.SetVariables(),
			// Intentionally placed before ResolveVariableReferencesInLookup, ResolveResourceReferences,
			// ResolveVariableReferencesInComplexVariables and ResolveVariableReferences.
			// See what is expected in PythonMutatorPhaseInit doc
			pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseInit),
			mutator.ResolveVariableReferencesInLookup(),
			mutator.ResolveVariableReferencesInComplexVariables(),
			mutator.ResolveResourceReferences(),
			mutator.ResolveVariableReferences(
				"bundle",
				"workspace",
				"variables",
			),
			mutator.SetRunAs(),
			mutator.OverrideCompute(),
			mutator.ProcessTargetMode(),
			mutator.DefaultQueueing(),
			mutator.ExpandPipelineGlobPaths(),
			mutator.TranslatePaths(),
			python.WrapperWarning(),
			permissions.ApplyBundlePermissions(),
			permissions.FilterCurrentUser(),
			metadata.AnnotateJobs(),
			metadata.AnnotatePipelines(),
			terraform.Initialize(),
			scripts.Execute(config.ScriptPostInit),
		},
	)
}
