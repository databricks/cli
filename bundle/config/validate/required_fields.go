package validate

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/internal/validation/generated"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type requiredFields struct{}

// TODO: also add to fast validate.
func RequiredFields() bundle.Mutator {
	return &requiredFields{}
}

func (f *requiredFields) Name() string {
	return "validate:required_fields"
}

func (f *requiredFields) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for k, requiredFields := range generated.RequiredFields {
		pattern, err := dyn.NewPatternFromString(k)
		if err != nil {
			return diag.FromErr(fmt.Errorf("invalid pattern %q for required field validation: %w", k, err))
		}

		dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			for _, requiredField := range requiredFields {
				vv := v.Get(requiredField)
				if vv.Kind() == dyn.KindInvalid || vv.Kind() == dyn.KindNil {
					diags = diags.Append(diag.Diagnostic{
						Severity:  diag.Error,
						Summary:   fmt.Sprintf("required field %q is not set", requiredField),
						Locations: v.Locations(),
						Paths:     []dyn.Path{p},
					})

					return dyn.NilValue, fmt.Errorf("required field %q is not set", requiredField)
				}
			}
			return v, nil
		})
	}

	return diags
}
