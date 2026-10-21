package env

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
)

func TestStrictProfileMatch(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  bool
	}{
		{"1", true},
		{"true", true},
		{"yes", true},
		{"0", false},
		{"false", false},
		{"no", false},
		{"", false},
	} {
		ctx := env.Set(t.Context(), StrictProfileMatchVariable, tc.value)
		assert.Equal(t, tc.want, StrictProfileMatch(ctx), "value %q", tc.value)
	}
}

func TestStrictProfileMatchUnset(t *testing.T) {
	assert.False(t, StrictProfileMatch(t.Context()))
}
