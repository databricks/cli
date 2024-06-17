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

func TestDeadConfigurationValidate(t *testing.T) {
	b := load(t, "validate/dead_configuration")

	ctx := context.Background()
	diags := bundle.ApplyReadOnly(ctx, bundle.ReadOnly(b), validate.DeadConfiguration())

	assert.Len(t, diags, 2)
	assert.Contains(t, diags, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "Multiple values found for the same configuration bundle.name. Only the value from location resources1.yml:2:9 will be used. Locations found: [resources1.yml:2:9 databricks.yml:2:9]",
		Location: dyn.Location{
			File:   "validate/dead_configuration/resources1.yml",
			Line:   2,
			Column: 9,
		},
		Path: dyn.MustPathFromString("bundle.name"),
	})
	assert.Equal(t, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  "Multiple values found for the same configuration bundle.compute_id. Only the value from location resources2.yml:2:15 will be used. Locations found: [resources2.yml:2:15 resources1.yml:3:15]",
		Location: dyn.Location{
			File:   "validate/dead_configuration/resources2.yml",
			Line:   2,
			Column: 15,
		},
		Path: dyn.MustPathFromString("bundle.compute_id"),
	}, diags[0])
}
