package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

func UniqueResourceKeys() bundle.ReadOnlyMutator {
	return &uniqueResourceKeys{}
}

// TODO: Might need to enforce sorted walk on dyn.Walk
type uniqueResourceKeys struct{}

func (m *uniqueResourceKeys) Name() string {
	return "validate:unique_resource_keys"
}

func (m *uniqueResourceKeys) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// Map of resource key to the paths and locations the resource is defined at.
	paths := map[string][]dyn.Path{}
	locations := map[string][]dyn.Location{}

	_, err := dyn.Walk(rb.Config().Value().Get("resources"), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// The path is expected to be of length 2, and of the form <resource_type>.<resource_identifier>.
		// Eg: jobs.my_job, pipelines.my_pipeline, etc.
		if len(p) < 2 {
			return v, nil
		}
		if len(p) > 2 {
			return v, dyn.ErrSkip
		}

		// The key for the resource. Eg: "my_job" for jobs.my_job.
		k := p[1].Key()

		paths[k] = append(paths[k], p)
		locations[k] = append(locations[k], v.Locations()...)
		return v, nil
	})
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	for k, ps := range paths {
		if len(ps) <= 1 {
			continue
		}

		// TODO: What happens on target overrides? Ensure they do not misbehave.
		// 1. What was the previous behaviour for target overrides?
		// 2. What if a completely new resource with a conflicting key is defined
		// in a target override.
		//
		// If there are multiple resources with the same key, report an error.
		// NOTE: This includes if the same resource is defined in multiple files as
		// TODO: continue this comment.
		diags = append(diags, diag.Diagnostic{
			Severity:  diag.Error,
			Summary:   fmt.Sprintf("multiple resources have been defined with the same key: %s", k),
			Locations: locations[k],
			Paths:     ps,
		})
	}
	return diags
}
