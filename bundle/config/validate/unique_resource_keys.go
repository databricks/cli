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

	// Maps of key to the paths and locations the resource / script is defined at.
	resourceAndScriptMetadata := map[string]*metadata{}
	addLocationToMetadata := func(k, prefix string, p dyn.Path, v dyn.Value) {
		mv, ok := resourceAndScriptMetadata[k]
		if !ok {
			mv = &metadata{
				paths:     nil,
				locations: nil,
			}
		}

		mv.paths = append(mv.paths, dyn.NewPath(dyn.Key(prefix)).Append(p...))
		mv.locations = append(mv.locations, v.Locations()...)

		resourceAndScriptMetadata[k] = mv
	}

	// Gather the paths and locations of all resources
	rv := b.Config.Value().Get("resources")
	if rv.Kind() != dyn.KindInvalid && rv.Kind() != dyn.KindNil {
		_, err := dyn.MapByPattern(
			rv,
			dyn.NewPattern(dyn.AnyKey(), dyn.AnyKey()),
			func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				// The key for the resource. Eg: "my_job" for jobs.my_job.
				k := p[1].Key()
				addLocationToMetadata(k, "resources", p, v)
				return v, nil
			},
		)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// track locations for all scripts.
	sv := b.Config.Value().Get("scripts")
	if sv.Kind() != dyn.KindInvalid && sv.Kind() != dyn.KindNil {
		_, err := dyn.MapByPattern(
			sv,
			dyn.NewPattern(dyn.AnyKey()),
			func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				// The key for the script. Eg: "my_script" for scripts.my_script.
				k := p[0].Key()
				addLocationToMetadata(k, "scripts", p, v)
				return v, nil
			},
		)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// If duplicate keys are found, report an error.
	for k, v := range resourceAndScriptMetadata {
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
			Summary:   "multiple resources or scripts have been defined with the same key: " + k,
			Locations: v.locations,
			Paths:     v.paths,
		})
	}

	return diags
}
