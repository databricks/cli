package config_tests

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentKeySupported(t *testing.T) {
	_, diags := loadTargetWithDiags("./python_wheel/environment_key", "default")
	require.Empty(t, diags)
}

func TestEnvironmentKeyProvidedAndNoPanic(t *testing.T) {
	b, diags := loadTargetWithDiags("./environment_key_only", "default")
	require.Empty(t, diags)

	diags = bundle.Apply(context.Background(), b, libraries.ExpandGlobReferences())
	require.Empty(t, diags)
}
