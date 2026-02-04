package legacytemplates

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

		builder, err := NewEnvFileBuilder("https://test.cloud.databricks.com", "DEFAULT", "test-app", appYml, resources)
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

		builder, err := NewEnvFileBuilder("https://test.cloud.databricks.com", "DEFAULT", "test-app", appYml, nil)
		require.NoError(t, err)
		assert.Len(t, builder.env, 0)
	})

	t.Run("with invalid yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		appYml := filepath.Join(tmpDir, "app.yml")
		err := os.WriteFile(appYml, []byte(`invalid: yaml: content:\n  - broken`), 0o644)
		require.NoError(t, err)

		_, err = NewEnvFileBuilder("https://test.cloud.databricks.com", "DEFAULT", "test-app", appYml, nil)
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

		builder, err := NewEnvFileBuilder("https://test.cloud.databricks.com", "DEFAULT", "test-app", appYml, nil)
		require.NoError(t, err)
		assert.Len(t, builder.env, 0)
	})

	t.Run("with camelCase valueFrom (legacy template format)", func(t *testing.T) {
		tmpDir := t.TempDir()
		appYml := filepath.Join(tmpDir, "app.yml")
		err := os.WriteFile(appYml, []byte(`command: ["python", "app.py"]
env:
  - name: DATABRICKS_WAREHOUSE_ID
    valueFrom: sql-warehouse
  - name: STATIC_VAR
    value: static-value
`), 0o644)
		require.NoError(t, err)

		resources := map[string]string{
			"sql-warehouse": "abc123",
		}

		builder, err := NewEnvFileBuilder("https://test.cloud.databricks.com", "DEFAULT", "test-app", appYml, resources)
		require.NoError(t, err)
		assert.Len(t, builder.env, 2)
		assert.Equal(t, "DATABRICKS_WAREHOUSE_ID", builder.env[0].Name)
		assert.Equal(t, "sql-warehouse", builder.env[0].ValueFrom)
		assert.Equal(t, "STATIC_VAR", builder.env[1].Name)
		assert.Equal(t, "static-value", builder.env[1].Value)
	})
}

func TestBuild(t *testing.T) {
	t.Run("with direct values", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:    "https://test.cloud.databricks.com",
			profile: "DEFAULT",
			appName: "test-app",
			env: []EnvVar{
				{Name: "FOO", Value: "bar"},
				{Name: "BAZ", Value: "qux"},
			},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, content, "DATABRICKS_HOST=https://test.cloud.databricks.com")
		assert.Contains(t, content, "DATABRICKS_APP_NAME=test-app")
		assert.Contains(t, content, "MLFLOW_TRACKING_URI=databricks")
		assert.Contains(t, content, "FOO=bar")
		assert.Contains(t, content, "BAZ=qux")
	})

	t.Run("with value_from references", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:    "https://test.cloud.databricks.com",
			profile: "DEFAULT",
			appName: "test-app",
			env: []EnvVar{
				{Name: "DATABRICKS_WAREHOUSE_ID", ValueFrom: "sql-warehouse"},
				{Name: "SERVING_ENDPOINT_NAME", ValueFrom: "serving-endpoint"},
			},
			resources: map[string]string{
				"sql-warehouse":    "abc123",
				"serving-endpoint": "my-endpoint",
			},
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, content, "MLFLOW_TRACKING_URI=databricks")
		assert.Contains(t, content, "DATABRICKS_WAREHOUSE_ID=abc123")
		assert.Contains(t, content, "SERVING_ENDPOINT_NAME=my-endpoint")
	})

	t.Run("with mixed value and value_from", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:    "https://test.cloud.databricks.com",
			profile: "DEFAULT",
			appName: "test-app",
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
		assert.Contains(t, content, "MLFLOW_TRACKING_URI=databricks")
		assert.Contains(t, content, "STATIC_VAR=static-value")
		assert.Contains(t, content, "DATABRICKS_WAREHOUSE_ID=xyz789")
		assert.Contains(t, content, "ANOTHER_STATIC=another-value")
	})

	t.Run("with missing resource reference", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:    "https://test.cloud.databricks.com",
			profile: "DEFAULT",
			appName: "test-app",
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
			host:    "https://test.cloud.databricks.com",
			profile: "DEFAULT",
			appName: "test-app",
			env: []EnvVar{
				{Name: "EMPTY_VAR", Value: ""},
			},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, content, "MLFLOW_TRACKING_URI=databricks")
		assert.Contains(t, content, "EMPTY_VAR=")
	})

	t.Run("with no env vars", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:      "https://test.cloud.databricks.com",
			profile:   "DEFAULT",
			env:       []EnvVar{},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		// Should still have DATABRICKS_HOST and MLFLOW_TRACKING_URI
		assert.Contains(t, content, "DATABRICKS_HOST=https://test.cloud.databricks.com")
		assert.Contains(t, content, "MLFLOW_TRACKING_URI=databricks")
	})

	t.Run("skips empty names", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:    "https://test.cloud.databricks.com",
			profile: "DEFAULT",
			appName: "test-app",
			env: []EnvVar{
				{Name: "", Value: "should-be-skipped"},
				{Name: "VALID", Value: "valid-value"},
			},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, content, "MLFLOW_TRACKING_URI=databricks")
		assert.NotContains(t, content, "should-be-skipped")
		assert.Contains(t, content, "VALID=valid-value")
	})

	t.Run("does not add DATABRICKS_HOST if already present", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:    "https://test.cloud.databricks.com",
			profile: "DEFAULT",
			appName: "test-app",
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
		assert.Contains(t, content, "MLFLOW_TRACKING_URI=databricks")
		// Count occurrences - should be exactly 1
		count := strings.Count(content, "DATABRICKS_HOST=")
		assert.Equal(t, 1, count)
	})

	t.Run("with no host and no env vars but has profile", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:      "",
			profile:   "my-profile",
			env:       []EnvVar{},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		// Should still have MLFLOW_TRACKING_URI
		assert.Contains(t, content, "MLFLOW_TRACKING_URI=databricks://my-profile")
	})
}

