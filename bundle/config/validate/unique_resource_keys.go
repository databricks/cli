package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// TODO: Ensure the solution here works for dup keys in the same resource type.

func UniqueResourceKeys() bundle.ReadOnlyMutator {
	return &uniqueResourceKeys{}
}

type uniqueResourceKeys struct{}

func (m *uniqueResourceKeys) Name() string {
	return "validate:unique_resource_keys"
}

// TODO: Ensure all duplicate key sites are returned to the user in the diagnostics.
func (m *uniqueResourceKeys) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	paths := make(map[string]dyn.Path)
	rv := rb.Config().ReadOnlyValue().Get("resources")

	// Walk the resources tree and accumulate the resource identifiers.
	_, err := dyn.Walk(rv, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// The path is expected to be of length 2, and of the form <resource_type>.<resource_identifier>.
		// Eg: jobs.my_job, pipelines.my_pipeline, etc.
		if len(p) < 2 {
			return v, nil
		}
		if len(p) > 2 {
			return v, dyn.ErrSkip
		}

		// If the resource identifier already exists in the map, return an error.
		k := p[1].Key()
		if _, ok := paths[k]; ok {
			// Value of the resource that's already been seen
			ov, _ := dyn.GetByPath(rv, paths[k])

			// Value of the newly encountered resource with a duplicate identifier.
			nv, _ := dyn.GetByPath(rv, p)

			// Error, encountered a duplicate resource identifier.
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("multiple resources named %s (%s at %s, %s at %s)", k, paths[k].String(), ov.Location(), p.String(), nv.Location()),
				Location: nv.Location(),
			})
		}

		// Accumulate the resource identifier and its path.
		paths[k] = p
		return v, nil
	})

	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}
