package validate

import (
	"context"
	"sort"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
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
		locations []dyn.Location
		paths     []dyn.Path
	}

	// Maps of resource key to the paths and locations the resource is defined at.
	resourceMetadata := map[string]*metadata{}

	rv := b.Config.Value().Get("resources")

	// return early if no resources are defined or the resources block is empty.
	if rv.Kind() == dyn.KindInvalid || rv.Kind() == dyn.KindNil {
		return diags
	}

	// Gather the paths and locations of all resources.
	_, err := dyn.MapByPattern(
		rv,
		dyn.NewPattern(dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// The key for the resource. Eg: "my_job" for jobs.my_job.
			k := p[1].Key()

			m, ok := resourceMetadata[k]
			if !ok {
				m = &metadata{
					paths:     []dyn.Path{},
					locations: []dyn.Location{},
				}
			}

			m.paths = append(m.paths, p)
			m.locations = append(m.locations, v.Locations()...)

			resourceMetadata[k] = m
			return v, nil
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}

	for k, v := range resourceMetadata {
		if len(v.locations) <= 1 {
			continue
		}

		// Sort the locations and paths for consistent error messages. This helps
		// with unit testing.
		sort.Slice(v.locations, func(i, j int) bool {
			l1 := v.locations[i]
			l2 := v.locations[j]

			if l1.File != l2.File {
				return l1.File < l2.File
			}
			if l1.Line != l2.Line {
				return l1.Line < l2.Line
			}
			return l1.Column < l2.Column
		})
		sort.Slice(v.paths, func(i, j int) bool {
			return v.paths[i].String() < v.paths[j].String()
		})

		// If there are multiple resources with the same key, report an error.
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   "multiple resources have been defined with the same key: " + k,
			Locations: v.locations,
			Paths:     v.paths,
		})
	}

	return diags
}
