package config_tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComplexVariables(t *testing.T) {
	_, diags := loadTargetWithDiags("variables/complex", "default")
	require.Empty(t, diags)
}
