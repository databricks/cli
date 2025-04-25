package resourcemutator

import (
	"context"
	"errors"

	"github.com/databricks/cli/bundle/config/mutator"

	"github.com/databricks/cli/libs/dyn/merge"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// When a new resource is added to configuration, we apply bundle
// settings and defaults to it. Initialization is applied only once.
//
// If bundle is modified outside of 'resources' section, these changes are discarded.
func applyInitializeMutators(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := bundle.ApplySeq(
		ctx,
		b,
		// Reads (typed): b.Config.RunAs, b.Config.Workspace.CurrentUser (validates run_as configuration)
		// Reads (dynamic): run_as (checks if run_as is specified)
		// Updates (typed): b.Config.Resources.Jobs[].RunAs (sets job run_as fields to bundle run_as; only if Experimental.UseLegacyRunAs is set)
		// Updates (typed): range b.Config.Resources.Pipelines[].Permissions (set permission based on bundle run_as; only if Experimental.UseLegacyRunAs is set)
		SetRunAs(),

		// Reads (typed): b.Config.Bundle.{Mode,ClusterId} (checks mode and cluster ID settings)
		// Reads (env): DATABRICKS_CLUSTER_ID (environment variable for backward compatibility)
		// Reads (typed): b.Config.Resources.Jobs.*.Tasks.*.ForEachTask
		// Updates (typed): b.Config.Bundle.ClusterId (sets from environment if in development mode)
		// Updates (typed): b.Config.Resources.Jobs.*.Tasks.*.{NewCluster,ExistingClusterId,JobClusterKey,EnvironmentKey} (replaces compute settings with specified cluster ID)
		// OR corresponding fields on ForEachTask if that is present
		// Overrides job compute settings with a specified cluster ID for development or testing
		OverrideCompute(),

		// ApplyPresets should have more priority than defaults below, so it should be run first
		ApplyPresets(),
	)

	if diags.HasError() {
		return diags
	}

	defaults := []struct {
		pattern string
		value   any
	}{
		{"resources.dashboards.*.parent_path", b.Config.Workspace.ResourcePath},
		{"resources.dashboards.*.embed_credentials", false},
		{"resources.volumes.*.volume_type", "MANAGED"},
	}

	for _, defaultDef := range defaults {
		diags = diags.Extend(bundle.SetDefault(ctx, b, defaultDef.pattern, defaultDef.value))
		if diags.HasError() {
			return diags
		}
	}

	diags = diags.Extend(bundle.ApplySeq(ctx, b,
		// Reads (typed): b.Config.Resources.Jobs (checks job configurations)
		// Updates (typed): b.Config.Resources.Jobs[].Queue (sets Queue.Enabled to true for jobs without queue settings)
		// Enable queueing for jobs by default, following the behavior from API 2.2+.
		DefaultQueueing(),

		// Reads (typed): b.Config.Permissions (validates permission levels)
		// Reads (dynamic): resources.{jobs,pipelines,experiments,models,model_serving_endpoints,dashboards,apps}.*.permissions (reads existing permissions)
		// Updates (dynamic): resources.{jobs,pipelines,experiments,models,model_serving_endpoints,dashboards,apps}.*.permissions (adds permissions from bundle-level configuration)
		// Applies bundle-level permissions to all supported resources
		ApplyBundlePermissions(),

		// Reads (typed): b.Config.Workspace.CurrentUser.UserName (gets current user name)
		// Updates (dynamic): resources.*.*.permissions (removes permissions entries where user_name or service_principal_name matches current user)
		// Removes the current user from all resource permissions as the Terraform provider implicitly grants ownership
		FilterCurrentUser(),
	))

	return diags
}

// Normalization is applied multiple times if resource is modified during initialization
//
// If bundle is modified outside of 'resources' section, these changes are discarded.
func applyNormalizeMutators(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	return bundle.ApplySeq(
		ctx,
		b,
		// Reads (dynamic): * (strings) (searches for variable references in string values)
		// Updates (dynamic): resources.* (strings) (resolves variable references to their actual values)
		// Resolves variable references in 'resources' using bundle, workspace, and variables prefixes
		mutator.ResolveVariableReferencesOnlyResources(
			"bundle",
			"workspace",
			"variables",
		),

		// Reads (dynamic): resources.pipelines.*.libraries (checks for notebook.path and file.path fields)
		// Updates (dynamic): resources.pipelines.*.libraries (expands glob patterns in path fields to multiple library entries)
		// Expands glob patterns in pipeline library paths to include all matching files
		ExpandPipelineGlobPaths(),

		// Reads (dynamic): resources.jobs.*.job_clusters (reads job clusters to merge)
		// Updates (dynamic): resources.jobs.*.job_clusters (merges job clusters with the same job_cluster_key)
		MergeJobClusters(),

		// Reads (dynamic): resources.jobs.*.parameters (reads job parameters to merge)
		// Updates (dynamic): resources.jobs.*.parameters (merges job parameters with the same name)
		MergeJobParameters(),

		// Reads (dynamic): resources.jobs.*.tasks (reads job tasks to merge)
		// Updates (dynamic): resources.jobs.*.tasks (merges job tasks with the same task_key)
		MergeJobTasks(),

		// Reads (dynamic): resources.pipelines.*.clusters (reads pipeline clusters to merge)
		// Updates (dynamic): resources.pipelines.*.clusters (merges pipeline clusters with the same label)
		MergePipelineClusters(),

		// Reads (dynamic): resources.apps.*.resources (reads app resources to merge)
		// Updates (dynamic): resources.apps.*.resources (merges app resources with the same name)
		MergeApps(),

		// Reads (typed): resources.pipelines.*.{catalog,schema,target}, resources.volumes.*.{catalog_name,schema_name} (checks for schema references)
		// Updates (typed): resources.pipelines.*.{schema,target}, resources.volumes.*.schema_name (converts implicit schema references to explicit ${resources.schemas.<schema_key>.name} syntax)
		// Translates implicit schema references in DLT pipelines or UC Volumes to explicit syntax to capture dependencies
		CaptureSchemaDependency(),
	)
}

// NormalizeAndInitializeResources initializes and normalizes specified resources,
// and should be used by mutators after they have added resources.
func NormalizeAndInitializeResources(
	ctx context.Context,
	b *bundle.Bundle,
	addedResources ResourceKeySet,
) diag.Diagnostics {
	if addedResources.IsEmpty() {
		return nil
	}

	var diags diag.Diagnostics
	var snapshot dyn.Value

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		snapshot = root

		return selectResources(root, addedResources)
	})
	if err != nil {
		return diags.Extend(diag.Errorf("failed to select resources: %s", err))
	}

	diags = diags.Extend(applyNormalizeMutators(ctx, b))
	if diags.HasError() {
		return diags
	}

	diags = diags.Extend(applyInitializeMutators(ctx, b))
	if diags.HasError() {
		return diags
	}

	// after mutators, we merge updated resources back to snapshot to preserve non-selected resources
	err = b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return mergeResources(root, snapshot)
	})
	if err != nil {
		return diags.Extend(diag.Errorf("failed to merge resources: %s", err))
	}

	return diags
}

// NormalizeResources normalizes resources specified resources,
// and should be used by mutators after they have updated resources.
func NormalizeResources(
	ctx context.Context,
	b *bundle.Bundle,
	updatedResources ResourceKeySet,
) diag.Diagnostics {
	if updatedResources.IsEmpty() {
		return nil
	}

	var diags diag.Diagnostics
	var snapshot dyn.Value

	err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		snapshot = root

		return selectResources(root, updatedResources)
	})
	if err != nil {
		return diags.Extend(diag.Errorf("failed to select resources: %s", err))
	}

	diags = diags.Extend(applyNormalizeMutators(ctx, b))
	if diags.HasError() {
		return diags
	}

	// after mutators, we merge updated resources back to snapshot to preserve non-selected resources
	err = b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return mergeResources(root, snapshot)
	})
	if err != nil {
		return diags.Extend(diag.Errorf("failed to merge resources: %s", err))
	}

	return diags
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
