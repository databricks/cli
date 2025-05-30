package structwalk

import (
	"testing"

	"github.com/databricks/cli/libs/structdiff/structpath"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func flatten(t *testing.T, value any) map[string]any {
	results := make(map[string]any)
	err := Walk(value, func(path *structpath.PathNode, value any) {
		results[path.String()] = value
	})
	require.NoError(t, err)
	return results
}

func TestValueNil(t *testing.T) {
	assert.Empty(t, flatten(t, nil))
}

func TestValueEmptyMap(t *testing.T) {
	assert.Empty(t, flatten(t, make(map[string]int)))
}

func TestValueEmptySlice(t *testing.T) {
	assert.Empty(t, flatten(t, []string{}))
}

func TestValueInt(t *testing.T) {
	assert.Equal(t, map[string]any{"": 5}, flatten(t, 5))
}

func TestValueTypesEmpty(t *testing.T) {
	expected := map[string]any{
		".-":               "",
		".ArrayString[0]":  "",
		".ArrayString[1]":  "",
		".Array[0].X":      0,
		".Array[1].X":      0,
		".BoolField":       false,
		".EmptyTagField":   "",
		".IntField":        0,
		".Nested.X":        0,
		".ValidFieldNoTag": "",
		".valid_field":     "",
	}

	assert.Equal(t, expected, flatten(t, Types{}))
	assert.Equal(t, expected, flatten(t, &Types{}))
}

func TestValueJobSettings(t *testing.T) {
	jobSettings := jobs.JobSettings{
		Name:              "test-job",
		MaxConcurrentRuns: 5,
		TimeoutSeconds:    3600,
		Tags:              map[string]string{"env": "test", "team": "data"},
	}

	results := make(map[string]any)

	err := Walk(jobSettings, func(path *structpath.PathNode, val any) {
		results[path.DynPath()] = val
	})

	require.NoError(t, err)
	require.NotEmpty(t, results)

	// Get sorted keys for consistent testing
	sortedKeys := utils.SortedKeys(results)

	// Test first 5 paths (alphabetically sorted)
	require.GreaterOrEqual(t, len(sortedKeys), 5, "Should have at least 5 results")

	expectedFirst5Paths := []string{
		"budget_policy_id",
		"description",
		"edit_mode",
		"format",
		"max_concurrent_runs",
	}

	for i, expectedPath := range expectedFirst5Paths {
		assert.Equal(t, expectedPath, sortedKeys[i], "First 5 paths mismatch at index %d", i)
	}

	// Test last 5 paths
	require.GreaterOrEqual(t, len(sortedKeys), 5, "Should have at least 5 results for last 5 test")

	expectedLast5Paths := []string{
		"name",
		"performance_target",
		`tags.env`,
		`tags.team`,
		"timeout_seconds",
	}

	lastIndex := len(sortedKeys) - 5
	for i, expectedPath := range expectedLast5Paths {
		assert.Equal(t, expectedPath, sortedKeys[lastIndex+i], "Last 5 paths mismatch at index %d", i)
	}

	// Test some specific paths that we know should exist with expected values
	pathsToFind := map[string]any{
		// Map access with different keys
		`tags.env`:  "test",
		`tags.team`: "data",
		// Basic scalar fields
		"name":                "test-job",
		"max_concurrent_runs": 5,
		"timeout_seconds":     3600,
	}

	for expectedPath, expectedValue := range pathsToFind {
		value, found := results[expectedPath]
		assert.True(t, found, "Expected path not found: %s", expectedPath)
		if found {
			assert.Equal(t, expectedValue, value, "Value mismatch for path %s", expectedPath)
		}
	}

	// Test some interesting edge cases - fields with zero/empty values
	edgeCasePaths := []string{
		"budget_policy_id", // empty string
		"description",      // empty string
		"edit_mode",        // zero value enum
		"format",           // zero value enum
	}

	for _, path := range edgeCasePaths {
		_, found := results[path]
		assert.True(t, found, "Edge case path not found: %s", path)
	}
}

func TestVisitNil(t *testing.T) {
	err := Walk(jobs.JobSettings{}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "visit callback must not be nil")
}
