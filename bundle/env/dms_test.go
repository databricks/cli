package env

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentHistoryEnv(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"", false},
		{"0", false},
		{"false", false},
		// Only "true" counts, so a near miss leaves recording off.
		{"1", false},
		{"TRUE", false},
		{"yes", false},
	} {
		ctx := env.Set(t.Context(), DeploymentHistoryVariable, tc.value)
		assert.Equal(t, tc.want, RecordsDeploymentHistory(ctx, nil), "value %q", tc.value)
	}
}

func TestDeploymentHistoryEnvUnset(t *testing.T) {
	assert.False(t, RecordsDeploymentHistory(t.Context(), nil))
}

func TestRecordsDeploymentHistory(t *testing.T) {
	for _, tc := range []struct {
		name       string
		configured *bool
		value      string
		want       bool
	}{
		{"config true overrides env false", new(true), "false", true},
		{"config false overrides env true", new(false), "true", false},
		{"config true without env", new(true), "", true},
		{"config false without env", new(false), "", false},
		{"unset config uses env true", nil, "true", true},
		{"unset config uses env false", nil, "false", false},
		{"unset config and env", nil, "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), DeploymentHistoryVariable, tc.value)
			assert.Equal(t, tc.want, RecordsDeploymentHistory(ctx, tc.configured))
		})
	}
}
