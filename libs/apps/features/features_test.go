package features

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateFeatureIDs(t *testing.T) {
	tests := []struct {
		name        string
		featureIDs  []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid feature - analytics",
			featureIDs:  []string{"analytics"},
			expectError: false,
		},
		{
			name:        "empty feature list",
			featureIDs:  []string{},
			expectError: false,
		},
		{
			name:        "nil feature list",
			featureIDs:  nil,
			expectError: false,
		},
		{
			name:        "unknown feature",
			featureIDs:  []string{"unknown-feature"},
			expectError: true,
			errorMsg:    "unknown feature",
		},
		{
			name:        "mix of valid and invalid",
			featureIDs:  []string{"analytics", "invalid"},
			expectError: true,
			errorMsg:    "unknown feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFeatureIDs(tt.featureIDs)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateFeatureDependencies(t *testing.T) {
	tests := []struct {
		name        string
		featureIDs  []string
		flagValues  map[string]string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "analytics with warehouse provided",
			featureIDs:  []string{"analytics"},
			flagValues:  map[string]string{"warehouse-id": "abc123"},
			expectError: false,
		},
		{
			name:        "analytics without warehouse",
			featureIDs:  []string{"analytics"},
			flagValues:  map[string]string{},
			expectError: true,
			errorMsg:    "--warehouse-id",
		},
		{
			name:        "analytics with empty warehouse",
			featureIDs:  []string{"analytics"},
			flagValues:  map[string]string{"warehouse-id": ""},
			expectError: true,
			errorMsg:    "--warehouse-id",
		},
		{
			name:        "no features - no dependencies needed",
			featureIDs:  []string{},
			flagValues:  map[string]string{},
			expectError: false,
		},
		{
			name:        "unknown feature - gracefully ignored",
			featureIDs:  []string{"unknown"},
			flagValues:  map[string]string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFeatureDependencies(tt.featureIDs, tt.flagValues)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGetFeatureIDs(t *testing.T) {
	ids := GetFeatureIDs()

	assert.NotEmpty(t, ids)
	assert.Contains(t, ids, "analytics")
}

func TestBuildPluginStrings(t *testing.T) {
	tests := []struct {
		name           string
		featureIDs     []string
		expectedImport string
		expectedUsage  string
	}{
		{
			name:           "no features",
			featureIDs:     []string{},
			expectedImport: "",
			expectedUsage:  "",
		},
		{
			name:           "nil features",
			featureIDs:     nil,
			expectedImport: "",
			expectedUsage:  "",
		},
		{
			name:           "analytics feature",
			featureIDs:     []string{"analytics"},
			expectedImport: "analytics",
			expectedUsage:  "analytics()",
		},
		{
			name:           "unknown feature - ignored",
			featureIDs:     []string{"unknown"},
			expectedImport: "",
			expectedUsage:  "",
		},
		{
			name:           "mix of known and unknown",
			featureIDs:     []string{"analytics", "unknown"},
			expectedImport: "analytics",
			expectedUsage:  "analytics()",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			importStr, usageStr := BuildPluginStrings(tt.featureIDs)
			assert.Equal(t, tt.expectedImport, importStr)
			assert.Equal(t, tt.expectedUsage, usageStr)
		})
	}
}

func TestCollectDependencies(t *testing.T) {
	tests := []struct {
		name         string
		featureIDs   []string
		expectedDeps int
		expectedIDs  []string
	}{
		{
			name:         "no features",
			featureIDs:   []string{},
			expectedDeps: 0,
			expectedIDs:  nil,
		},
		{
			name:         "analytics feature",
			featureIDs:   []string{"analytics"},
			expectedDeps: 1,
			expectedIDs:  []string{"sql_warehouse_id"},
		},
		{
			name:         "unknown feature",
			featureIDs:   []string{"unknown"},
			expectedDeps: 0,
			expectedIDs:  nil,
		},
		{
			name:         "duplicate features - deduped deps",
			featureIDs:   []string{"analytics", "analytics"},
			expectedDeps: 1,
			expectedIDs:  []string{"sql_warehouse_id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := CollectDependencies(tt.featureIDs)
			assert.Len(t, deps, tt.expectedDeps)

			if tt.expectedIDs != nil {
				for i, expectedID := range tt.expectedIDs {
					assert.Equal(t, expectedID, deps[i].ID)
				}
			}
		})
	}
}

func TestCollectResourceFiles(t *testing.T) {
	tests := []struct {
		name              string
		featureIDs        []string
		expectedResources int
	}{
		{
			name:              "no features",
			featureIDs:        []string{},
			expectedResources: 0,
		},
		{
			name:              "analytics feature",
			featureIDs:        []string{"analytics"},
			expectedResources: 1,
		},
		{
			name:              "unknown feature",
			featureIDs:        []string{"unknown"},
			expectedResources: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resources := CollectResourceFiles(tt.featureIDs)
			assert.Len(t, resources, tt.expectedResources)

			if tt.expectedResources > 0 && tt.featureIDs[0] == "analytics" {
				assert.NotEmpty(t, resources[0].BundleVariables)
				assert.NotEmpty(t, resources[0].BundleResources)
			}
		})
	}
}

func TestDetectPluginsFromServer(t *testing.T) {
	tests := []struct {
		name            string
		serverContent   string
		expectedPlugins []string
	}{
		{
			name: "analytics plugin",
			serverContent: `import { createApp, server, analytics } from '@databricks/appkit';
createApp({
  plugins: [
    server(),
    analytics(),
  ],
}).catch(console.error);`,
			expectedPlugins: []string{"analytics"},
		},
		{
			name: "analytics with other plugins not in AvailableFeatures",
			serverContent: `import { createApp, server, analytics, genie } from '@databricks/appkit';
createApp({
  plugins: [
    server(),
    analytics(),
    genie(),
  ],
}).catch(console.error);`,
			expectedPlugins: []string{"analytics"}, // Only analytics is detected since genie is not in AvailableFeatures
		},
		{
			name:            "no recognized plugins",
			serverContent:   `import { createApp, server } from '@databricks/appkit';`,
			expectedPlugins: nil,
		},
		{
			name: "plugin not in AvailableFeatures",
			serverContent: `createApp({
  plugins: [oauth()],
});`,
			expectedPlugins: nil, // oauth is not in AvailableFeatures, so not detected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp dir with server file
			tempDir := t.TempDir()
			serverDir := tempDir + "/src/server"
			require.NoError(t, os.MkdirAll(serverDir, 0o755))
			require.NoError(t, os.WriteFile(serverDir+"/index.ts", []byte(tt.serverContent), 0o644))

			plugins, err := DetectPluginsFromServer(tempDir)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedPlugins, plugins)
		})
	}
}

