package validate

import (
	"context"
	"fmt"
	"sort"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/internal/validation/generated"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

type enum struct{}

func Enum() bundle.Mutator {
	return &enum{}
}

func (f *enum) Name() string {
	return "validate:enum"
}

func (f *enum) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	for k, validValues := range generated.EnumFields {
		pattern, err := dyn.NewPatternFromString(k)
		if err != nil {
			return diag.FromErr(fmt.Errorf("invalid pattern %q for enum field validation: %w", k, err))
		}

		_, err = dyn.MapByPattern(b.Config.Value(), pattern, func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
			// Skip validation if the value is not set
			if v.Kind() == dyn.KindInvalid || v.Kind() == dyn.KindNil {
				return v, nil
			}

			// Get the string value for comparison
			strValue, ok := v.AsString()
			if !ok {
				return v, nil
			}

			// Check if the value is in the list of valid enum values
			validValue := false
			for _, valid := range validValues {
				if strValue == valid {
					validValue = true
					break
				}
			}

			if !validValue {
				diags = diags.Append(diag.Diagnostic{
					Severity:  diag.Warning,
					Summary:   fmt.Sprintf("invalid value %q for enum field. Valid values are %v", strValue, validValues),
					Locations: v.Locations(),
					Paths:     []dyn.Path{p},
				})
			}

			return v, nil
		})
		if dyn.IsExpectedMapError(err) || dyn.IsExpectedSequenceError(err) || dyn.IsExpectedMapToIndexError(err) || dyn.IsExpectedSequenceToIndexError(err) {
			// No map or sequence value is set at this pattern, so we ignore it.
			continue
		}
		if err != nil {
			return diag.FromErr(err)
		}
	}

	// Sort diagnostics to make them deterministic
	sort.Slice(diags, func(i, j int) bool {
		// First sort by summary
		if diags[i].Summary != diags[j].Summary {
			return diags[i].Summary < diags[j].Summary
		}

		// Then sort by locations as a tie breaker if summaries are the same.
		iLocs := fmt.Sprintf("%v", diags[i].Locations)
		jLocs := fmt.Sprintf("%v", diags[j].Locations)
		return iLocs < jLocs
	})

	return diags
}
