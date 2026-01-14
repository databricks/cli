package annotation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	t.Run("loads valid file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "annotations.yml")
		content := `github.com/databricks/cli/bundle/config.Bundle:
  name:
    description: The bundle name
    since_version: v0.228.0
  cluster_id:
    description: The cluster ID
`
		err := os.WriteFile(path, []byte(content), 0o644)
		require.NoError(t, err)

		file, err := Load(path)
		require.NoError(t, err)

		assert.Equal(t, "The bundle name", file["github.com/databricks/cli/bundle/config.Bundle"]["name"].Description)
		assert.Equal(t, "v0.228.0", file["github.com/databricks/cli/bundle/config.Bundle"]["name"].SinceVersion)
		assert.Equal(t, "The cluster ID", file["github.com/databricks/cli/bundle/config.Bundle"]["cluster_id"].Description)
	})

	t.Run("returns empty file for nonexistent path", func(t *testing.T) {
		file, err := Load("/nonexistent/path/annotations.yml")
		require.NoError(t, err)
		assert.Empty(t, file)
	})

	t.Run("returns empty file for empty content", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "empty.yml")
		err := os.WriteFile(path, []byte(""), 0o644)
		require.NoError(t, err)

		file, err := Load(path)
		require.NoError(t, err)
		assert.Empty(t, file)
	})

	t.Run("returns error for invalid yaml", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "invalid.yml")
		err := os.WriteFile(path, []byte("not: valid: yaml: content"), 0o644)
		require.NoError(t, err)

		_, err = Load(path)
		assert.Error(t, err)
	})
}

func TestSave(t *testing.T) {
	t.Run("saves file with annotations", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "annotations.yml")

		file := File{
			"github.com/databricks/cli/bundle/config.Bundle": {
				"name": Descriptor{
					Description:  "The bundle name",
					SinceVersion: "v0.228.0",
				},
				"cluster_id": Descriptor{
					Description: "The cluster ID",
				},
			},
		}

		err := file.Save(path)
		require.NoError(t, err)

		// Verify by loading it back
		loaded, err := Load(path)
		require.NoError(t, err)

		assert.Equal(t, "The bundle name", loaded["github.com/databricks/cli/bundle/config.Bundle"]["name"].Description)
		assert.Equal(t, "v0.228.0", loaded["github.com/databricks/cli/bundle/config.Bundle"]["name"].SinceVersion)
		assert.Equal(t, "The cluster ID", loaded["github.com/databricks/cli/bundle/config.Bundle"]["cluster_id"].Description)
	})

	t.Run("sorts types alphabetically", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "annotations.yml")

		file := File{
			"z_type": {"field": Descriptor{Description: "z"}},
			"a_type": {"field": Descriptor{Description: "a"}},
			"m_type": {"field": Descriptor{Description: "m"}},
		}

		err := file.Save(path)
		require.NoError(t, err)

		content, err := os.ReadFile(path)
		require.NoError(t, err)

		aIdx := indexOf(string(content), "a_type:")
		mIdx := indexOf(string(content), "m_type:")
		zIdx := indexOf(string(content), "z_type:")

		assert.Less(t, aIdx, mIdx, "a_type should come before m_type")
		assert.Less(t, mIdx, zIdx, "m_type should come before z_type")
	})
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
