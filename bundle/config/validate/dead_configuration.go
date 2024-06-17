package validate

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
)

// This mutator emits warnings if a configuration value is being override by another
// value in the configuration files, effectively making the configuration useless.
func DeadConfiguration() bundle.ReadOnlyMutator {
	return &deadConfiguration{}
}

type deadConfiguration struct{}

func (v *deadConfiguration) Name() string {
	return "validate:dead_configuration"
}

func isKindScalar(v dyn.Value) bool {
	switch v.Kind() {
	case dyn.KindString, dyn.KindInt, dyn.KindBool, dyn.KindFloat, dyn.KindTime, dyn.KindNil:
		return true
	default:
		return false
	}
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

// TODO: Did the target override issues have to do with "default" preset?
func (v *deadConfiguration) Apply(ctx context.Context, rb bundle.ReadOnlyBundle) diag.Diagnostics {
	diags := diag.Diagnostics{}

	_, err := dyn.Walk(rb.Config().Value(), func(p dyn.Path, v dyn.Value) (dyn.Value, error) {
		if !isKindScalar(v) {
			return v, nil
		}

		if len(v.YamlLocations()) >= 2 {
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
