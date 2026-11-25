package resourcemutator

import (
	"context"
	"errors"
	"fmt"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/stages"
	"github.com/databricks/cli/bundle/config/validate"

	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/logdiag"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/dyn"
)

// When a new resource is added to configuration, we apply bundle
// settings and defaults to it. Initialization is applied only once.
//
// If bundle is modified outside of 'resources' section, these changes are discarded.
func applyInitializeMutators(ctx context.Context, b *bundle.Bundle) {
	bundle.ApplySeqContext(
		ctx,
		b,
		// Reads (typed): b.Config.RunAs, b.Config.Workspace.CurrentUser (validates run_as configuration)
		// Reads (dynamic): run_as (checks if run_as is specified)
		// Updates (typed): b.Config.Resources.Jobs[].RunAs (sets job run_as fields to bundle run_as; only if Experimental.UseLegacyRunAs is set)
		// Updates (typed): range b.Config.Resources.Pipelines[].Permissions (set permission based on bundle run_as; only if Experimental.UseLegacyRunAs is set)
		SetRunAs(),

		ApplyStage(stages.StageBeforePresets),

		// ApplyPresets should have more priority than defaults below, so it should be run first
		ApplyPresets(),
	)

	if logdiag.HasError(ctx) {
		return
	}

	// The defaults of different resource types set disjoint fields, so their
	// relative order does not matter.
	defaultsContext := resources.DefaultsContext{
		ResourcePath: b.Config.Workspace.ResourcePath,
	}
	for key, d := range config.ResourceTypesImplementing[resources.FieldDefaulter]() {
		for _, fd := range d.FieldDefaults(defaultsContext) {
			bundle.SetDefault(ctx, b, "resources."+key+".*."+fd.Path, fd.Value)
			if logdiag.HasError(ctx) {
				return
			}
		}
	}

	bundle.ApplySeqContext(ctx, b,
		ApplyStage(stages.StageAfterDefaults),

		// Reads (typed): b.Config.Permissions (validates permission levels)
		// Reads (dynamic): resources.{jobs,pipelines,experiments,models,model_serving_endpoints,dashboards,apps,vector_search_endpoints,...}.*.permissions (reads existing permissions)
		// Updates (dynamic): resources.{jobs,pipelines,experiments,models,model_serving_endpoints,dashboards,apps,vector_search_endpoints,...}.*.permissions (adds permissions from bundle-level configuration)
		// Applies bundle-level permissions to all supported resources
		ApplyBundlePermissions(),

		// Reads (typed): b.Config.Workspace.CurrentUser.UserName (gets current user name)
		// Updates (dynamic): resources.*.*.permissions
		FixPermissions(),
	)
}

// Normalization is applied multiple times if resource is modified during initialization
//
// If bundle is modified outside of 'resources' section, these changes are discarded.
func applyNormalizeMutators(ctx context.Context, b *bundle.Bundle) {
	bundle.ApplySeqContext(
		ctx,
		b,

		validate.SingleNodeCluster(),

		// Reads (dynamic): * (strings) (searches for variable references in string values)
		// Updates (dynamic): resources.* (strings) (resolves variable references to their actual values)
		// Resolves variable references in 'resources' using bundle, workspace, and variables prefixes
		mutator.ResolveVariableReferencesOnlyResources(),

		ApplyStage(stages.StageAfterResourceVariables),
		ApplyStage(stages.StageAfterTargetMerge),

		// Reads (dynamic): resources.{catalogs,schemas,external_locations,volumes,registered_models,vector_search_indexes}.*.grants
		// Updates (dynamic): same paths — merges grant entries by principal and deduplicates privileges
		MergeGrants(),

		// Reads (typed): resources.{volumes,registered_models,pipelines,quality_monitors,model_serving_endpoints}.*.{catalog_name,schema_name,...}
		// Updates (typed): same paths — converts implicit schema/catalog references to explicit ${resources.schemas/catalogs.<key>.name} syntax
		// Also updates: resources.schemas.*.catalog_name (catalog dependency for schemas)
		// Translates implicit schema and catalog references across all UC resources to explicit syntax to capture dependencies
		CaptureUCDependencies(),

		ApplyStage(stages.StageAfterUCDependencies),
	)
}

// NormalizeAndInitializeResources initializes and normalizes specified resources,
// and should be used by mutators after they have added resources.
func NormalizeAndInitializeResources(
	ctx context.Context,
	b *bundle.Bundle,
	addedResources ResourceKeySet,
) {
	if addedResources.IsEmpty() {
		return
	}

	var snapshot dyn.Value

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		snapshot = root

		return selectResources(root, addedResources)
	})
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("failed to select resources: %w", err))
		return
	}

	applyNormalizeMutators(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	applyInitializeMutators(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	// after mutators, we merge updated resources back to snapshot to preserve non-selected resources
	err = b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return mergeResources(root, snapshot)
	})
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("failed to merge resources: %w", err))
	}
}

