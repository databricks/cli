package config_tests

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/stretchr/testify/require"
)

func TestSuggestTargetIfWrongPassed(t *testing.T) {
	t.Setenv("BUNDLE_ROOT", filepath.Join("target_overrides", "workspace"))
	_, _, err := internal.RequireErrorRun(t, "bundle", "validate", "-e", "incorrect")
	require.ErrorContains(t, err, "Available targets: development, staging")
}
