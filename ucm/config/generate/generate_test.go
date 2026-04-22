package generate_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/generate"
	"github.com/databricks/cli/ucm/config/resources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSaveToFileEmitsNameAndResources(t *testing.T) {
	r := &config.Root{
		Ucm: config.Ucm{Name: "scanned-prod"},
		Workspace: config.Workspace{
			Host: "https://example.cloud.databricks.com",
		},
		Resources: config.Resources{
			Catalogs: map[string]*resources.Catalog{
				"team_alpha": {Name: "team_alpha", Comment: "alpha"},
			},
		},
	}

	dir := t.TempDir()
	out := filepath.Join(dir, "ucm.yml")
	require.NoError(t, generate.SaveToFile(r, out, false))

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	contents := string(data)

	assert.Contains(t, contents, "name: scanned-prod")
	assert.Contains(t, contents, "host:")
	assert.Contains(t, contents, "team_alpha")
	assert.Contains(t, contents, "alpha")
}

func TestSaveToFileRefusesToOverwriteWithoutForce(t *testing.T) {
	r := &config.Root{Ucm: config.Ucm{Name: "n"}}
	dir := t.TempDir()
	out := filepath.Join(dir, "ucm.yml")
	require.NoError(t, os.WriteFile(out, []byte("existing"), 0o644))

	err := generate.SaveToFile(r, out, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestSaveToFileOverwritesWithForce(t *testing.T) {
	r := &config.Root{Ucm: config.Ucm{Name: "fresh"}}
	dir := t.TempDir()
	out := filepath.Join(dir, "ucm.yml")
	require.NoError(t, os.WriteFile(out, []byte("existing"), 0o644))

	require.NoError(t, generate.SaveToFile(r, out, true))

	data, err := os.ReadFile(out)
	require.NoError(t, err)
	assert.Contains(t, string(data), "fresh")
	assert.NotContains(t, string(data), "existing")
}

func TestSaveToFileRejectsNilRoot(t *testing.T) {
	err := generate.SaveToFile(nil, filepath.Join(t.TempDir(), "ucm.yml"), false)
	require.Error(t, err)
}
