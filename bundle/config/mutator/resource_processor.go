package mutator

import (
	"context"
	"errors"

	"github.com/databricks/cli/libs/dyn/merge"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// ResourceProcessor handles initialization and normalization of resources.
// It should be used by mutators after they have added or modified resources.
type ResourceProcessor interface {
	Process(ctx context.Context, b *bundle.Bundle, opts ResourceProcessorOpts) diag.Diagnostics
}

type ResourceProcessorOpts struct {
	// AddedResources is a list of resources that have been added by mutator.
	//
	// These resources are first normalized, and then get default values.
	AddedResources ResourceKeySet

	// UpdatedResources are resources that have been previously added,
	// but have been updated by mutator.
	//
	// These resources are normalized.
	UpdatedResources ResourceKeySet
}

type resourceProcessor struct {
	initializeResources []bundle.Mutator
	normalizeResources  []bundle.Mutator
}

func NewResourceProcessor(initializeResources, normalizeResources []bundle.Mutator) ResourceProcessor {
	return resourceProcessor{
		initializeResources: initializeResources,
		normalizeResources:  normalizeResources,
	}
}

func (r resourceProcessor) Process(ctx context.Context, b *bundle.Bundle, opts ResourceProcessorOpts) diag.Diagnostics {
	diags := diag.Diagnostics{}

	if !opts.AddedResources.IsEmpty() {
		var snapshot dyn.Value

		err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
			snapshot = root

			return selectResources(root, opts.AddedResources)
		})
		if err != nil {
			return diags.Extend(diag.Errorf("failed to select resources: %s", err))
		}

		diags = diags.Extend(bundle.ApplySeq(ctx, b, r.normalizeResources...))
		if diags.HasError() {
			return diags
		}

		diags = diags.Extend(bundle.ApplySeq(ctx, b, r.initializeResources...))
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
	}

	if !opts.UpdatedResources.IsEmpty() {
		var snapshot dyn.Value

		err := b.Config.Mutate(func(root dyn.Value) (dyn.Value, error) {
			snapshot = root

			return selectResources(root, opts.UpdatedResources)
		})
		if err != nil {
			return diags.Extend(diag.Errorf("failed to select resources: %s", err))
		}

		diags = diags.Extend(bundle.ApplySeq(ctx, b, r.normalizeResources...))
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
