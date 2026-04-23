package loader_test

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config"
	"github.com/databricks/cli/ucm/config/loader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessIncludeMerges(t *testing.T) {
	root := t.TempDir()
	testutil.WriteFile(t, filepath.Join(root, "ucm.yml"), "ucm: {name: main}\n")
	testutil.WriteFile(t, filepath.Join(root, "extra.yml"), `
workspace:
  host: https://bar.cloud.databricks.com
`)

	cfg, diags := config.Load(filepath.Join(root, "ucm.yml"))
	require.NoError(t, diags.Error())
	u := &ucm.Ucm{RootPath: root, Config: *cfg}

	m := loader.ProcessInclude(filepath.Join(root, "extra.yml"), "extra.yml")
	assert.Equal(t, "ProcessInclude(extra.yml)", m.Name())

	out := ucm.Apply(t.Context(), u, m)
	require.NoError(t, out.Error())
	assert.Equal(t, "https://bar.cloud.databricks.com", u.Config.Workspace.Host)
	assert.Equal(t, "main", u.Config.Ucm.Name)
}

func TestProcessIncludeWarnsOnNestedInclude(t *testing.T) {
	root := t.TempDir()
	testutil.WriteFile(t, filepath.Join(root, "ucm.yml"), "ucm: {name: main}\n")
	testutil.WriteFile(t, filepath.Join(root, "extra.yml"), `
include:
  - other.yml
`)

	cfg, diags := config.Load(filepath.Join(root, "ucm.yml"))
	require.NoError(t, diags.Error())
	u := &ucm.Ucm{RootPath: root, Config: *cfg}

	out := ucm.Apply(t.Context(), u, loader.ProcessInclude(filepath.Join(root, "extra.yml"), "extra.yml"))
	require.NoError(t, out.Error())
	require.Len(t, out, 1)
	assert.Contains(t, out[0].Summary, "Include section is defined outside root file")
}
