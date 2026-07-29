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
		name       string
		enabled    bool
		forceAllow string
		wantError  bool
	}{
		{name: "flag unset", enabled: false, wantError: false},
		{name: "flag set", enabled: true, wantError: true},
		{name: "flag set with force allow", enabled: true, forceAllow: "1", wantError: false},
		{name: "flag set with empty force allow", enabled: true, forceAllow: "", wantError: true},
		{name: "flag unset with force allow", enabled: false, forceAllow: "1", wantError: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			b := &bundle.Bundle{
				Config: config.Root{
					Experimental: &config.Experimental{RecordDeploymentHistory: tc.enabled},
				},
			}

			ctx := env.Set(t.Context(), bundleenv.ForceAllowRecordDeploymentHistoryVariable, tc.forceAllow)
			diags := ValidateRecordDeploymentHistory().Apply(ctx, b)

			if !tc.wantError {
				assert.Empty(t, diags)
				return
			}
			require.Len(t, diags, 1)
			assert.Equal(t, diag.Error, diags[0].Severity)
			assert.Equal(t, "experimental.record_deployment_history is not supported yet", diags[0].Summary)
			assert.Equal(t, recordDeploymentHistoryPath, diags[0].Paths[0].String())
		})
	}
}

func TestValidateRecordDeploymentHistoryNoExperimentalBlock(t *testing.T) {
	b := &bundle.Bundle{Config: config.Root{}}
	assert.Empty(t, ValidateRecordDeploymentHistory().Apply(t.Context(), b))
}
