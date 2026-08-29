package aircmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(`
[__settings__]
default_profile = second

[first]
host = https://first.test
databricks_internal_user_workspace_dir = /Workspace/first

[second]
host = https://second.test
databricks_internal_user_workspace_dir = /Workspace/second
`), 0o600))

	value, err := profileValue(&config.Config{ConfigFile: path, Profile: "first"}, "databricks_internal_user_workspace_dir")
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/first", value)

	value, err = profileValue(&config.Config{ConfigFile: path}, "databricks_internal_user_workspace_dir")
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/second", value)

	value, err = profileValue(&config.Config{ConfigFile: path, Profile: "missing"}, "databricks_internal_user_workspace_dir")
	require.NoError(t, err)
	assert.Empty(t, value)

	value, err = profileValue(&config.Config{ConfigFile: path, Profile: "first"}, "missing")
	require.NoError(t, err)
	assert.Empty(t, value)
}

func TestProfileValueSingleProfileDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".databrickscfg")
	require.NoError(t, os.WriteFile(path, []byte(`
[only]
host = https://only.test
databricks_internal_user_workspace_dir = /Workspace/only
`), 0o600))

	value, err := profileValue(&config.Config{ConfigFile: path}, "databricks_internal_user_workspace_dir")
	require.NoError(t, err)
	assert.Equal(t, "/Workspace/only", value)
}
