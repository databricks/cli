package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle/config/mutator/resourcemutator"

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
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/logdiag"
)

// The initialize phase fills in defaults and connects to the workspace.
// Interpolation of fields referring to the "bundle" and "workspace" keys
// happens upon completion of this phase.
func Initialize(ctx context.Context, b *bundle.Bundle) {
	var err error

	log.Info(ctx, "Phase: initialize")

	b.DirectDeployment, err = IsDirectDeployment(ctx)
	if err != nil {
		logdiag.LogError(ctx, err)
		return
	}

	bundle.ApplySeqContext(ctx, b,
		// Reads (dynamic): resource.*.*
		// Checks that none of resources.<type>.<key> is nil. Raises error otherwise.
		validate.AllResourcesHaveValues(),
		validate.NoInterpolationInAuthConfig(),
		validate.Scripts(),

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

		// Reads (dynamic): variables.* (checks if there's a value assigned to variable already or if it has lookup reference)
		// Updates (dynamic): variables.*.value (sets values from environment variables, variable files, or defaults)
		// Resolves and sets values for bundle variables in the following order: from environment variables, from variable files and then defaults
		mutator.SetVariables(),

		// Reads (dynamic): variables.*.lookup (checks for variable references in lookup fields)
		// Updates (dynamic): variables.*.lookup (resolves variable references in lookup fields)
		// Prevents circular references between lookup variables
		mutator.ResolveVariableReferencesInLookup(),

		// Reads (dynamic): variables.*.lookup (checks for variables with lookup fields)
		// Updates (dynamic): variables.*.value (sets values based on resolved lookups)
		mutator.ResolveResourceReferences(),

		// Reads (dynamic): * (strings) (searches for variable references in string values)
		// Updates (dynamic): * (except 'resources') (strings) (resolves variable references to their actual values)
		// Resolves variable references in configuration (except resources) using bundle, workspace,
		// and variables prefixes
		mutator.ResolveVariableReferencesWithoutResources(
			"bundle",
			"workspace",
			"variables",
		),

		// Check for invalid use of /Volumes in workspace paths
		validate.ValidateVolumePath(),

		// ApplyTargetMode sets default values for 'presets' section.
		//
		// It must run before ProcessStaticResources and PythonMutator using
		// ApplyPresets through ResourceProcessor.
		resourcemutator.ApplyTargetMode(),

		// Reads (typed): b.SyncRoot (checks if bundle root is in /Workspace/)
		// Updates (typed): b.SyncRoot (replaces with extension-aware path when running on Databricks Runtime)
		// Configure use of WSFS for reads if the CLI is running on Databricks.
		mutator.ConfigureWSFS(),

		// Static resources (e.g. YAML) are already loaded, we initialize and normalize them before Python
		resourcemutator.ProcessStaticResources(),

		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseLoadResources),
		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseApplyMutators),
		// This is the last mutator that can change bundle resources.
		//
		// After PythonMutator, mutators must not change bundle resources, or such changes are not
		// going to be visible in Python code.

		// Validate all required fields are set. This is run after variable interpolation and PyDABs mutators
		// since they can also set and modify resources.
		validate.Required(),

		// Reads (typed): b.Config.Permissions (checks if current user or their groups have CAN_MANAGE permissions)
		// Reads (typed): b.Config.Workspace.CurrentUser (gets current user information)
		// Provides diagnostic recommendations if the current deployment identity isn't explicitly granted CAN_MANAGE permissions
		permissions.PermissionDiagnostics(),

		mutator.TranslatePaths(),

		// Reads (typed): b.Config.Experimental.PythonWheelWrapper, b.Config.Presets.SourceLinkedDeployment (checks Python wheel wrapper and deployment mode settings)
		// Reads (dynamic): resources.jobs.*.tasks (checks for tasks with local libraries and incompatible DBR versions)
		// Provides warnings when Python wheel tasks are used with DBR < 13.3 or when wheel wrapper is incompatible with source-linked deployment
		trampoline.WrapperWarning(),

		// Reads (typed): b.Config.Presets.ArtifactsDynamicVersion (checks if artifacts preset is enabled)
		// Updates (typed): b.Config.Artifacts[].DynamicVersion (sets to true when preset is enabled)
		// Applies the artifacts_dynamic_version preset to enable dynamic versioning on all artifacts
		mutator.ApplyArtifactsDynamicVersion(),

		// Reads (typed): b.Config.Artifacts, b.BundleRootPath (checks artifact configurations and bundle path)
		// Updates (typed): b.Config.Artifacts (auto-creates Python wheel artifact if none defined but setup.py exists)
		// Updates (dynamic): artifacts.*.{path,build_command,files.*.source} (sets default paths, build commands, and makes relative paths absolute)
		// Prepares artifacts by cleaning build directories, expanding file globs, and configuring Python wheel builds
		artifacts.Prepare(),

		// Reads (dynamic): resources.apps.*.source_code_path, resources.apps.*.config (checks for duplicate source code paths and deprecated config sections)
		// Validates app configurations by detecting duplicate source code paths and warning about deprecated config sections
		apps.Validate(),

		resourcemutator.ValidateTargetMode(),
		// Reads (typed): b.Config.Workspace.RootPath (checks if path is in shared workspace)
		// Reads (typed): b.Config.Permissions (checks if users group has CAN_MANAGE permission)
		// Validates that when using a shared workspace path, appropriate permissions are configured
		permissions.ValidateSharedRootPermissions(),

		// Annotate resources with "deployment" metadata.
		//
		// We don't include this step into initializeResources because these mutators set fields that are
		// not part of the bundle schema, and they are not meant to be specified or overridden by the user.
		// For example, PythonMutator reports that "deployment" is not a valid field.
		//
		// Reads (typed): b.Config.Resources.Jobs (checks job configurations)
		// Updates (typed): b.Config.Resources.Jobs[].JobSettings.{Deployment,EditMode,Format} (sets deployment metadata, locks UI editing, and sets format to multi-task)
		// Annotates jobs with bundle deployment metadata and configures job settings for bundle deployments
		metadata.AnnotateJobs(),

		// Reads (typed): b.Config.Resources.Pipelines (checks pipeline configurations)
		// Updates (typed): b.Config.Resources.Pipelines[].CreatePipeline.Deployment (sets deployment metadata for bundle deployments)
		// Annotates pipelines with bundle deployment metadata
		metadata.AnnotatePipelines(),
	)

	if logdiag.HasError(ctx) {
		return
	}

	if !b.DirectDeployment {
		// Reads (typed): b.Config.Bundle.Terraform (checks terraform configuration)
		// Updates (typed): b.Config.Bundle.Terraform (sets default values if not already set)
		// Updates (typed): b.Terraform (initializes Terraform executor with proper environment variables and paths)
		// Initializes Terraform with the correct binary, working directory, and environment variables for authentication

		bundle.ApplyContext(ctx, b, terraform.Initialize())
	}

	if logdiag.HasError(ctx) {
		return
	}

	// Reads (typed): b.Config.Experimental.Scripts["post_init"] (checks if script is defined)
	// Executes the post_init script hook defined in the bundle configuration
	bundle.ApplyContext(ctx, b, scripts.Execute(config.ScriptPostInit))
}

func IsDirectDeployment(ctx context.Context) (bool, error) {
	deployment := env.Get(ctx, "DATABRICKS_CLI_DEPLOYMENT")
	// We use "direct-exp" while direct backend is not suitable for end users.
	// Once we consider it usable we'll change the value to "direct".
	// This is to prevent accidentally running direct backend with older CLI versions where it was still considered experimental.
	if deployment == "direct-exp" {
		return true, nil
	} else if deployment == "terraform" || deployment == "" {
		return false, nil
	} else {
		return false, fmt.Errorf("Unexpected setting for DATABRICKS_CLI_DEPLOYMENT=%#v (expected 'terraform' or 'direct-exp' or absent/empty which means 'terraform')", deployment)
	}
}
