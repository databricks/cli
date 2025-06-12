package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/internal/validation/generated"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type required struct{}

func Required() bundle.Mutator {
	return &required{}
}

func (f *required) Name() string {
	return "validate:required"
}

func (f *required) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for k, requiredFields := range generated.RequiredFields {
		pattern, err := dyn.NewPatternFromString(k)
		if err != nil {
			return diag.FromErr(fmt.Errorf("invalid pattern %q for required field validation: %w", k, err))
		}

		_, err = dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			for _, requiredField := range requiredFields {
				vv := v.Get(requiredField)
				if vv.Kind() == dyn.KindInvalid || vv.Kind() == dyn.KindNil {
					diags = diags.Append(diag.Diagnostic{
						Severity:  diag.Warning,
						Summary:   fmt.Sprintf("required field %q is not set", requiredField),
						Locations: v.Locations(),
						Paths:     []dyn.Path{p},
					})
				}
			}
			return v, nil
		})
		if dyn.IsExpectedMapError(err) || dyn.IsExpectedSequenceError(err) || dyn.IsExpectedMapToIndexError(err) || dyn.IsExpectedSequenceToIndexError(err) {
			// No map or sequence defined for this pattern, so we ignore it.
			continue
		}
		if err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
