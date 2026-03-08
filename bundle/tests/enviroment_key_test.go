package config_tests

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/libraries"
	"github.com/stretchr/testify/require"
)

func TestEnvironmentKeyProvidedAndNoPanic(t *testing.T) {
	b, diags := loadTargetWithDiags(t, "./environment_key_only", "default")
	require.Empty(t, diags)

	diags = bundle.Apply(t.Context(), b, libraries.ExpandGlobReferences())
	require.Empty(t, diags)
}
