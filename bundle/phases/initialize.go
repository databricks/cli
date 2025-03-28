package phases

import (
	"context"
	"fmt"

	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/merge"

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

	var diags diag.Diagnostics

	diags = bundle.ApplySeq(ctx, b,
		validate.AllResourcesHaveValues(),

		// Update all path fields in the sync block to be relative to the bundle root path.
		mutator.RewriteSyncPaths(),

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
		mutator.ResolveVariableReferencesWithoutResources(
			"bundle",
			"workspace",
			"variables",
		),

		// Intentionally placed before ResolveVariableReferencesInLookup, ResolveResourceReferences,
		// ResolveVariableReferencesInComplexVariables and ResolveVariableReferences.
		// See what is expected in PythonMutatorPhaseInit doc
		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseInit),
		pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseLoadResources),
	)
	if diags.HasError() {
		return diags
	}

	resourcePaths, err := extractResourcePaths(b.Config.Value())
	if err != nil {
		return diags.Extend(diag.FromErr(err))
	}

	diags = diags.Extend(NormalizeResources(ctx, b, resourcePaths))
	if diags.HasError() {
		return diags
	}

	diags = diags.Extend(InitializeResources(ctx, b, resourcePaths))
	if diags.HasError() {
		return diags
	}

	diags = diags.Extend(bundle.Apply(ctx, b, pythonmutator.PythonMutator(pythonmutator.PythonMutatorPhaseApplyMutators)))
	if diags.HasError() {
		return diags
	}

	// TODO move into PythonMutator and avoid normalizing resources that haven't changed
	resourcePaths2, err := extractResourcePaths(b.Config.Value())
	diags = diags.Extend(NormalizeResources(ctx, b, resourcePaths2))
	if diags.HasError() {
		return diags
	}

	diags = diags.Extend(bundle.ApplySeq(ctx, b,
		// Provide permission config errors & warnings after initializing all variables
		permissions.PermissionDiagnostics(),

		// Configure use of WSFS for reads if the CLI is running on Databricks.
		mutator.ConfigureWSFS(),

		mutator.TranslatePaths(),
		trampoline.WrapperWarning(),

		artifacts.Prepare(),

		apps.Validate(),

		permissions.ValidateSharedRootPermissions(),

		metadata.AnnotateJobs(),
		metadata.AnnotatePipelines(),
		terraform.Initialize(),
		scripts.Execute(config.ScriptPostInit),
	))

	return diags
}

func InitializeResources(ctx context.Context, b *bundle.Bundle, paths []resourcePath) diag.Diagnostics {
	return ApplySeqForResourcePaths(
		ctx,
		b,
		paths,
		mutator.SetRunAs(),
		mutator.OverrideCompute(),
		mutator.ConfigureDashboardDefaults(),
		mutator.ConfigureVolumeDefaults(),
		mutator.ProcessTargetMode(),
		mutator.ApplyPresets(),
		mutator.DefaultQueueing(),
		permissions.ApplyBundlePermissions(),
		permissions.FilterCurrentUser(),
	)
}

