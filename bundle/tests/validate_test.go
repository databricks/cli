package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/validate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
)

func TestConflictingConfigurationValidate(t *testing.T) {
	b := load(t, "validate/conflicting_configuration")

	ctx := context.Background()
	diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), validate.ConflictingConfiguration())

	assert.Len(t, diags, 2)
	assert.Contains(t, diags, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "Multiple values found for the same configuration variables.baz.default. Only the value from location resources2.yml:3:14 will be used. Locations found: [resources2.yml:3:14 resources1.yml:9:14]",
		Location: dyn.Location{
			File:   "validate/conflicting_configuration/resources2.yml",
			Line:   3,
			Column: 14,
		},
		Path: dyn.MustPathFromString("variables.baz.default"),
	})
	assert.Equal(t, diags[0], diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "Multiple values found for the same configuration variables.bar.default. Only the value from location resources1.yml:6:14 will be used. Locations found: [resources1.yml:6:14 databricks.yml:9:14]",
		Location: dyn.Location{
			File:   "validate/conflicting_configuration/resources1.yml",
			Line:   6,
			Column: 14,
		},
		Path: dyn.MustPathFromString("variables.bar.default"),
	})
}
