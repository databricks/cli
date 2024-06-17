package validate

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

	func DeadConfiguration() bundle.ReadOnlyMutator {
	return &deadConfiguration{}
}

type deadConfiguration struct{}

func (v *deadConfiguration) Name() string {
	return "validate:dead_configuration"
}

// TODO: Does a key without a value have kind nil? In that case maybe also check for nil values and include a warning.

func isKindScalar(v dyn.Value) bool {
	return v.Kind() == dyn.KindString || v.Kind() == dyn.KindInt || v.Kind() == dyn.KindBool || v.Kind() == dyn.KindFloat || v.Kind() == dyn.KindTime
}

func deadConfigurationWarning(v dyn.Value, p dyn.Path, rb bundle.ReadOnlyBundle) string {
	loc := v.Location()
	rel, err := filepath.Rel(rb.RootPath(), loc.File)
	if err == nil {
		loc.File = rel
	}

	yamlLocations := v.YamlLocations()
	for i, yamlLocation := range yamlLocations {
		rel, err := filepath.Rel(rb.RootPath(), yamlLocation.File)
		if err == nil {
			yamlLocations[i].File = rel
		}
	}

	return fmt.Sprintf("Multiple values found for the same configuration %s. Only the value from location %s will be used. Locations found: %s", p.String(), loc, yamlLocations)
}

func (v *deadConfiguration) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	_, err := dyn.Walk(rb.Config().Value(), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		if !isKindScalar(v) {
			return v, nil
		}

		if len(v.YamlLocations()) >= 2 {
			// TODO: See how this renders in the terminal. Is the configuration key clear?
			diags = diags.Append(diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  deadConfigurationWarning(v, p, rb),
				Location: v.Location(),
				Path:     p,
			})
		}
		return v, nil
	})
	diags = append(diags, diag.FromErr(err)...)
	return diags
}