func TestWriteEnvFile(t *testing.T) {
	t.Run("creates .env file", func(t *testing.T) {
		tmpDir := t.TempDir()
		builder := &EnvFileBuilder{
			host:    "https://test.cloud.databricks.com",
			profile: "DEFAULT",
			appName: "test-app",
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
		assert.Contains(t, string(content), "DATABRICKS_HOST=https://test.cloud.databricks.com")
		assert.Contains(t, string(content), "MLFLOW_TRACKING_URI=databricks")
		assert.Contains(t, string(content), "FOO=bar")
	})

	t.Run("creates file even when no env vars from app.yml", func(t *testing.T) {
		tmpDir := t.TempDir()
		builder := &EnvFileBuilder{
			host:      "https://test.cloud.databricks.com",
			profile:   "DEFAULT",
			env:       []EnvVar{},
			resources: nil,
		}

		err := builder.WriteEnvFile(tmpDir)
		require.NoError(t, err)

		envPath := filepath.Join(tmpDir, ".env")
		content, err := os.ReadFile(envPath)
		require.NoError(t, err)
		// Should still have DATABRICKS_HOST and MLFLOW_TRACKING_URI
		assert.Contains(t, string(content), "DATABRICKS_HOST=https://test.cloud.databricks.com")
		assert.Contains(t, string(content), "MLFLOW_TRACKING_URI=databricks")
	})

	t.Run("overwrites existing .env file", func(t *testing.T) {
		tmpDir := t.TempDir()
		envPath := filepath.Join(tmpDir, ".env")

		// Create existing .env
		err := os.WriteFile(envPath, []byte("OLD_VAR=old-value\n"), 0o644)
		require.NoError(t, err)

		builder := &EnvFileBuilder{
			host:    "https://test.cloud.databricks.com",
			profile: "DEFAULT",
			appName: "test-app",
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
		assert.Contains(t, string(content), "MLFLOW_TRACKING_URI=databricks")
		assert.Contains(t, string(content), "NEW_VAR=new-value")
	})

	t.Run("fails with missing resource reference", func(t *testing.T) {
		tmpDir := t.TempDir()
		builder := &EnvFileBuilder{
			host:    "https://test.cloud.databricks.com",
			profile: "DEFAULT",
			appName: "test-app",
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

func TestBuildMLflowTrackingURI(t *testing.T) {
	t.Run("overrides MLFLOW_TRACKING_URI from app.yml with default profile", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:    "https://test.cloud.databricks.com",
			profile: "DEFAULT",
			appName: "test-app",
			env: []EnvVar{
				{Name: "MLFLOW_TRACKING_URI", Value: "databricks"},
				{Name: "OTHER_VAR", Value: "other-value"},
			},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		// Should only contain one MLFLOW_TRACKING_URI and it should be databricks
		assert.Contains(t, content, "MLFLOW_TRACKING_URI=databricks")
		count := strings.Count(content, "MLFLOW_TRACKING_URI=")
		assert.Equal(t, 1, count, "Should have exactly one MLFLOW_TRACKING_URI")
	})

	t.Run("overrides MLFLOW_TRACKING_URI from app.yml with named profile", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:    "https://test.cloud.databricks.com",
			profile: "my-profile",
			env: []EnvVar{
				{Name: "MLFLOW_TRACKING_URI", Value: "databricks"},
				{Name: "OTHER_VAR", Value: "other-value"},
			},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		// Should override with databricks://my-profile
		assert.Contains(t, content, "MLFLOW_TRACKING_URI=databricks://my-profile")
		count := strings.Count(content, "MLFLOW_TRACKING_URI=")
		assert.Equal(t, 1, count, "Should have exactly one MLFLOW_TRACKING_URI")
	})

	t.Run("uses databricks for empty profile", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:    "https://test.cloud.databricks.com",
			profile: "",
			env: []EnvVar{
				{Name: "OTHER_VAR", Value: "other-value"},
			},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, content, "MLFLOW_TRACKING_URI=databricks")
	})

	t.Run("uses databricks for DEFAULT profile", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:      "https://test.cloud.databricks.com",
			profile:   "DEFAULT",
			env:       []EnvVar{},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, content, "MLFLOW_TRACKING_URI=databricks")
		assert.NotContains(t, content, "databricks://")
	})

	t.Run("uses databricks://<profile> for named profile", func(t *testing.T) {
		builder := &EnvFileBuilder{
			host:      "https://test.cloud.databricks.com",
			profile:   "my-workspace-profile",
			env:       []EnvVar{},
			resources: nil,
		}

		content, err := builder.Build()
		require.NoError(t, err)
		assert.Contains(t, content, "MLFLOW_TRACKING_URI=databricks://my-workspace-profile")
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
			"sql-warehouse":    "abc123xyz",
			"experiment":       "exp-456",
			"serving-endpoint": "my-endpoint",
		}

		// Build and write .env
		builder, err := NewEnvFileBuilder("https://my-workspace.cloud.databricks.com", "DEFAULT", "my-test-app", appYml, resources)
		require.NoError(t, err)

		err = builder.WriteEnvFile(tmpDir)
		require.NoError(t, err)

		// Verify the generated .env file
		envPath := filepath.Join(tmpDir, ".env")
		content, err := os.ReadFile(envPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "DATABRICKS_HOST=https://my-workspace.cloud.databricks.com")
		assert.Contains(t, contentStr, "DATABRICKS_APP_NAME=my-test-app")
		assert.Contains(t, contentStr, "MLFLOW_TRACKING_URI=databricks")
		assert.Contains(t, contentStr, "DATABRICKS_WAREHOUSE_ID=abc123xyz")
		assert.Contains(t, contentStr, "MLFLOW_EXPERIMENT_ID=exp-456")
		assert.Contains(t, contentStr, "SERVING_ENDPOINT_NAME=my-endpoint")
		assert.Contains(t, contentStr, "APP_NAME=my-test-app")
		assert.Contains(t, contentStr, "DEBUG=true")
	})

	t.Run("end-to-end with camelCase (legacy template format)", func(t *testing.T) {
		// Setup: Create a temporary directory with app.yml using camelCase
		tmpDir := t.TempDir()
		appYml := filepath.Join(tmpDir, "app.yml")
		appYmlContent := `command: ["uv", "run", "start-app"]
env:
  - name: MLFLOW_TRACKING_URI
    value: "databricks"
  - name: MLFLOW_REGISTRY_URI
    value: "databricks-uc"
  - name: MLFLOW_EXPERIMENT_ID
    valueFrom: experiment
  - name: API_PROXY
    value: "http://localhost:8000/invocations"
  - name: CHAT_APP_PORT
    value: "3000"
`
		err := os.WriteFile(appYml, []byte(appYmlContent), 0o644)
		require.NoError(t, err)

		// Create resource map
		resources := map[string]string{
			"experiment": "77131241535601",
		}

		// Build and write .env
		builder, err := NewEnvFileBuilder("https://adb-1966697730403610.10.azuredatabricks.net", "DEFAULT", "app3", appYml, resources)
		require.NoError(t, err)

		err = builder.WriteEnvFile(tmpDir)
		require.NoError(t, err)

		// Verify the generated .env file
		envPath := filepath.Join(tmpDir, ".env")
		content, err := os.ReadFile(envPath)
		require.NoError(t, err)

		contentStr := string(content)
		assert.Contains(t, contentStr, "DATABRICKS_HOST=https://adb-1966697730403610.10.azuredatabricks.net")
		assert.Contains(t, contentStr, "DATABRICKS_APP_NAME=app3")
		assert.Contains(t, contentStr, "MLFLOW_TRACKING_URI=databricks")
		assert.Contains(t, contentStr, "MLFLOW_REGISTRY_URI=databricks-uc")
		assert.Contains(t, contentStr, "MLFLOW_EXPERIMENT_ID=77131241535601")
		assert.Contains(t, contentStr, "API_PROXY=http://localhost:8000/invocations")
		assert.Contains(t, contentStr, "CHAT_APP_PORT=3000")

		// Verify the experiment ID is not empty
		assert.NotContains(t, contentStr, "MLFLOW_EXPERIMENT_ID=\n")
		assert.NotContains(t, contentStr, "MLFLOW_EXPERIMENT_ID=$")

		// Verify that MLFLOW_TRACKING_URI is overridden (only one occurrence)
		count := strings.Count(contentStr, "MLFLOW_TRACKING_URI=")
		assert.Equal(t, 1, count, "Should have exactly one MLFLOW_TRACKING_URI")
	})

	t.Run("end-to-end with named profile overrides MLFLOW_TRACKING_URI", func(t *testing.T) {
		// Setup: Create a temporary directory with app.yml that has MLFLOW_TRACKING_URI
		tmpDir := t.TempDir()
		appYml := filepath.Join(tmpDir, "app.yml")
		appYmlContent := `command: ["uv", "run", "start-app"]
env:
  - name: MLFLOW_TRACKING_URI
    value: "databricks"
  - name: MLFLOW_REGISTRY_URI
    value: "databricks-uc"
  - name: MLFLOW_EXPERIMENT_ID
    valueFrom: experiment
`
		err := os.WriteFile(appYml, []byte(appYmlContent), 0o644)
		require.NoError(t, err)

		// Create resource map
		resources := map[string]string{
			"experiment": "12345",
		}

		// Build and write .env with a named profile
		builder, err := NewEnvFileBuilder("https://test.databricks.com", "my-profile", "test-app", appYml, resources)
		require.NoError(t, err)

		err = builder.WriteEnvFile(tmpDir)
		require.NoError(t, err)

		// Verify the generated .env file
		envPath := filepath.Join(tmpDir, ".env")
		content, err := os.ReadFile(envPath)
		require.NoError(t, err)

		contentStr := string(content)
		// Verify that MLFLOW_TRACKING_URI uses the profile format and overrides app.yml
		assert.Contains(t, contentStr, "DATABRICKS_APP_NAME=test-app")
		assert.Contains(t, contentStr, "MLFLOW_TRACKING_URI=databricks://my-profile")
		assert.Contains(t, contentStr, "MLFLOW_REGISTRY_URI=databricks-uc")
		assert.Contains(t, contentStr, "MLFLOW_EXPERIMENT_ID=12345")

		// Verify that there's only one MLFLOW_TRACKING_URI
		count := strings.Count(contentStr, "MLFLOW_TRACKING_URI=")
		assert.Equal(t, 1, count, "Should have exactly one MLFLOW_TRACKING_URI")
	})
}
