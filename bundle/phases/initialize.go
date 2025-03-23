package phases

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/apps"
	"github.com/databricks/cli/bundle/artifacts"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	pythonmutator "github.com/databricks/cli/bundle/config/mutator/python"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/bundle/deploy/metadata"
	"github.com/databricks/cli/bundle/deploy/terraform"
	"github.com/databricks/cli/bundle/permissions"
	"github.com/databricks/cli/bundle/scripts"
	"github.com/databricks/cli/bundle/trampoline"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

// The initialize phase fills in defaults and connects to the workspace.
// Interpolation of fields referring to the "bundle" and "workspace" keys
// happens upon completion of this phase.
func Initialize(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	log.Info(ctx, "Phase: initialize")

	return bundle.ApplySeq(ctx, b,
		// Reads (dynamic): resource.*.*
		// Checks that none of resources.<type>.<key> is nil. Raises error otherwise.
		validate.AllResourcesHaveValues(),

		// Reads (dynamic): workspace.{host,profile,...} (ensure that there are no variable references)
		validate.NoInterpolationInAuthConfig(),

		// Updates (dynamic): sync.{path,include,exclude}  (makes them relative to bundle root rather than to definition file)
		mutator.RewriteSyncPaths(),

		// Reads (dynamic): sync.paths (checks that it is absent)
		// Updates (static): b.Config.Sync.Path (default set to ["."])
		// Configure the default sync path to equal the bundle root if not explicitly configured.
		// By default, this means all files in the bundle root directory are synchronized.
		mutator.SyncDefaultPath(),

		// Figure out if the sync root path is identical or an ancestor of the bundle root path.
		// If it is an ancestor, this updates all paths to be relative to the sync root path.
		// Reads (static): b.Config.Sync.Paths (calculates longest common parent together with bundle root).
		// Updates (static) b.{SyncRoot,SyncRootPath}  (set to calculate sync root, which is either bundle root or some parent of bundle root)
		// Updates (static) b.Config.{Sync,Include,Exclude} they set to be relative to SyncRootPath instead of bundle root
		mutator.SyncInferRoot(),

		mutator.PopulateCurrentUser(),
		mutator.LoadGitDetails(),

		// This mutator needs to be run before variable interpolation and defining default workspace paths
		// because it affects how workspace variables are resolved.
		mutator.ApplySourceLinkedDeploymentPreset(),

		mutator.DefineDefaultWorkspaceRoot(),
		mutator.ExpandWorkspaceRoot(),
		mutator.DefineDefaultWorkspacePaths(),
		mutator.PrependWorkspacePrefix(),

		// This mutator needs to be run before variable interpolation because it
		// searches for strings with variable references in them.
		mutator.RewriteWorkspacePrefix(),

		mutator.SetVariables(),

		// Intentionally placed before ResolveVariableReferencesInLookup, ResolveResourceReferences,
		// ResolveVariableReferencesInComplexVariables and ResolveVariableReferences.
		// See what is expected in PythonMutatorPhaseInit doc
		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseInit),
		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseLoadResources),
		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseApplyMutators),
		mutator.ResolveVariableReferencesInLookup(),
		mutator.ResolveResourceReferences(),
		mutator.ResolveVariableReferences(
			"bundle",
			"workspace",
			"variables",
		),

		mutator.MergeJobClusters(),
		mutator.MergeJobParameters(),
		mutator.MergeJobTasks(),
		mutator.MergePipelineClusters(),
		mutator.MergeApps(),

		mutator.CaptureSchemaDependency(),

		// Provide permission config errors & warnings after initializing all variables
		permissions.PermissionDiagnostics(),
		mutator.SetRunAs(),
		mutator.OverrideCompute(),
		mutator.ConfigureDashboardDefaults(),
		mutator.ConfigureVolumeDefaults(),
		mutator.ProcessTargetMode(),
		mutator.ApplyPresets(),
		mutator.DefaultQueueing(),
		mutator.ExpandPipelineGlobPaths(),

		// Configure use of WSFS for reads if the CLI is running on Databricks.
		mutator.ConfigureWSFS(),

		mutator.TranslatePaths(),
		trampoline.WrapperWarning(),

		artifacts.Prepare(),

		apps.Validate(),

		permissions.ValidateSharedRootPermissions(),
		permissions.ApplyBundlePermissions(),
		permissions.FilterCurrentUser(),

		metadata.AnnotateJobs(),
		metadata.AnnotatePipelines(),
		terraform.Initialize(),
		scripts.Execute(config.ScriptPostInit),
	)
}
