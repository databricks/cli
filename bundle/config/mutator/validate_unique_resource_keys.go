package mutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// TODO: Ensure the solution here works for dup keys in the same resource type.

type validateUniqueResourceKeys struct{}

func ValidateUniqueResourceKeys() bundle.Mutator {
	return &validateUniqueResourceKeys{}
}

func (m *validateUniqueResourceKeys) Name() string {
	return "ValidateUniqueResourceKeys"
}

// TODO: Make this a readonly mutator.
// TODO: Make this a bit terser.

// TODO: Ensure all duplicate key sites are returned to the user in the diagnostics.
func (m *validateUniqueResourceKeys) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		paths := make(map[string]dyn.Path)
		rv := v.Get("resources")

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
				// Location of the existing resource in the map.
				ov, _ := dyn.GetByPath(rv, paths[k])
				ol := ov.Location()

				// Location of the newly encountered with a duplicate name.
				nv, _ := dyn.GetByPath(rv, p)
				nl := nv.Location()

				// Error, encountered a duplicate resource identifier.
				// TODO: Set location for one of the resources?
				diags = append(diags, diag.Diagnostic{
					Severity: diag.Error,
					Summary:  fmt.Sprintf("multiple resources named %s (%s at %s, %s at %s)", k, paths[k].String(), nl, p.String(), ol),
					Location: nl,
				})
			}

			// Accumulate the resource identifier and its path.
			paths[k] = p
			return v, nil
		})
		return v, err
	})
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}
