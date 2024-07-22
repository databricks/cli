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
// TODO: return multiple locations with diagnostics. Can be a followup.
type uniqueResourceKeys struct{}

func (m *uniqueResourceKeys) Name() string {
	return "validate:unique_resource_keys"
}

func conflictingResourceKeysErr(key string, p1 dyn.Path, l1 dyn.Location, p2 dyn.Path, l2 dyn.Location) diag.Diagnostic {
	return diag.Diagnostic{
		Severity: diag.Error,
		Summary:  fmt.Sprintf("multiple resources have been defined with the same key: %s (%s at %s, %s at %s)", key, p1, l1, p2, l2),
		Location: l1,
	}
}

func (m *uniqueResourceKeys) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	type r struct {
		p dyn.Path
		l dyn.Location
	}
	seenResource := make(map[string]r)

	seenResources := make(map[string]dyn.Location)
	_, err := dyn.Walk(rb.Config().Value().Get("resources"), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		// The path is expected to be of length 2, and of the form <resource_type>.<resource_identifier>.
		// Eg: jobs.my_job, pipelines.my_pipeline, etc.
		if len(p) < 2 {
			return v, nil
		}
		if len(p) > 2 {
			return v, dyn.ErrSkip
		}

		// Each resource should be completely defined in a single YAML file. We
		// do not allow users to split the definition of a single resource across
		// multiple files.
		// Users can use simple / complex variables to modularize their configuration.
		if locations := v.Locations(); len(locations) >= 2 {
			diags = append(diags, conflictingResourceKeysErr(p[1].Key(), p, locations[0], p, locations[1]))
		}

		// l, ok := seenResources[p[1].Key()]
		// if ok {
		// 	diags = append(diags, conflictingResourceKeysErr(p[1].Key(), p, l, p, v.Locations()[0]))
		// } else {
		// 	seenResources[p[1].Key()] = v.Locations()[0]
		// }

		// // key for the source. Eg: "my_job" for jobs.my_job.
		// key := p[1].Key()
		// info, ok := seenResources[key]

		// for _, l := range v.Locations() {
		// 	info, ok := seenResources[p[1].Key()]
		// 	if !ok {
		// 		seenResources[p[1].Key()] = resourceInfo{p, l}
		// 		continue
		// 	}

		// 	diags = append(diags)
		// }
		return v, nil
	})
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}
