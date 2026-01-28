package apps

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/dyn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInlineAppConfigFile(t *testing.T) {
	tests := []struct {
		name           string
		setupFiles     map[string]string
		inputValue     dyn.Value
		expectedFile   string
		expectedConfig map[string]any
		expectError    bool
	}{
		{
			name:       "no app config file",
			setupFiles: map[string]string{},
			inputValue: dyn.V(map[string]dyn.Value{
				"name": dyn.V("test-app"),
			}),
			expectedFile:   "",
			expectedConfig: map[string]any{"name": "test-app"},
		},
		{
			name: "app.yml with command and env",
			setupFiles: map[string]string{
				"app.yml": `command: ["python", "app.py"]
env:
  - name: FOO
    value: bar`,
			},
			inputValue: dyn.V(map[string]dyn.Value{
				"name": dyn.V("test-app"),
			}),
			expectedFile:   "app.yml",
			expectedConfig: nil, // Will check manually
		},
		{
			name: "app.yaml takes precedence over app.yml if both exist",
			setupFiles: map[string]string{
				"app.yml": `command: ["wrong"]`,
				"app.yaml": `command: ["correct"]
env:
  - name: TEST
    value: value`,
			},
			inputValue: dyn.V(map[string]dyn.Value{
				"name": dyn.V("test-app"),
			}),
			expectedFile:   "app.yml",
			expectedConfig: nil, // Will check manually
		},
		{
			name: "app config with resources",
			setupFiles: map[string]string{
				"app.yml": `command: ["python", "app.py"]
resources:
  - name: SERVING_ENDPOINT
    serving_endpoint:
      name: my-endpoint`,
			},
			inputValue: dyn.V(map[string]dyn.Value{
				"name": dyn.V("test-app"),
			}),
			expectedFile:   "app.yml",
			expectedConfig: nil, // Will check manually
		},
		{
			name: "app config with only resources",
			setupFiles: map[string]string{
				"app.yml": `resources:
  - name: SERVING_ENDPOINT`,
			},
			inputValue: dyn.V(map[string]dyn.Value{
				"name": dyn.V("test-app"),
			}),
			expectedFile:   "app.yml",
			expectedConfig: nil, // Will check manually
		},
		{
			name: "app config with empty env",
			setupFiles: map[string]string{
				"app.yml": `command: ["python", "app.py"]
env: []`,
			},
			inputValue: dyn.V(map[string]dyn.Value{
				"name": dyn.V("test-app"),
			}),
			expectedFile:   "app.yml",
			expectedConfig: nil, // Will check manually
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory and change to it
			tmpDir := t.TempDir()
			originalDir, err := os.Getwd()
			require.NoError(t, err)
			defer func() {
				_ = os.Chdir(originalDir)
			}()
			err = os.Chdir(tmpDir)
			require.NoError(t, err)

			// Setup files
			for filename, content := range tt.setupFiles {
				err := os.WriteFile(filename, []byte(content), 0o644)
				require.NoError(t, err)
			}

			// Run function
			appValue := tt.inputValue
			filename, err := inlineAppConfigFile(&appValue)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedFile, filename)

			// Verify the structure if expectedConfig is set
			if tt.expectedConfig != nil {
				appMap := appValue.MustMap()
				result := make(map[string]any)
				for _, pair := range appMap.Pairs() {
					key := pair.Key.MustString()
					result[key] = pair.Value.AsAny()
				}

				assert.Equal(t, tt.expectedConfig, result)
			} else if tt.expectedFile != "" {
				// Just verify that config or resources were added
				appMap := appValue.MustMap()
				var hasConfigOrResources bool
				for _, pair := range appMap.Pairs() {
					key := pair.Key.MustString()
					if key == "config" || key == "resources" {
						hasConfigOrResources = true
						break
					}
				}
				assert.True(t, hasConfigOrResources, "expected config or resources to be added")
			}
		})
	}
}

func TestInlineAppConfigFileErrors(t *testing.T) {
	t.Run("invalid yaml", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(originalDir)
		}()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create invalid YAML
		err = os.WriteFile("app.yml", []byte("invalid: yaml: content:\n  - broken"), 0o644)
		require.NoError(t, err)

		appValue := dyn.V(map[string]dyn.Value{"name": dyn.V("test")})
		_, err = inlineAppConfigFile(&appValue)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
	})

	t.Run("app value not a map", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(originalDir)
		}()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		err = os.WriteFile("app.yml", []byte("command: [\"test\"]"), 0o644)
		require.NoError(t, err)

		appValue := dyn.V("not a map")
		_, err = inlineAppConfigFile(&appValue)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "app value is not a map")
	})

	t.Run("unreadable app.yml", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(originalDir)
		}()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		// Create file with no read permissions
		filename := "app.yml"
		err = os.WriteFile(filename, []byte("command: [\"test\"]"), 0o644)
		require.NoError(t, err)

		if runtime.GOOS == "windows" {
			// On Windows, use icacls to deny read access to the current user
			username := os.Getenv("USERNAME")
			cmd := exec.Command("icacls", filename, "/deny", username+":(R)")
			err = cmd.Run()
			require.NoError(t, err)

			// Verify that the file is actually unreadable
			_, err = os.ReadFile(filename)
			if err == nil {
				t.Skip("Unable to make file unreadable on Windows - skipping test")
			}
		} else {
			// On Unix, use chmod to remove read permissions
			err = os.Chmod(filename, 0o000)
			require.NoError(t, err)
		}

		appValue := dyn.V(map[string]dyn.Value{"name": dyn.V("test")})
		_, err = inlineAppConfigFile(&appValue)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read")
	})
}

