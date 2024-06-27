package config_tests

import (
	"github.com/databricks/cli/cmd/root"
	assert "github.com/databricks/cli/libs/dyn/dynassert"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal"
)

func TestSuggestTargetIfWrongPassed(t *testing.T) {
	t.Setenv("BUNDLE_ROOT", filepath.Join("target_overrides", "workspace"))
	stdoutBytes, _, err := internal.RequireErrorRun(t, "bundle", "validate", "-e", "incorrect")
	stdout := stdoutBytes.String()

	assert.Error(t, root.ErrAlreadyPrinted, err)
	assert.Contains(t, stdout, "Available targets:")
	assert.Contains(t, stdout, "development")
	assert.Contains(t, stdout, "staging")
}
