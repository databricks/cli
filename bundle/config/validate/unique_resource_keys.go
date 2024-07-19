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

	type resourceInfo struct {
		p dyn.Path
		l dyn.Location
	}

	seenResources := make(map[string]resourceInfo)
	_, err := dyn.Walk(rb.Config().Value().Get("resources"), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// The path is expected to be of length 2, and of the form <resource_type>.<resource_identifier>.
		// Eg: jobs.my_job, pipelines.my_pipeline, etc.
		if len(p) < 2 {
			return v, nil
		}
		if len(p) > 2 {
			return v, dyn.ErrSkip
		}

		if len(v.Locations()) == 0 {

		// key for the source. Eg: "my_job" for jobs.my_job.
		key := p[1].Key()
		info, ok := seenResources[key]

		for _, l := range v.Locations() {
			info, ok := seenResources[p[1].Key()]
			if !ok {
				seenResources[p[1].Key()] = resourceInfo{p, l}
				continue
			}

			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("multiple resources have been defined with the same key: %s (%s at %s, %s at %s)", p[1].Key(), p, l, info.p, info.l),
				Location: l,
				Path:     p,
			})
		}
		return v, nil
	})
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}