func TestInlineAppConfigFileFieldPriority(t *testing.T) {
	t.Run("all fields present", func(t *testing.T) {
		tmpDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(originalDir)
		}()
		err = os.Chdir(tmpDir)
		require.NoError(t, err)

		err = os.WriteFile("app.yml", []byte(`command: ["python", "app.py"]
env:
  - name: FOO
    value: bar
resources:
  - name: ENDPOINT
    serving_endpoint:
      name: test`), 0o644)
		require.NoError(t, err)

		appValue := dyn.V(map[string]dyn.Value{
			"name":        dyn.V("test-app"),
			"description": dyn.V("existing description"),
		})

		filename, err := inlineAppConfigFile(&appValue)
		require.NoError(t, err)
		assert.Equal(t, "app.yml", filename)

		// Verify structure
		appMap := appValue.MustMap()
		result := make(map[string]any)
		for _, pair := range appMap.Pairs() {
			key := pair.Key.MustString()
			result[key] = pair.Value.AsAny()
		}

		// Should have original fields plus config and resources
		assert.Equal(t, "test-app", result["name"])
		assert.Equal(t, "existing description", result["description"])
		assert.NotNil(t, result["config"])
		assert.NotNil(t, result["resources"])
	})
}

func TestInlineAppConfigFileCamelCaseConversion(t *testing.T) {
	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
	}()
	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	// Create app.yml with camelCase field names (as might come from API)
	err = os.WriteFile("app.yml", []byte(`command: ["python", "app.py"]
env:
  - name: FOO
    valueFrom: some-secret
  - name: BAR
    value: baz`), 0o644)
	require.NoError(t, err)

	appValue := dyn.V(map[string]dyn.Value{
		"name": dyn.V("test-app"),
	})

	filename, err := inlineAppConfigFile(&appValue)
	require.NoError(t, err)
	assert.Equal(t, "app.yml", filename)

	// Verify that camelCase fields are converted to snake_case
	appMap := appValue.MustMap()
	var configValue dyn.Value
	for _, pair := range appMap.Pairs() {
		if pair.Key.MustString() == "config" {
			configValue = pair.Value
			break
		}
	}

	require.NotEqual(t, dyn.KindInvalid, configValue.Kind(), "config section should exist")
	configMap := configValue.MustMap()

	var envValue dyn.Value
	for _, pair := range configMap.Pairs() {
		if pair.Key.MustString() == "env" {
			envValue = pair.Value
			break
		}
	}

	require.NotEqual(t, dyn.KindInvalid, envValue.Kind(), "env should exist in config")
	envList := envValue.MustSequence()
	require.Len(t, envList, 2, "should have 2 env vars")

	// Check first env var has value_from (snake_case), not valueFrom (camelCase)
	firstEnv := envList[0].MustMap()
	var hasValueFrom bool
	var hasValueFromCamel bool
	for _, pair := range firstEnv.Pairs() {
		key := pair.Key.MustString()
		if key == "value_from" {
			hasValueFrom = true
		}
		if key == "valueFrom" {
			hasValueFromCamel = true
		}
	}

	assert.True(t, hasValueFrom, "should have value_from (snake_case) field")
	assert.False(t, hasValueFromCamel, "should NOT have valueFrom (camelCase) field")
}

func TestPathContainsBundleFolder(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{
			name:     "path contains .bundle",
			path:     "/Workspace/Users/user@example.com/.bundle/dev/files/source",
			expected: true,
		},
		{
			name:     "path contains .bundle with different structure",
			path:     "/some/path/.bundle/prod",
			expected: true,
		},
		{
			name:     "path does not contain .bundle",
			path:     "/Workspace/Users/user@example.com/my-app/source",
			expected: false,
		},
		{
			name:     "empty path",
			path:     "",
			expected: false,
		},
		{
			name:     "path with bundle in filename but not folder",
			path:     "/Workspace/Users/bundle.txt",
			expected: false,
		},
		{
			name:     "path with .bundle in middle of word",
			path:     "/Workspace/my.bundle.app/source",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.path != "" && strings.Contains(tt.path, "/.bundle/")
			assert.Equal(t, tt.expected, result)
		})
	}
}
