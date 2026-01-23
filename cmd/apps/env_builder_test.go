package apps

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEnvFileBuilder(t *testing.T) {
	t.Run("with valid app.yml", func(t *testing.T) {
		tmpDir := t.TempDir()
		appYml := filepath.Join(tmpDir, "app.yml")
		err := os.WriteFile(appYml, []byte(`command: ["python", "app.py"]
env:
  - name: FOO
    value: bar
  - name: DATABRICKS_WAREHOUSE_ID
    value_from: sql-warehouse
`), 0o644)
		require.NoError(t, err)

		resources := map[string]string{
			"sql-warehouse": "abc123",
		}

		builder, err := NewEnvFileBuilder("https://test.cloud.databricks.com", appYml, resources)
		require.NoError(t, err)
		assert.Equal(t, "https://test.cloud.databricks.com", builder.host)
		assert.Len(t, builder.env, 2)
		assert.Equal(t, "FOO", builder.env[0].Name)
		assert.Equal(t, "bar", builder.env[0].Value)
		assert.Equal(t, "DATABRICKS_WAREHOUSE_ID", builder.env[1].Name)
		assert.Equal(t, "sql-warehouse", builder.env[1].ValueFrom)
	})

	t.Run("with missing app.yml", func(t *testing.T) {
		tmpDir := t.TempDir()
		appYml := filepath.Join(tmpDir, "app.yml")

		builder, err := NewEnvFileBuilder("https://test.cloud.databricks.com", appYml, nil)
		require.NoError(t, err)
		assert.Len(t, builder.env, 0)
	})

	t.Run("with invalid yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		appYml := filepath.Join(tmpDir, "app.yml")
		err := os.WriteFile(appYml, []byte(`invalid: yaml: content:\n  - broken`), 0o644)
		require.NoError(t, err)

		_, err = NewEnvFileBuilder("https://test.cloud.databricks.com", appYml, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
	})

	t.Run("with empty env section", func(t *testing.T) {
		tmpDir := t.TempDir()
		appYml := filepath.Join(tmpDir, "app.yml")
		err := os.WriteFile(appYml, []byte(`command: ["python", "app.py"]
env: []
`), 0o644)
		require.NoError(t, err)

		builder, err := NewEnvFileBuilder("https://test.cloud.databricks.com", appYml, nil)
		require.NoError(t, err)
		assert.Len(t, builder.env, 0)
	})
}

func TestBuild(t *testing.T) {
	t.Run("with direct values", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host: "https://test.cloud.databricks.com",
			env: []EnvVar{
				{Name: "FOO", Value: "bar"},
				{Name: "BAZ", Value: "qux"},
			},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, content, "DATABRICKS_HOST=https://test.cloud.databricks.com")
		assert.Contains(t, content, "FOO=bar")
		assert.Contains(t, content, "BAZ=qux")
		assert.Contains(t, content, "# Environment variables from app.yml")
	})

	t.Run("with value_from references", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host: "https://test.cloud.databricks.com",
			env: []EnvVar{
				{Name: "DATABRICKS_WAREHOUSE_ID", ValueFrom: "sql-warehouse"},
				{Name: "SERVING_ENDPOINT_NAME", ValueFrom: "serving-endpoint"},
			},
			resources: map[string]string{
				"sql-warehouse":     "abc123",
				"serving-endpoint":  "my-endpoint",
			},
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, content, "DATABRICKS_WAREHOUSE_ID=abc123")
		assert.Contains(t, content, "SERVING_ENDPOINT_NAME=my-endpoint")
	})

	t.Run("with mixed value and value_from", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host: "https://test.cloud.databricks.com",
			env: []EnvVar{
				{Name: "STATIC_VAR", Value: "static-value"},
				{Name: "DATABRICKS_WAREHOUSE_ID", ValueFrom: "sql-warehouse"},
				{Name: "ANOTHER_STATIC", Value: "another-value"},
			},
			resources: map[string]string{
				"sql-warehouse": "xyz789",
			},
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, content, "STATIC_VAR=static-value")
		assert.Contains(t, content, "DATABRICKS_WAREHOUSE_ID=xyz789")
		assert.Contains(t, content, "ANOTHER_STATIC=another-value")
	})

	t.Run("with missing resource reference", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host: "https://test.cloud.databricks.com",
			env: []EnvVar{
				{Name: "MISSING_REF", ValueFrom: "nonexistent-resource"},
			},
			resources: map[string]string{},
		}

		_, err := builder.Build()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resource reference \"nonexistent-resource\" not found")
		assert.Contains(t, err.Error(), "MISSING_REF")
	})

	t.Run("with empty value", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host: "https://test.cloud.databricks.com",
			env: []EnvVar{
				{Name: "EMPTY_VAR", Value: ""},
			},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, content, "EMPTY_VAR=")
	})

	t.Run("with no env vars", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:      "https://test.cloud.databricks.com",
			env:       []EnvVar{},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.Equal(t, "", content)
	})

	t.Run("skips empty names", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host: "https://test.cloud.databricks.com",
			env: []EnvVar{
				{Name: "", Value: "should-be-skipped"},
				{Name: "VALID", Value: "valid-value"},
			},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.NotContains(t, content, "should-be-skipped")
		assert.Contains(t, content, "VALID=valid-value")
	})

	t.Run("does not add DATABRICKS_HOST if already present", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host: "https://test.cloud.databricks.com",
			env: []EnvVar{
				{Name: "DATABRICKS_HOST", Value: "https://custom.databricks.com"},
				{Name: "OTHER_VAR", Value: "other-value"},
			},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		// Should only contain the custom host, not the builder's host
		assert.Contains(t, content, "DATABRICKS_HOST=https://custom.databricks.com")
		// Count occurrences - should be exactly 1
		count := strings.Count(content, "DATABRICKS_HOST=")
		assert.Equal(t, 1, count)
	})

	t.Run("with no host and no env vars", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:      "",
			env:       []EnvVar{},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.Equal(t, "", content)
	})
}

