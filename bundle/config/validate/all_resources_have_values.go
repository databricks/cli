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

func AllResourcesHaveValues() bundle.Mutator {
	return &allResourcesHaveValues{}
}

type allResourcesHaveValues struct{}

func (m *allResourcesHaveValues) Name() string {
	return "validate:AllResourcesHaveValues"
}

func (m *allResourcesHaveValues) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	_, err := dyn.MapByPattern(
		b.Config.Value(),
		dyn.NewPattern(dyn.Key("resources"), dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if v.Kind() != dyn.KindNil {
				return v, nil
			}

			// Type of the resource, stripped of the trailing 's' to make it
			// singular.
			rType := strings.TrimSuffix(p[1].Key(), "s")

			// Name of the resource. Eg: "foo" in "jobs.foo".
			rName := p[2].Key()

			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("%s %s is not defined", rType, rName),
				Locations: v.Locations(),
				Paths:     []dyn.Path{slices.Clone(p)},
			})

			return v, nil
		},
	)
	if err != nil {
		diags = append(diags, diag.FromErr(err)...)
	}

	return diags
}
