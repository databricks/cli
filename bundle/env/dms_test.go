package env

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestRecordDeploymentHistoryEnv(t *testing.T) {
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
		ctx := env.Set(t.Context(), RecordDeploymentHistoryVariable, tc.value)
		assert.Equal(t, tc.want, recordDeploymentHistoryEnv(ctx), "value %q", tc.value)
	}
}

func TestRecordDeploymentHistoryEnvUnset(t *testing.T) {
	assert.False(t, recordDeploymentHistoryEnv(t.Context()))
}

func TestRecordsDeploymentHistory(t *testing.T) {
	// The bundle setting alone is enough, and so is the environment; the env var
	// exists so the acceptance suite can record without touching every databricks.yml.
	assert.True(t, RecordsDeploymentHistory(t.Context(), true))
	assert.False(t, RecordsDeploymentHistory(t.Context(), false))

	ctx := env.Set(t.Context(), RecordDeploymentHistoryVariable, "true")
	assert.True(t, RecordsDeploymentHistory(ctx, false))
}
