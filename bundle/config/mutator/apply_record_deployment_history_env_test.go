package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyRecordDeploymentHistoryEnv(t *testing.T) {
	tests := []struct {
		name         string
		envValue     string
		envSet       bool
		experimental *config.Experimental
		want         bool
	}{
		{
			name:   "env not set leaves config untouched",
			envSet: false,
			want:   false,
		},
		{
			name:     "env set enables the experimental setting",
			envValue: "true",
			envSet:   true,
			want:     true,
		},
		{
			name:         "env set enables it even when experimental is already present",
			envValue:     "true",
			envSet:       true,
			experimental: &config.Experimental{PythonWheelWrapper: true},
			want:         true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx := t.Context()
			if tc.envSet {
				ctx = env.Set(ctx, "DATABRICKS_BUNDLE_RECORD_DEPLOYMENT_HISTORY", tc.envValue)
			}

			b := &bundle.Bundle{
				Config: config.Root{
					Experimental: tc.experimental,
				},
			}

			diags := bundle.Apply(ctx, b, mutator.ApplyRecordDeploymentHistoryEnv())
			require.NoError(t, diags.Error())

			if tc.want {
				require.NotNil(t, b.Config.Experimental)
				assert.True(t, b.Config.Experimental.RecordDeploymentHistory)
			} else {
				// When the env var is unset the mutator must not allocate Experimental.
				assert.Nil(t, b.Config.Experimental)
			}
		})
	}
}
