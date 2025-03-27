package phases

import (
	"context"

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

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/log"
)

// The initialize phase fills in defaults and connects to the workspace.
// Interpolation of fields referring to the "bundle" and "workspace" keys
// happens upon completion of this phase.
func Initialize(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	log.Info(ctx, "Phase: initialize")

	nonIdempotentMutators0 := []bundle.Mutator{
		// Update all path fields in the sync block to be relative to the bundle root path.
		mutator.RewriteSyncPaths(),
	}

	// Mutator is idempotent, if it's idempotent on all possible inputs. For a given input:
	//
	// - if mutator returns an error, it's idempotent because errors interrupt execution
	// - if mutator returns a warning, it should return 'diag = nil' for its output
	// - if mutator is called for its output, it shouldn't change it

	// idempotentMutators are
	idempotentMutators := []bundle.Mutator{
		validate.AllResourcesHaveValues(),

		// Configure the default sync path to equal the bundle root if not explicitly configured.
		// By default, this means all files in the bundle root directory are synchronized.
		mutator.SyncDefaultPath(),

		// Figure out if the sync root path is identical or an ancestor of the bundle root path.
		// If it is an ancestor, this updates all paths to be relative to the sync root path.
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

		permissions.ApplyBundlePermissions(),
		permissions.FilterCurrentUser(),
	}

	initializeBundle := func(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
		return bundle.ApplySeq(ctx, b, idempotentMutators...)
	}

	nonIdempotentMutators := []bundle.Mutator{
		permissions.ValidateSharedRootPermissions(),

		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseInit, initializeBundle),
		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseLoadResources, initializeBundle),
		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseApplyMutators, initializeBundle),

		// Configure use of WSFS for reads if the CLI is running on Databricks.
		mutator.ConfigureWSFS(),

		mutator.TranslatePaths(),
		trampoline.WrapperWarning(),

		// artifacts.Prepare should be done after ConfigureWSFS and TranslatePaths
		artifacts.Prepare(),

		apps.Validate(),

		// FIXME can't be part of idempotentMutators "annotate" mutator set deployment field that
		// is not part of the bundle schema, and can't be input into Python code
		metadata.AnnotateJobs(),
		metadata.AnnotatePipelines(),

		terraform.Initialize(),
		scripts.Execute(config.ScriptPostInit),
	}

	mutators := append(nonIdempotentMutators0, append(idempotentMutators, nonIdempotentMutators...)...)

	return bundle.ApplySeq(ctx, b, mutators...)
}
