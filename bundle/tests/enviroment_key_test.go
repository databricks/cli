package config_tests

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnvironmentKeySupported(t *testing.T) {
	_, diags := loadTargetWithDiags("./environment_key", "default")
	require.Empty(t, diags)
}
