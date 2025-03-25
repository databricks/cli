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

		// Updates (dynamic): sync.{paths,include,exclude} (makes them relative to bundle root rather than to definition file)
		// Rewrites sync paths to be relative to the bundle root instead of the file they were defined in.
		mutator.RewriteSyncPaths(),

		// Reads (dynamic): sync.paths (checks that it is absent)
		// Updates (static): b.Config.Sync.Paths (default set to ["."])
		// Configure the default sync path to equal the bundle root if not explicitly configured.
		// By default, this means all files in the bundle root directory are synchronized.
		mutator.SyncDefaultPath(),

		// Figure out if the sync root path is identical or an ancestor of the bundle root path.
		// If it is an ancestor, this updates all paths to be relative to the sync root path.
		// Reads (static): b.Config.Sync.Paths (calculates longest common parent together with bundle root).
		// Updates (static) b.{SyncRoot,SyncRootPath}  (set to calculate sync root, which is either bundle root or some parent of bundle root)
		// Updates (static) b.Config.{Sync,Include,Exclude} they set to be relative to SyncRootPath instead of bundle root
		mutator.SyncInferRoot(),

		// Reads (static): b.Config.Workspace.CurrentUser (checks if it's already set)
		// Updates (static): b.Config.Workspace.CurrentUser (sets user information from API)
		// Updates (static): b.Tagging (configures tagging object based on workspace client)
		mutator.PopulateCurrentUser(),
		// Updates (static): b.WorktreeRoot (sets to SyncRoot if no git repo found, otherwise to git worktree root)
		// Updates (static): b.Config.Bundle.Git.{ActualBranch,Branch,Commit,OriginURL,BundleRootPath} (loads git repository details)
		// Loads git repository information and updates bundle configuration with git details
		mutator.LoadGitDetails(),

		// This mutator needs to be run before variable interpolation and defining default workspace paths
		// because it affects how workspace variables are resolved.
		mutator.ApplySourceLinkedDeploymentPreset(),

		// Reads (static): b.Config.Workspace.RootPath (checks if it's already set)
		// Reads (static): b.Config.Bundle.Name, b.Config.Bundle.Target (used to construct default path)
		// Updates (static): b.Config.Workspace.RootPath (sets to ~/.bundle/{name}/{target} if not set)
		mutator.DefineDefaultWorkspaceRoot(),

		// Reads (static): b.Config.Workspace.RootPath (checks if it's already set)
		// Reads (static): b.Config.Workspace.CurrentUser (used to expand ~ in path)
		// Updates (static): b.Config.Workspace.RootPath (expands ~ to user's home directory if present)
		mutator.ExpandWorkspaceRoot(),

		// Reads (static): b.Config.Workspace.RootPath (used to construct default paths)
		// Updates (static): b.Config.Workspace.FilePath, b.Config.Workspace.ResourcePath, b.Config.Workspace.ArtifactPath, b.Config.Workspace.StatePath (sets default paths if not already set)
		mutator.DefineDefaultWorkspacePaths(),
		mutator.PrependWorkspacePrefix(),

		// This mutator needs to be run before variable interpolation because it
		// searches for strings with variable references in them.
		mutator.RewriteWorkspacePrefix(),

		// Reads (dynamic): variables.* (checks for existing values, defaults, and lookup references)
		// Updates (dynamic): variables.*.value (sets values from environment variables, variable files, or defaults)
		// Resolves and sets values for bundle variables from environment variables, variable files, or defaults
		mutator.SetVariables(),

		// Intentionally placed before ResolveVariableReferencesInLookup, ResolveResourceReferences,
		// ResolveVariableReferencesInComplexVariables and ResolveVariableReferences.
		// See what is expected in PythonMutatorPhaseInit doc
		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseInit),
		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseLoadResources),
		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseApplyMutators),

		// Reads (dynamic): variables.*.lookup (checks for variable references in lookup fields)
		// Updates (dynamic): variables.*.lookup (resolves variable references in lookup fields)
		// Prevents circular references between lookup variables
		mutator.ResolveVariableReferencesInLookup(),
		// Reads (dynamic): variables.*.lookup (checks for variables with lookup fields)
		// Updates (dynamic): variables.*.value (sets values based on resolved lookups)
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