// NormalizeResources normalizes resources specified resources,
// and should be used by mutators after they have updated resources.
func NormalizeResources(
	ctx context.Context,
	b *bundle.Bundle,
	updatedResources ResourceKeySet,
) {
	if updatedResources.IsEmpty() {
		return
	}

	var snapshot dyn.Value

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		snapshot = root

		return selectResources(root, updatedResources)
	})
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("failed to select resources: %w", err))
		return
	}

	applyNormalizeMutators(ctx, b)
	if logdiag.HasError(ctx) {
		return
	}

	// Permissions added to an existing resource by a Python mutator must still get the
	// deploying user as IS_OWNER, otherwise the Permissions API rejects the PUT with
	// "must have exactly one owner" (#5682). FixPermissions is idempotent, so re-running
	// it on resources that already have an owner is a no-op. ApplyBundlePermissions is
	// intentionally not re-run here: it is not idempotent (it appends bundle-level
	// permissions) and already ran for these resources in ProcessStaticResources.
	bundle.ApplyContext(ctx, b, FixPermissions())
	if logdiag.HasError(ctx) {
		return
	}

	// after mutators, we merge updated resources back to snapshot to preserve non-selected resources
	err = b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return mergeResources(root, snapshot)
	})
	if err != nil {
		logdiag.LogError(ctx, fmt.Errorf("failed to merge resources: %w", err))
	}
}

// selectResources returns bundle configuration with resources only present in resourcePaths.
func selectResources(root dyn.Value, resourcePaths ResourceKeySet) (dyn.Value, error) {
	resourcesKeyString := "resources"
	resourcesPath := dyn.NewPath(dyn.Key(resourcesKeyString))

	newRoot := root
	var err error

	// remove resource types that are not in resourcePaths
	newRoot, err = dyn.MapByPath(
		newRoot,
		resourcesPath,
		func(p dyn.Path, resources dyn.Value) (dyn.Value, error) {
			return merge.Select(resources, resourcePaths.Types())
		},
	)
	if err != nil {
		return dyn.InvalidValue, err
	}

	// for each resource type, remove resources by name
	for _, resourceType := range resourcePaths.Types() {
		resourceTypePath := resourcesPath.Append(dyn.Key(resourceType))

		newRoot, err = dyn.MapByPath(
			newRoot,
			resourceTypePath,
			func(p dyn.Path, resource dyn.Value) (dyn.Value, error) {
				return merge.Select(resource, resourcePaths.Names(resourceType))
			},
		)
		if err != nil {
			return dyn.InvalidValue, err
		}
	}

	return newRoot, err
}

// mergeResources returns bundle configuration by merging all resources from src into dst,
// overriding existing resources if they exist.
func mergeResources(src, dst dyn.Value) (dyn.Value, error) {
	resourcesKey := dyn.Key("resources")

	newDst := dst

	// merge 'resources.<type>.<name>'
	_, err := dyn.MapByPattern(
		src,
		dyn.NewPattern(resourcesKey, dyn.AnyKey(), dyn.AnyKey()),
		func(path dyn.Path, v dyn.Value) (dyn.Value, error) {
			// if parent 'resources.<type>' doesn't exist, handle it on the next pass
			updated, _ := dyn.SetByPath(newDst, path, v)
			if !updated.IsValid() {
				return v, nil
			} else {
				newDst = updated
			}

			return v, nil
		},
	)
	if err != nil {
		return newDst, err
	}

	// merge 'resources.<type>'
	_, err = dyn.MapByPattern(
		src,
		dyn.NewPattern(resourcesKey, dyn.AnyKey()),
		func(path dyn.Path, v dyn.Value) (dyn.Value, error) {
			// if already exists, we already handled it in the previous pass
			existing, _ := dyn.GetByPath(newDst, path)
			if existing.IsValid() {
				return v, nil
			}

			// if parent 'resources' doesn't exist, handle it on the next pass
			updated, _ := dyn.SetByPath(newDst, path, v)
			if !updated.IsValid() {
				return v, nil
			} else {
				newDst = updated
				return v, nil
			}
		},
	)
	if err != nil {
		return newDst, err
	}

	// merge 'resources'
	_, err = dyn.MapByPattern(
		src,
		dyn.NewPattern(resourcesKey),
		func(path dyn.Path, v dyn.Value) (dyn.Value, error) {
			// if already exists, we already handled it in the previous pass
			existing, _ := dyn.GetByPath(newDst, path)
			if existing.IsValid() {
				return v, nil
			}

			updated, _ := dyn.SetByPath(newDst, path, v)
			if !updated.IsValid() {
				return v, errors.New("failed to update resources")
			} else {
				newDst = updated
				return v, nil
			}
		},
	)
	if err != nil {
		return newDst, err
	}

	return newDst, nil
}
