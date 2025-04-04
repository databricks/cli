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

		// Updates (dynamic): sync.{paths,include,exclude} (makes them relative to bundle root rather than to definition file)
		// Rewrites sync paths to be relative to the bundle root instead of the file they were defined in.
		mutator.RewriteSyncPaths(),

		// Reads (dynamic): sync.paths (checks that it is absent)
		// Updates (typed): b.Config.Sync.Paths (default set to ["."])
		// Configure the default sync path to equal the bundle root if not explicitly configured.
		// By default, this means all files in the bundle root directory are synchronized.
		mutator.SyncDefaultPath(),

		// Figure out if the sync root path is identical or an ancestor of the bundle root path.
		// If it is an ancestor, this updates all paths to be relative to the sync root path.
		// Reads (typed): b.Config.Sync.Paths (calculates longest common parent together with bundle root).
		// Updates (typed) b.{SyncRoot,SyncRootPath}  (set to calculated sync root, which is either bundle root or some parent of bundle root)
		// Updates (typed) b.Config.{Sync,Include,Exclude} they set to be relative to SyncRootPath instead of bundle root
		mutator.SyncInferRoot(),

		// Reads (typed): b.Config.Workspace.CurrentUser (checks if it's already set)
		// Updates (typed): b.Config.Workspace.CurrentUser (sets user information from API)
		// Updates (typed): b.Tagging (configures tagging object based on current cloud)
		mutator.PopulateCurrentUser(),

		// Updates (typed): b.WorktreeRoot (sets to SyncRoot if no git repo found, otherwise to git worktree root)
		// Updates (typed): b.Config.Bundle.Git.{ActualBranch,Branch,Commit,OriginURL,BundleRootPath} (sets values based on git repository details)
		// Loads git repository information and updates bundle configuration with git details
		mutator.LoadGitDetails(),

		// This mutator needs to be run before variable interpolation and defining default workspace paths
		// because it affects how workspace variables are resolved.
		mutator.ApplySourceLinkedDeploymentPreset(),

		// Reads (typed): b.Config.Workspace.RootPath (checks if it's already set)
		// Reads (typed): b.Config.Bundle.Name, b.Config.Bundle.Target (used to construct default path)
		// Updates (typed): b.Config.Workspace.RootPath (sets to ~/.bundle/{name}/{target} if not set)
		mutator.DefineDefaultWorkspaceRoot(),

		// Reads (typed): b.Config.Workspace.RootPath (checks if it's already set)
		// Reads (typed): b.Config.Workspace.CurrentUser (used to expand ~ in path)
		// Updates (typed): b.Config.Workspace.RootPath (expands ~ to user's home directory if present)
		mutator.ExpandWorkspaceRoot(),

		// Reads (typed): b.Config.Workspace.RootPath (used to construct default paths)
		// Updates (typed): b.Config.Workspace.{FilePath,ResourcePath,ArtifactPath,StatePath} (sets default paths if not already set)
		mutator.DefineDefaultWorkspacePaths(),

		// Reads (dynamic): workspace.{root_path,file_path,artifact_path,state_path,resource_path} (reads paths to prepend prefix)
		// Updates (dynamic): workspace.{root_path,file_path,artifact_path,state_path,resource_path} (prepends "/Workspace" to paths that don't already have specific prefixes)
		mutator.PrependWorkspacePrefix(),

		// Reads (dynamic): * (strings) (searches for strings with workspace path variables prefixed with "/Workspace")
		// Updates (dynamic): * (strings) (removes "/Workspace" prefix from workspace path variables)
		// Finds and removes "/Workspace" prefix from all strings in bundle configuration.
		// This mutator needs to be run before variable interpolation because it
		// searches for strings with variable references in them.
		mutator.RewriteWorkspacePrefix(),

		// Reads (dynamic): variables.* (checks for existing values, defaults, and lookup references)
		// Updates (dynamic): variables.*.value (sets values from environment variables, variable files, or defaults)
		// Resolves and sets values for bundle variables in the following order: from environment variables, from variable files and then defaults
		mutator.SetVariables(),

		// Intentionally placed before ResolveVariableReferencesInLookup, ResolveResourceReferences,
		// ResolveVariableReferencesInComplexVariables and ResolveVariableReferences.
		// See what is expected in PythonMutatorPhaseInit doc
		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseLoad),
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

		// Reads (dynamic): * (strings) (searches for variable references in string values)
		// Updates (dynamic): * (strings) (resolves variable references to their actual values)
		// Resolves variable references in configuration using bundle, workspace, and variables prefixes
		mutator.ResolveVariableReferencesWithoutResources(
			"bundle",
			"workspace",
			"variables",
		),
		mutator.ResolveVariableReferencesOnlyResources(
			"bundle",
			"workspace",
			"variables",
		),
		mutator.NormalizePaths(),

		// Reads (dynamic): resources.pipelines.*.libraries (checks for notebook.path and file.path fields)
		// Updates (dynamic): resources.pipelines.*.libraries (expands glob patterns in path fields to multiple library entries)
		// Expands glob patterns in pipeline library paths to include all matching files
		mutator.ExpandPipelineGlobPaths(),

		// Reads (dynamic): resources.jobs.*.job_clusters (reads job clusters to merge)
		// Updates (dynamic): resources.jobs.*.job_clusters (merges job clusters with the same job_cluster_key)
		mutator.MergeJobClusters(),

		// Reads (dynamic): resources.jobs.*.parameters (reads job parameters to merge)
		// Updates (dynamic): resources.jobs.*.parameters (merges job parameters with the same name)
		mutator.MergeJobParameters(),

		// Reads (dynamic): resources.jobs.*.tasks (reads job tasks to merge)
		// Updates (dynamic): resources.jobs.*.tasks (merges job tasks with the same task_key)
		mutator.MergeJobTasks(),

		// Reads (dynamic): resources.pipelines.*.clusters (reads pipeline clusters to merge)
		// Updates (dynamic): resources.pipelines.*.clusters (merges pipeline clusters with the same label)
		mutator.MergePipelineClusters(),

		// Reads (dynamic): resources.apps.*.resources (reads app resources to merge)
		// Updates (dynamic): resources.apps.*.resources (merges app resources with the same name)
		mutator.MergeApps(),

		// Reads (typed): resources.pipelines.*.{catalog,schema,target}, resources.volumes.*.{catalog_name,schema_name} (checks for schema references)
		// Updates (typed): resources.pipelines.*.{schema,target}, resources.volumes.*.schema_name (converts implicit schema references to explicit ${resources.schemas.<schema_key>.name} syntax)
		// Translates implicit schema references in DLT pipelines or UC Volumes to explicit syntax to capture dependencies
		mutator.CaptureSchemaDependency(),

		// Reads (typed): b.Config.Permissions (checks if current user or their groups have CAN_MANAGE permissions)
		// Reads (typed): b.Config.Workspace.CurrentUser (gets current user information)
		// Provides diagnostic recommendations if the current deployment identity isn't explicitly granted CAN_MANAGE permissions
		permissions.PermissionDiagnostics(),

		// Reads (typed): b.Config.RunAs, b.Config.Workspace.CurrentUser (validates run_as configuration)
		// Reads (dynamic): run_as (checks if run_as is specified)
		// Updates (typed): b.Config.Resources.Jobs[].RunAs (sets job run_as fields to bundle run_as; only if Experimental.UseLegacyRunAs is set)
		// Updates (typed): range b.Config.Resources.Pipelines[].Permissions (set permission based on bundle run_as; only if Experimental.UseLegacyRunAs is set)
		mutator.SetRunAs(),

		// Reads (typed): b.Config.Bundle.{Mode,ClusterId} (checks mode and cluster ID settings)
		// Reads (env): DATABRICKS_CLUSTER_ID (environment variable for backward compatibility)
		// Reads (typed): b.Config.Resources.Jobs.*.Tasks.*.ForEachTask
		// Updates (typed): b.Config.Bundle.ClusterId (sets from environment if in development mode)
		// Updates (typed): b.Config.Resources.Jobs.*.Tasks.*.{NewCluster,ExistingClusterId,JobClusterKey,EnvironmentKey} (replaces compute settings with specified cluster ID)
		// OR corresponding fields on ForEachTask if that is present
		// Overrides job compute settings with a specified cluster ID for development or testing
		mutator.OverrideCompute(),

		// Reads (dynamic): resources.dashboards.* (checks for existing parent_path and embed_credentials)
		// Updates (dynamic): resources.dashboards.*.parent_path (sets to workspace.resource_path if not set)
		// Updates (dynamic): resources.dashboards.*.embed_credentials (sets to false if not set)
		mutator.ConfigureDashboardDefaults(),

		// Reads (dynamic): resources.volumes.* (checks for existing volume_type)
		// Updates (dynamic): resources.volumes.*.volume_type (sets to "MANAGED" if not set)
		mutator.ConfigureVolumeDefaults(),

		// Reads (typed): b.Config.Bundle.Mode, b.Config.Workspace.{RootPath,FilePath,ResourcePath,ArtifactPath,StatePath}, b.Config.Workspace.CurrentUser (validates paths and user info)
		// Updates (typed): b.Config.Bundle.Deployment.Lock.Enabled, b.Config.Presets.{NamePrefix,Tags,JobsMaxConcurrentRuns,TriggerPauseStatus,PipelinesDevelopment} (configures development mode settings)
		// Validates and configures bundle settings based on target mode (development or production)
		mutator.ProcessTargetMode(),

		// Reads (typed): b.Config.Presets.{NamePrefix,Tags,JobsMaxConcurrentRuns,TriggerPauseStatus,PipelinesDevelopment} (reads preset configurations)
		// Updates (typed): b.Config.Resources.{Jobs,Pipelines,Models,Experiments,ModelServingEndpoints,RegisteredModels,QualityMonitors,Schemas,Clusters,Dashboards} (applies prefix to names, adds tags, sets pause status and development mode)
		// Applies all presets to resources, including name prefixes, tags, and resource-specific settings like pause status
		mutator.ApplyPresets(),

		// Reads (typed): b.Config.Resources.Jobs (checks job configurations)
		// Updates (typed): b.Config.Resources.Jobs[].Queue (sets Queue.Enabled to true for jobs without queue settings)
		// Enable queueing for jobs by default, following the behavior from API 2.2+.
		mutator.DefaultQueueing(),

		// Reads (typed): b.SyncRoot (checks if bundle root is in /Workspace/)
		// Updates (typed): b.SyncRoot (replaces with extension-aware path when running on Databricks Runtime)
		// Configure use of WSFS for reads if the CLI is running on Databricks.
		mutator.ConfigureWSFS(),

		mutator.TranslatePaths(),

		// Reads (typed): b.Config.Experimental.PythonWheelWrapper, b.Config.Presets.SourceLinkedDeployment (checks Python wheel wrapper and deployment mode settings)
		// Reads (dynamic): resources.jobs.*.tasks (checks for tasks with local libraries and incompatible DBR versions)
		// Provides warnings when Python wheel tasks are used with DBR < 13.3 or when wheel wrapper is incompatible with source-linked deployment
		trampoline.WrapperWarning(),

		// Reads (typed): b.Config.Artifacts, b.BundleRootPath (checks artifact configurations and bundle path)
		// Updates (typed): b.Config.Artifacts (auto-creates Python wheel artifact if none defined but setup.py exists)
		// Updates (dynamic): artifacts.*.{path,build_command,files.*.source} (sets default paths, build commands, and makes relative paths absolute)
		// Prepares artifacts by cleaning build directories, expanding file globs, and configuring Python wheel builds
		artifacts.Prepare(),

		// Reads (dynamic): resources.apps.*.source_code_path, resources.apps.*.config (checks for duplicate source code paths and deprecated config sections)
		// Validates app configurations by detecting duplicate source code paths and warning about deprecated config sections
		apps.Validate(),

		// Reads (typed): b.Config.Workspace.RootPath (checks if path is in shared workspace)
		// Reads (typed): b.Config.Permissions (checks if users group has CAN_MANAGE permission)
		// Validates that when using a shared workspace path, appropriate permissions are configured
		permissions.ValidateSharedRootPermissions(),

		// Reads (typed): b.Config.Permissions (validates permission levels)
		// Reads (dynamic): resources.{jobs,pipelines,experiments,models,model_serving_endpoints,dashboards,apps}.*.permissions (reads existing permissions)
		// Updates (dynamic): resources.{jobs,pipelines,experiments,models,model_serving_endpoints,dashboards,apps}.*.permissions (adds permissions from bundle-level configuration)
		// Applies bundle-level permissions to all supported resources
		permissions.ApplyBundlePermissions(),

		// Reads (typed): b.Config.Workspace.CurrentUser.UserName (gets current user name)
		// Updates (dynamic): resources.*.*.permissions (removes permissions entries where user_name or service_principal_name matches current user)
		// Removes the current user from all resource permissions as the Terraform provider implicitly grants ownership
		permissions.FilterCurrentUser(),

		// Reads (typed): b.Config.Resources.Jobs (checks job configurations)
		// Updates (typed): b.Config.Resources.Jobs[].JobSettings.{Deployment,EditMode,Format} (sets deployment metadata, locks UI editing, and sets format to multi-task)
		// Annotates jobs with bundle deployment metadata and configures job settings for bundle deployments
		metadata.AnnotateJobs(),

		// Reads (typed): b.Config.Resources.Pipelines (checks pipeline configurations)
		// Updates (typed): b.Config.Resources.Pipelines[].CreatePipeline.Deployment (sets deployment metadata for bundle deployments)
		// Annotates pipelines with bundle deployment metadata
		metadata.AnnotatePipelines(),

		// Reads (typed): b.Config.Bundle.Terraform (checks terraform configuration)
		// Updates (typed): b.Config.Bundle.Terraform (sets default values if not already set)
		// Updates (typed): b.Terraform (initializes Terraform executor with proper environment variables and paths)
		// Initializes Terraform with the correct binary, working directory, and environment variables for authentication
		terraform.Initialize(),

		// Reads (typed): b.Config.Experimental.Scripts["post_init"] (checks if script is defined)
		// Executes the post_init script hook defined in the bundle configuration
		scripts.Execute(config.ScriptPostInit),
	)
}