func NormalizeResources(ctx context.Context, b *bundle.Bundle, paths []resourcePath) diag.Diagnostics {
	return ApplySeqForResourcePaths(
		ctx,
		b,
		paths,
		mutator.ResolveVariableReferencesOnlyResources(
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
		mutator.ExpandPipelineGlobPaths(),
	)
}

type resourcePath struct {
	resourceType string
	resourceName string
}

func ApplySeqForResourcePaths(ctx context.Context, b *bundle.Bundle, paths []resourcePath, mutators ...bundle.Mutator) diag.Diagnostics {
	initialConfig := b.Config.Value()

	err := b.Config.Mutate(func(value dyn.Value) (dyn.Value, error) {
		return sliceConfigResources(value, paths)
	})
	if err != nil {
		return diag.Errorf("failed to slice config: %s", err)
	}

	resourcesPath := dyn.NewPath(dyn.Key("resources"))

	// FIXME these paths are changed in mutators
	presetsPath := dyn.NewPath(dyn.Key("presets"))
	bundleDeploymentPath := dyn.NewPath(dyn.Key("bundle"), dyn.Key("deployment"))

	diags := bundle.ApplySeq(ctx, b, mutators...)
	if diags.HasError() {
		return diags
	}

	err = b.Config.Mutate(func(value dyn.Value) (dyn.Value, error) {
		return merge.Override(
			initialConfig,
			value,
			merge.OverrideVisitor{
				VisitDelete: func(valuePath dyn.Path, left dyn.Value) error {
					if !valuePath.HasPrefix(resourcesPath) {
						return fmt.Errorf("unexpected delete at %s", valuePath.String())
					}

					if len(valuePath) <= 3 {
						return merge.ErrOverrideUndoDelete
					}

					return nil
				},
				VisitInsert: func(valuePath dyn.Path, right dyn.Value) (dyn.Value, error) {
					// FIXME avoid modifing paths outside of resources
					if !valuePath.HasPrefix(resourcesPath) &&
						!valuePath.HasPrefix(presetsPath) &&
						!valuePath.HasPrefix(bundleDeploymentPath) {
						return dyn.InvalidValue, fmt.Errorf("unexpected insert at %s", valuePath.String())
					}

					return right, nil
				},
				VisitUpdate: func(valuePath dyn.Path, left, right dyn.Value) (dyn.Value, error) {
					if !valuePath.HasPrefix(resourcesPath) &&
						!valuePath.HasPrefix(presetsPath) &&
						!valuePath.HasPrefix(bundleDeploymentPath) {
						return dyn.InvalidValue, fmt.Errorf("unexpected update at %s", valuePath.String())
					}

					return right, nil
				},
			},
		)
	})

	return diags.Extend(diag.FromErr(err))
}

func extractResourcePaths(config dyn.Value) ([]resourcePath, error) {
	resourcesKey := dyn.Key("resources")
	pattern := dyn.NewPattern(resourcesKey, dyn.AnyKey(), dyn.AnyKey())
	resourcePaths := make([]resourcePath, 0)

	_, err := dyn.MapByPattern(config, pattern, func(path dyn.Path, value dyn.Value) (dyn.Value, error) {
		resourcePath, err := extractResourceKey(path)
		if err != nil {
			return dyn.InvalidValue, err
		}

		resourcePaths = append(resourcePaths, resourcePath)
		return value, nil
	})

	return resourcePaths, err
}

func extractResourceKey(path dyn.Path) (resourcePath, error) {
	if len(path) != 3 {
		return resourcePath{}, fmt.Errorf("can't parse resource key")
	}

	if path[0].Key() != "resources" {
		return resourcePath{}, fmt.Errorf("can't parse resource key")
	}

	resourceType := path[1].Key()
	resourceName := path[2].Key()

	if resourceType == "" || resourceName == "" {
		return resourcePath{}, fmt.Errorf("can't parse resource key")
	}

	return resourcePath{
		resourceType: resourceType,
		resourceName: resourceName,
	}, nil
}

// sliceConfigResources remove all resources that are not in the paths
// we keep remaining configuration because it's used in mutators.
func sliceConfigResources(config dyn.Value, paths []resourcePath) (dyn.Value, error) {
	resourcesKey := dyn.Key("resources")
	pattern := dyn.NewPattern(resourcesKey, dyn.AnyKey())
	resourceKeys := make(map[string]map[string]struct{})

	for _, path := range paths {
		if _, ok := resourceKeys[path.resourceType]; !ok {
			resourceKeys[path.resourceType] = make(map[string]struct{})
		}

		resourceKeys[path.resourceType][path.resourceName] = struct{}{}
	}

	return dyn.MapByPattern(config, pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		if v.Kind() != dyn.KindMap {
			return v, nil
		}

		resourceType := p[1].Key()
		newMapping := dyn.NewMapping()
		mapping := v.MustMap()

		// short-circuit: particular type is not needed
		if _, ok := resourceKeys[resourceType]; !ok {
			return dyn.NewValue(newMapping, v.Locations()), nil
		}

		for _, pair := range mapping.Pairs() {
			resourceName := pair.Key.MustString()

			if _, ok := resourceKeys[resourceType][resourceName]; ok {
				err := newMapping.Set(pair.Key, pair.Value)
				if err != nil {
					return dyn.InvalidValue, err
				}
			}
		}

		return dyn.NewValue(newMapping, v.Locations()), nil
	})
}
