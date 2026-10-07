package validate

import (
	"cmp"
	"context"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
)

// This mutator validates that:
//
//  1. Each resource key is unique across different resource types. No two resources
//     of the same type can have the same key. This is because command like "bundle run"
//     rely on the resource key to identify the resource to run.
//     Eg: jobs.foo and pipelines.foo are not allowed simultaneously.
//
//  2. Each resource definition is contained within a single file, and is not spread
//     across multiple files. Note: This is not applicable to resource configuration
//     defined in a target override. That is why this mutator MUST run before the target
//     overrides are merged.
func UniqueResourceKeys() bundle.Mutator {
	return &uniqueResourceKeys{}
}

type uniqueResourceKeys struct{}

func (m *uniqueResourceKeys) Name() string {
	return "validate:unique_resource_keys"
}

func (m *uniqueResourceKeys) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	type metadata struct {
		locations []diag.Location
		paths     []*structpath.PathNode
	}

	// Maps of key to the paths and locations the resource / script is defined at.
	resourceAndScriptMetadata := map[string]*metadata{}
	addLocationToMetadata := func(k string, fullPath *structpath.PathNode) {
		mv, ok := resourceAndScriptMetadata[k]
		if !ok {
			mv = &metadata{
				paths:     nil,
				locations: nil,
			}
		}

		mv.paths = append(mv.paths, fullPath)
		mv.locations = append(mv.locations, b.Config.LocationsAt(fullPath)...)

		resourceAndScriptMetadata[k] = mv
	}

	// Gather the paths and locations of all resources
	for _, group := range b.Config.Resources.AllResources() {
		for k := range group.Resources {
			addLocationToMetadata(k, structpath.NewPath(nil, "resources", group.Description.PluralName, k))
		}
	}

	// track locations for all scripts.
	for k := range b.Config.Scripts {
		addLocationToMetadata(k, structpath.NewPath(nil, "scripts", k))
	}

	// If duplicate keys are found, report an error.
	for k, v := range resourceAndScriptMetadata {
		if len(v.locations) <= 1 {
			continue
		}

		// Sort the locations and paths for consistent error messages. This helps
		// with unit testing.
		slices.SortFunc(v.locations, func(a, b diag.Location) int {
			if n := cmp.Compare(a.File, b.File); n != 0 {
				return n
			}
			if n := cmp.Compare(a.Line, b.Line); n != 0 {
				return n
			}
			return cmp.Compare(a.Column, b.Column)
		})
		slices.SortFunc(v.paths, func(a, b *structpath.PathNode) int {
			return cmp.Compare(a.String(), b.String())
		})

		// If there are multiple resources with the same key, report an error.
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "multiple resources or scripts have been defined with the same key: " + k,
			Locations: v.locations,
			Paths:     v.paths,
		})
	}

	return diags
}
