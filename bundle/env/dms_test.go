package env

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestDeploymentHistoryEnv(t *testing.T) {
	for _, tc := range []struct {
		value        string
		wantEnabled  bool
		wantExplicit bool
	}{
		{"true", true, true},
		{"false", false, true},
		// Only "true" turns recording on, so a near miss turns it off.
		{"0", false, true},
		{"1", false, true},
		{"TRUE", false, true},
		{"yes", false, true},
		// Empty is no choice, like unset.
		{"", false, false},
	} {
		ctx := env.Set(t.Context(), DeploymentHistoryVariable, tc.value)
		enabled, explicit := DeploymentHistoryChoice(ctx, nil)
		assert.Equal(t, tc.wantEnabled, enabled, "value %q", tc.value)
		assert.Equal(t, tc.wantExplicit, explicit, "value %q", tc.value)
	}
}

func TestDeploymentHistoryEnvUnset(t *testing.T) {
	enabled, explicit := DeploymentHistoryChoice(t.Context(), nil)
	assert.False(t, enabled)
	assert.False(t, explicit)
}

func TestDeploymentHistoryChoice(t *testing.T) {
	for _, tc := range []struct {
		name         string
		configured   *bool
		value        string
		wantEnabled  bool
		wantExplicit bool
	}{
		{"config true env true", new(true), "true", true, true},
		{"config true env false", new(true), "false", true, true},
		{"config true env unset", new(true), "", true, true},
		{"config false env true", new(false), "true", false, true},
		{"config false env false", new(false), "false", false, true},
		{"config false env unset", new(false), "", false, true},
		{"config unset env true", nil, "true", true, true},
		{"config unset env false", nil, "false", false, true},
		{"config unset env unset", nil, "", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), DeploymentHistoryVariable, tc.value)
			enabled, explicit := DeploymentHistoryChoice(ctx, tc.configured)
			assert.Equal(t, tc.wantEnabled, enabled)
			assert.Equal(t, tc.wantExplicit, explicit)
		})
	}
}