func TestDetectPluginsFromServerAlternatePath(t *testing.T) {
	// Test server/server.ts path (common in some templates)
	tempDir := t.TempDir()
	serverDir := tempDir + "/server"
	require.NoError(t, os.MkdirAll(serverDir, 0o755))

	serverContent := `import { createApp, server, analytics } from '@databricks/appkit';
createApp({
  plugins: [
    server(),
    analytics(),
  ],
}).catch(console.error);`

	require.NoError(t, os.WriteFile(serverDir+"/server.ts", []byte(serverContent), 0o644))

	plugins, err := DetectPluginsFromServer(tempDir)
	require.NoError(t, err)
	assert.Equal(t, []string{"analytics"}, plugins)
}

func TestDetectPluginsFromServerNoFile(t *testing.T) {
	tempDir := t.TempDir()
	plugins, err := DetectPluginsFromServer(tempDir)
	require.NoError(t, err)
	assert.Nil(t, plugins)
}

func TestGetPluginDependencies(t *testing.T) {
	tests := []struct {
		name         string
		pluginNames  []string
		expectedDeps []string
	}{
		{
			name:         "analytics plugin",
			pluginNames:  []string{"analytics"},
			expectedDeps: []string{"sql_warehouse_id"},
		},
		{
			name:         "unknown plugin",
			pluginNames:  []string{"server"},
			expectedDeps: nil,
		},
		{
			name:         "empty plugins",
			pluginNames:  []string{},
			expectedDeps: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := GetPluginDependencies(tt.pluginNames)
			if tt.expectedDeps == nil {
				assert.Empty(t, deps)
			} else {
				assert.Len(t, deps, len(tt.expectedDeps))
				for i, dep := range deps {
					assert.Equal(t, tt.expectedDeps[i], dep.ID)
				}
			}
		})
	}
}

func TestHasFeaturesDirectory(t *testing.T) {
	// Test with features directory
	tempDir := t.TempDir()
	require.NoError(t, os.MkdirAll(tempDir+"/features", 0o755))
	assert.True(t, HasFeaturesDirectory(tempDir))

	// Test without features directory
	tempDir2 := t.TempDir()
	assert.False(t, HasFeaturesDirectory(tempDir2))
}

func TestMapPluginsToFeatures(t *testing.T) {
	tests := []struct {
		name             string
		pluginNames      []string
		expectedFeatures []string
	}{
		{
			name:             "analytics plugin maps to analytics feature",
			pluginNames:      []string{"analytics"},
			expectedFeatures: []string{"analytics"},
		},
		{
			name:             "unknown plugin",
			pluginNames:      []string{"server", "unknown"},
			expectedFeatures: nil,
		},
		{
			name:             "empty plugins",
			pluginNames:      []string{},
			expectedFeatures: nil,
		},
		{
			name:             "duplicate plugins",
			pluginNames:      []string{"analytics", "analytics"},
			expectedFeatures: []string{"analytics"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			features := MapPluginsToFeatures(tt.pluginNames)
			if tt.expectedFeatures == nil {
				assert.Empty(t, features)
			} else {
				assert.Equal(t, tt.expectedFeatures, features)
			}
		})
	}
}

func TestPluginPatternGeneration(t *testing.T) {
	// Test that the plugin pattern is dynamically generated from AvailableFeatures
	// This ensures new features with PluginImport are automatically detected

	// Get all plugin imports from AvailableFeatures
	var expectedPlugins []string
	for _, f := range AvailableFeatures {
		if f.PluginImport != "" {
			expectedPlugins = append(expectedPlugins, f.PluginImport)
		}
	}

	// Test that each plugin is matched by the pattern
	for _, plugin := range expectedPlugins {
		testCode := fmt.Sprintf("plugins: [%s()]", plugin)
		matches := pluginPattern.FindAllStringSubmatch(testCode, -1)
		assert.NotEmpty(t, matches, "Pattern should match plugin: %s", plugin)
		assert.Equal(t, plugin, matches[0][1], "Captured group should be plugin name: %s", plugin)
	}

	// Test that non-plugin function calls are not matched
	testCode := "const x = someOtherFunction()"
	matches := pluginPattern.FindAllStringSubmatch(testCode, -1)
	assert.Empty(t, matches, "Pattern should not match non-plugin functions")
}
