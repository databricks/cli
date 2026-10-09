package validate

import (
	"context"
	"fmt"
	"maps"
	"slices"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/internal/validation/generated"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
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

	// Generate prefix tree for all enum fields.
	patterns, err := newPatternSet(slices.Collect(maps.Keys(generated.EnumFields)))
	if err != nil {
		return diag.FromErr(fmt.Errorf("enum field validation: %w", err))
	}

	err = structvar.Walk(b.Config.View(), func(np *structpath.PathNode, v structvar.View) error {
		// If the path matches no pattern, we do not need to validate any enum
		// fields in it.
		pattern, ok := patterns.find(np)
		if !ok {
			return nil
		}

		// Get the string value for comparison
		strValue, ok := v.AsString()
		if !ok {
			return nil
		}

		// Skip validation for values containing variable references (e.g.
		// ${resources.jobs.my_job.id}) since they are not yet resolved.
		if structvar.ContainsVariableReference(strValue) {
			return nil
		}

		// Get valid values for this pattern
		validValues := generated.EnumFields[pattern]

		// Check if the value is in the list of valid enum values
		validValue := slices.Contains(validValues, strValue)

		if !validValue {
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  fmt.Sprintf("invalid value %q for enum field. Valid values are %v", strValue, validValues),
				Paths:    []*structpath.PathNode{np},
			})
		}

		return nil
	})
	if err != nil {
		return diag.FromErr(err)
	}

	sortDiagnostics(diags)

	return diags
}