func TestWriteEnvFile(t *testing.T) {
	t.Run("creates .env file", func(t *testing.T) {
		tmpDir := t.TempDir()
		builder := &EnvFileBuilder{
			host: "https://test.cloud.databricks.com",
			env: []EnvVar{
				{Name: "FOO", Value: "bar"},
			},
			resources: nil,
		}

		err := builder.WriteEnvFile(tmpDir)
		require.NoError(t, err)

		envPath := filepath.Join(tmpDir, ".env")
		content, err := os.ReadFile(envPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "FOO=bar")
		assert.Contains(t, string(content), "DATABRICKS_HOST=https://test.cloud.databricks.com")
	})

	t.Run("does not create file when no env vars", func(t *testing.T) {
		tmpDir := t.TempDir()
		builder := &EnvFileBuilder{
			host:      "https://test.cloud.databricks.com",
			env:       []EnvVar{},
			resources: nil,
		}

		err := builder.WriteEnvFile(tmpDir)
		require.NoError(t, err)

		envPath := filepath.Join(tmpDir, ".env")
		_, err = os.Stat(envPath)
		assert.True(t, os.IsNotExist(err))
	})

	t.Run("overwrites existing .env file", func(t *testing.T) {
		tmpDir := t.TempDir()
		envPath := filepath.Join(tmpDir, ".env")

		// Create existing .env
		err := os.WriteFile(envPath, []byte("OLD_VAR=old-value\n"), 0o644)
		require.NoError(t, err)

		builder := &EnvFileBuilder{
			host: "https://test.cloud.databricks.com",
			env: []EnvVar{
				{Name: "NEW_VAR", Value: "new-value"},
			},
			resources: nil,
		}

		err = builder.WriteEnvFile(tmpDir)
		require.NoError(t, err)

		content, err := os.ReadFile(envPath)
		require.NoError(t, err)
		assert.NotContains(t, string(content), "OLD_VAR")
		assert.Contains(t, string(content), "NEW_VAR=new-value")
	})

	t.Run("fails with missing resource reference", func(t *testing.T) {
		tmpDir := t.TempDir()
		builder := &EnvFileBuilder{
			host: "https://test.cloud.databricks.com",
			env: []EnvVar{
				{Name: "BAD_REF", ValueFrom: "missing-resource"},
			},
			resources: map[string]string{},
		}

		err := builder.WriteEnvFile(tmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resource reference \"missing-resource\" not found")
	})
}

func TestEnvFileBuilderIntegration(t *testing.T) {
	t.Run("end-to-end workflow", func(t *testing.T) {
		// Setup: Create a temporary directory with app.yml
		tmpDir := t.TempDir()
		appYml := filepath.Join(tmpDir, "app.yml")
		appYmlContent := `command: ["python", "app.py"]
env:
  - name: DATABRICKS_WAREHOUSE_ID
    value_from: sql-warehouse
  - name: MLFLOW_EXPERIMENT_ID
    value_from: experiment
  - name: SERVING_ENDPOINT_NAME
    value_from: serving-endpoint
  - name: APP_NAME
    value: my-test-app
  - name: DEBUG
    value: "true"
`
		err := os.WriteFile(appYml, []byte(appYmlContent), 0o644)
		require.NoError(t, err)

		// Create resource map with names matching databricks.yml resource.name fields
		resources := map[string]string{
			"sql-warehouse":     "abc123xyz",
			"experiment":        "exp-456",
			"serving-endpoint":  "my-endpoint",
		}

		// Build and write .env
		builder, err := NewEnvFileBuilder("https://my-workspace.cloud.databricks.com", appYml, resources)
		require.NoError(t, err)

		err = builder.WriteEnvFile(tmpDir)
		require.NoError(t, err)

		// Verify the generated .env file
		envPath := filepath.Join(tmpDir, ".env")
		content, err := os.ReadFile(envPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "DATABRICKS_HOST=https://my-workspace.cloud.databricks.com")
		assert.Contains(t, contentStr, "DATABRICKS_WAREHOUSE_ID=abc123xyz")
		assert.Contains(t, contentStr, "MLFLOW_EXPERIMENT_ID=exp-456")
		assert.Contains(t, contentStr, "SERVING_ENDPOINT_NAME=my-endpoint")
		assert.Contains(t, contentStr, "APP_NAME=my-test-app")
		assert.Contains(t, contentStr, "DEBUG=true")
		assert.Contains(t, contentStr, "# Environment variables from app.yml")
	})
}
