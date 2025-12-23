package app

import (
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
