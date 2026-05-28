package env

import (
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/assert"
)

func TestIsManagedState(t *testing.T) {
	cases := []struct {
		value string
		want  bool
	}{
		{"true", true},
		{"TRUE", true},
		{"1", true},
		{"yes", true},
		{"on", true},
		{"false", false},
		{"0", false},
		{"no", false},
		{"off", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			testutil.CleanupEnvironment(t)
			if tc.value != "" {
				t.Setenv(managedStateVariable, tc.value)
			}
			assert.Equal(t, tc.want, IsManagedState(t.Context()))
		})
	}
}

func TestIsManagedStateUnset(t *testing.T) {
	testutil.CleanupEnvironment(t)
	assert.False(t, IsManagedState(t.Context()))
}

func TestIsManagedStateGarbageValueFallsBackToFalse(t *testing.T) {
	testutil.CleanupEnvironment(t)
	t.Setenv(managedStateVariable, "garbage")
	assert.False(t, IsManagedState(t.Context()))
}
