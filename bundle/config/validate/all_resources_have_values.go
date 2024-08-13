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
	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
		rv := v.Get("resources")

		// Skip if there are no resources block defined, or the resources block is empty.
		if rv.Kind() == dyn.KindInvalid || rv.Kind() == dyn.KindNil {
			return v, nil
		}

		_, err := dyn.MapByPattern(
			rv,
			dyn.NewPattern(dyn.AnyKey(), dyn.AnyKey()),
			func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
				if v.Kind() == dyn.KindInvalid || v.Kind() == dyn.KindNil {
					// Type of the resource, stripped of the trailing 's' to make it
					// singular.
					rType := strings.TrimSuffix(p[0].Key(), "s")

					rName := p[1].Key()
					return v, fmt.Errorf("%s %s is not defined", rType, rName)
				}
				return v, nil
			},
		)
		return v, err

	})
	return diag.FromErr(err)
}
