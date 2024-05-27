package validate

import (
	"context"
	"fmt"
	"slices"
	"strings"

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

// Validate a single resource is only defined at a single location. We do not allow
// a single resource to be defined at multiple files.
// func validateSingleYamlFile(rv dyn.Value, rp dyn.Path) error {
// 	rrv, err := dyn.GetByPath(rv, rp)
// 	if err != nil {
// 		return err
// 	}

// 	_, err = dyn.Walk(rrv, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
// 		locations := v.YamlLocations()
// 		if len(locations) <= 1 {
// 			return v, nil
// 		}

// 		return v, fmt.Errorf("resource %s has been defined at multiple locations: (%s)", rp.String(), strings.Join(ls, ", "))
// 	})
// 	return err
// }

// TODO: Ensure all duplicate key sites are returned to the user in the diagnostics.
// TODO: Add unit tests for this mutator.
// TODO: l is not effective location because of additive semantics. Clarify this in the
// documentation.
func (m *uniqueResourceKeys) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	// Map of resource identifiers to their paths.
	// Example entry: my_job -> jobs.my_job
	seenPaths := make(map[string]dyn.Path)

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
		if _, ok := seenPaths[k]; ok {
			// Value of the resource that's already been seen
			ov, _ := dyn.GetByPath(rv, seenPaths[k])

			// Value of the newly encountered resource with a duplicate identifier.
			nv, _ := dyn.GetByPath(rv, p)

			// Error, encountered a duplicate resource identifier.
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("multiple resources named %s (%s at %s, %s at %s)", k, seenPaths[k].String(), ov.Location(), p.String(), nv.Location()),
				Location: nv.Location(),
			})
		}

		s := p.String()
		if s == "jobs.foo" {
			fmt.Println("jobs.foo")
		}

		yamlLocations := v.YamlLocations()
		if len(yamlLocations) > 1 {
			// Sort locations to make the error message deterministic.
			ls := make([]string, 0, len(yamlLocations))
			for _, l := range yamlLocations {
				ls = append(ls, l.String())
			}
			slices.Sort(ls)

			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  fmt.Sprintf("resource %s has been defined at multiple locations: (%s)", p.String(), strings.Join(ls, ", ")),
				Location: v.Location(),
			})
		}

		// err := validateSingleYamlFile(rv, p)
		// if err != nil {
		// 	diags = append(diags, diag.Diagnostic{
		// 		Severity: diag.Error,
		// 		Summary:  err.Error(),
		// 		Location: v.Location(),
		// 	})
		// }

		// Accumulate the resource identifier and its path.
		seenPaths[k] = p
		return v, nil
	})

	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}
