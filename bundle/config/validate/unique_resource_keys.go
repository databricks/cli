package validate

import (
	"context"
	"fmt"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

func UniqueResourceKeys() bundle.Mutator {
	return &uniqueResourceKeys{}
}

// TODO: Comment why this mutator needs to be run before target overrides.
type uniqueResourceKeys struct{}

func (m *uniqueResourceKeys) Name() string {
	return "validate:unique_resource_keys"
}

func (m *uniqueResourceKeys) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// Map of resource key to the pathsByKey and locations the resource is defined at.
	pathsByKey := map[string][]dyn.Path{}
	locationsByKey := map[string][]dyn.Location{}

	// Gather the paths and locations of all resources.
	// TODO: confirm MapByPattern behaves as I expect it to.
	_, err := dyn.MapByPattern(
		b.Config.Value().Get("resources"),
		dyn.NewPattern(dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// The key for the resource. Eg: "my_job" for jobs.my_job.
			k := p[1].Key()

			// dyn.Path under the hood is a slice. So, we need to clone it.
			pathsByKey[k] = append(pathsByKey[k], slices.Clone(p))

			locationsByKey[k] = append(locationsByKey[k], v.Locations()...)
			return v, nil
		},
	)
	if err != nil {
		return diag.FromErr(err)
	}

	for k, locations := range locationsByKey {
		if len(locations) <= 1 {
			continue
		}

		// If there are multiple resources with the same key, report an error.
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("multiple resources have been defined with the same key: %s", k),
			Locations: locations,
			Paths:     pathsByKey[k],
		})
	}

	return diags
}
