package ucm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm/deploy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeTfstateForTarget seeds .databricks/ucm/<target>/terraform.tfstate
// under fixtureDir with the resources slice.
func writeTfstateForTarget(t *testing.T, fixtureDir, target string, resources []map[string]any) {
	t.Helper()
	dir := filepath.Join(fixtureDir, filepath.FromSlash(deploy.LocalCacheDir), target)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	blob := map[string]any{
		"version":   4,
		"resources": resources,
	}
	data, err := json.Marshal(blob)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, deploy.TfStateFileName), data, 0o600))
}

func TestCmd_Summary_NoStatePrintsPlaceholder(t *testing.T) {
	stdout, stderr, err := runVerb(t, validFixtureDir(t), "summary")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, "No deployed resources")
}

func TestCmd_Summary_WithStatePrintsCounts(t *testing.T) {
	work := cloneFixture(t, validFixtureDir(t))
	// The valid fixture declares no explicit target, so SelectDefaultTarget
	// chooses the synthesised "default" target. Seed a tfstate there.
	writeTfstateForTarget(t, work, "default", []map[string]any{
		{"type": "databricks_catalog"},
		{"type": "databricks_catalog"},
		{"type": "databricks_schema"},
	})

	stdout, stderr, err := runVerbInDir(t, work, "summary")
	t.Logf("stdout=%q stderr=%q", stdout, stderr)

	require.NoError(t, err)
	assert.Contains(t, stdout, "databricks_catalog")
	assert.Contains(t, stdout, "databricks_schema")
	assert.Contains(t, stdout, "2")
	assert.Contains(t, stdout, "1")
}
