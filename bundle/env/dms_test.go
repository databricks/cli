package env

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestDMS(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"1", true},
		{"", false},
		{"0", false},
		{"false", false},
	} {
		ctx := env.Set(t.Context(), DMSVariable, tc.value)
		assert.Equal(t, tc.want, DMS(ctx), "value %q", tc.value)
	}
}

func TestDMSUnset(t *testing.T) {
	assert.False(t, DMS(t.Context()))
}

func TestRecordsDeploymentHistory(t *testing.T) {
	// The bundle setting alone is enough, and so is the environment; the env var
	// exists so the acceptance suite can record without touching every databricks.yml.
	assert.True(t, RecordsDeploymentHistory(t.Context(), true))
	assert.False(t, RecordsDeploymentHistory(t.Context(), false))

	ctx := env.Set(t.Context(), DMSVariable, "true")
	assert.True(t, RecordsDeploymentHistory(ctx, false))
}
