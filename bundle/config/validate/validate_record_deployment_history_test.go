package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	bundleenv "github.com/databricks/cli/bundle/env"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateRecordDeploymentHistory(t *testing.T) {
	tests := []struct {
		name      string
		enabled   bool
		recordEnv string
		wantError bool
	}{
		{name: "flag unset", enabled: false, wantError: false},
		{name: "flag set", enabled: true, wantError: true},
		{name: "flag set with record env", enabled: true, recordEnv: "true", wantError: false},
		{name: "flag set with empty record env", enabled: true, recordEnv: "", wantError: true},
		{name: "flag unset with record env", enabled: false, recordEnv: "true", wantError: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Experimental: &config.Experimental{RecordDeploymentHistory: tc.enabled},
				},
			}

			ctx := env.Set(t.Context(), bundleenv.RecordDeploymentHistoryVariable, tc.recordEnv)
			diags := ValidateRecordDeploymentHistory().Apply(ctx, b)

			if !tc.wantError {
				assert.Empty(t, diags)
				return
			}
			require.Len(t, diags, 1)
			assert.Equal(t, diag.Error, diags[0].Severity)
			assert.Equal(t, "experimental.record_deployment_history is not supported yet; remove this setting from your bundle configuration", diags[0].Summary)
			assert.Equal(t, recordDeploymentHistoryPath, diags[0].Paths[0].String())
		})
	}
}

func TestValidateRecordDeploymentHistoryNoExperimentalBlock(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{}}
	assert.Empty(t, ValidateRecordDeploymentHistory().Apply(t.Context(), b))
}
