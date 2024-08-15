package validate

import (
	"context"
	"fmt"
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
	rv := b.Config.Value().Get("resources")

	diags := diag.Diagnostics{}

	dyn.MapByPattern(
		rv,
		dyn.NewPattern(dyn.AnyKey(), dyn.AnyKey()),
		func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			if v.Kind() != dyn.KindNil {
				return v, nil
			}

			// Type of the resource, stripped of the trailing 's' to make it
			// singular.
			rType := strings.TrimSuffix(p[0].Key(), "s")

			// Name of the resource. Eg: "foo" in "jobs.foo".
			rName := p[1].Key()

			// Prepend "resources" to the path.
			fullPath := dyn.NewPath(dyn.Key("resources"))
			fullPath = append(fullPath, p...)

			diags = append(diags, diag.Diagnostic{
				Severity:  diag.Error,
				Summary:   fmt.Sprintf("%s %s is not defined", rType, rName),
				Locations: v.Locations(),
				Paths:     []dyn.Path{fullPath},
			})

			return v, nil
		},
	)

	return diags
}
