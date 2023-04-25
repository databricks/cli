package mutator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVariablesParseBundleVar(t *testing.T) {
	s := `BUNDLE_VAR_FOO=Bar`
	key, val, err := parseBundleVar(s)
	assert.NoError(t, err)
	assert.Equal(t, "foo", key)
	assert.Equal(t, "Bar", val)
}

func TestVariablesParseBundleVarOnInvalidPrefix(t *testing.T) {
	s := `INVALID_PREFIX_FOO=Bar`
	_, _, err := parseBundleVar(s)
	assert.ErrorContains(t, err, "environment variable INVALID_PREFIX_FOO=Bar does not have expected prefix BUNDLE_VAR_")
}

func TestVariablesParseBundleVarOnInvalidFormat(t *testing.T) {
	s := `BUNDLE_VAR_FOO=Bar=bar`
	_, _, err := parseBundleVar(s)
	assert.ErrorContains(t, err, "unexpected format for environment variable: BUNDLE_VAR_FOO=Bar=bar")
}
