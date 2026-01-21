package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle/internal/annotation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input    string
		expected [3]int
		wantErr  bool
	}{
		{"v0.228.0", [3]int{0, 228, 0}, false},
		{"v1.2.3", [3]int{1, 2, 3}, false},
		{"0.228.0", [3]int{0, 228, 0}, false},
		{"v0.228", [3]int{}, true},
		{"invalid", [3]int{}, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := parseVersion(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, result)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b     [3]int
		expected int
	}{
		{[3]int{0, 228, 0}, [3]int{0, 228, 0}, 0},
		{[3]int{0, 228, 0}, [3]int{0, 229, 0}, -1},
		{[3]int{0, 229, 0}, [3]int{0, 228, 0}, 1},
		{[3]int{1, 0, 0}, [3]int{0, 999, 999}, 1},
		{[3]int{0, 228, 1}, [3]int{0, 228, 0}, 1},
	}

	for _, tt := range tests {
		result := compareVersions(tt.a, tt.b)
		assert.Equal(t, tt.expected, result)
	}
}

func TestFilterVersionsAfter(t *testing.T) {
	versions := []string{"v0.228.0", "v0.229.0", "0.229.1", "v0.230.0", "v1.0.0"}

	t.Run("empty after returns all", func(t *testing.T) {
		result := filterVersionsAfter(versions, "")
		assert.Equal(t, versions, result)
	})

	t.Run("filters after v0.228.0", func(t *testing.T) {
		result := filterVersionsAfter(versions, "v0.228.0")
		assert.Equal(t, []string{"v0.229.0", "0.229.1", "v0.230.0", "v1.0.0"}, result)
	})

	t.Run("filters after v0.229.1", func(t *testing.T) {
		result := filterVersionsAfter(versions, "v0.229.1")
		assert.Equal(t, []string{"v0.230.0", "v1.0.0"}, result)
	})

	t.Run("filters after v0.230.0", func(t *testing.T) {
		result := filterVersionsAfter(versions, "v0.230.0")
		assert.Equal(t, []string{"v1.0.0"}, result)
	})

	t.Run("returns empty for last version", func(t *testing.T) {
		result := filterVersionsAfter(versions, "v1.0.0")
		assert.Empty(t, result)
	})

	t.Run("returns empty for future version", func(t *testing.T) {
		result := filterVersionsAfter(versions, "v1.0.1")
		assert.Empty(t, result)
	})
}

func TestReadLastProcessedVersion(t *testing.T) {
	dir := t.TempDir()

	t.Run("file exists", func(t *testing.T) {
		path := filepath.Join(dir, "version1")
		err := os.WriteFile(path, []byte("v0.281.0\n"), 0o644)
		require.NoError(t, err)

		result := readLastProcessedVersion(path)
		assert.Equal(t, "v0.281.0", result)
	})

	t.Run("file does not exist", func(t *testing.T) {
		path := filepath.Join(dir, "nonexistent")
		result := readLastProcessedVersion(path)
		assert.Equal(t, "", result)
	})

	t.Run("file with whitespace", func(t *testing.T) {
		path := filepath.Join(dir, "version2")
		err := os.WriteFile(path, []byte("  v0.280.0  \n"), 0o644)
		require.NoError(t, err)

		result := readLastProcessedVersion(path)
		assert.Equal(t, "v0.280.0", result)
	})
}

func TestIsBundleCliPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"github.com/databricks/cli/bundle/config.Root", true},
		{"github.com/databricks/cli/libs/dyn.Value", true},
		{"github.com/databricks/databricks-sdk-go/service/jobs.Job", false},
		{"github.com/other/package.Type", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := isBundleCliPath(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFlattenSchema(t *testing.T) {
	schema := map[string]any{
		"$defs": map[string]any{
			"github.com": map[string]any{
				"databricks": map[string]any{
					"cli": map[string]any{
						"bundle": map[string]any{
							"config.Bundle": map[string]any{
								"properties": map[string]any{
									"name":       map[string]any{"type": "string"},
									"cluster_id": map[string]any{"type": "string"},
								},
							},
						},
					},
				},
			},
		},
		"properties": map[string]any{
			"bundle":    map[string]any{},
			"resources": map[string]any{},
		},
	}

	fields := flattenSchema(schema)

	assert.True(t, fields["github.com/databricks/cli/bundle/config.Bundle.name"])
	assert.True(t, fields["github.com/databricks/cli/bundle/config.Bundle.cluster_id"])
	assert.True(t, fields["github.com/databricks/cli/bundle/config.Root.bundle"])
	assert.True(t, fields["github.com/databricks/cli/bundle/config.Root.resources"])
}

func TestFilterToCurrentFields(t *testing.T) {
	sinceVersions := map[string]map[string]string{
		"type.A": {
			"field1": "v0.228.0",
			"field2": "v0.229.0",
		},
		"type.B": {
			"field3": "v0.230.0",
		},
	}

	currentFields := map[string]bool{
		"type.A.field1": true,
		"type.B.field3": true,
		// field2 is not in current schema (was removed)
	}

	result := filterToCurrentFields(sinceVersions, currentFields)

	assert.Equal(t, "v0.228.0", result["type.A"]["field1"])
	assert.NotContains(t, result["type.A"], "field2")
	assert.Equal(t, "v0.230.0", result["type.B"]["field3"])
}

func TestUpdateAnnotationsWithVersions(t *testing.T) {
	t.Run("adds new since_version", func(t *testing.T) {
		annotations := annotation.File{
			"github.com/databricks/cli/bundle/config.Bundle": {
				"name": annotation.Descriptor{Description: "The bundle name"},
			},
		}

		sinceVersions := map[string]map[string]string{
			"github.com/databricks/cli/bundle/config.Bundle": {
				"name":       "v0.228.0",
				"cluster_id": "v0.229.0",
			},
		}

		added := updateAnnotationsWithVersions(annotations, sinceVersions, nil, true)

		assert.Equal(t, 2, added)
		assert.Equal(t, "v0.228.0", annotations["github.com/databricks/cli/bundle/config.Bundle"]["name"].SinceVersion)
		assert.Equal(t, "v0.229.0", annotations["github.com/databricks/cli/bundle/config.Bundle"]["cluster_id"].SinceVersion)
	})

	t.Run("skips existing since_version", func(t *testing.T) {
		annotations := annotation.File{
			"github.com/databricks/cli/bundle/config.Bundle": {
				"name": annotation.Descriptor{
					Description:  "The bundle name",
					SinceVersion: "v0.200.0", // Already set
				},
			},
		}

		sinceVersions := map[string]map[string]string{
			"github.com/databricks/cli/bundle/config.Bundle": {
				"name": "v0.228.0",
			},
		}

		added := updateAnnotationsWithVersions(annotations, sinceVersions, nil, true)

		assert.Equal(t, 0, added)
		// Should keep the original value
		assert.Equal(t, "v0.200.0", annotations["github.com/databricks/cli/bundle/config.Bundle"]["name"].SinceVersion)
	})

	t.Run("filters by CLI types", func(t *testing.T) {
		annotations := annotation.File{}

		sinceVersions := map[string]map[string]string{
			"github.com/databricks/cli/bundle/config.Bundle": {
				"name": "v0.228.0",
			},
			"github.com/databricks/databricks-sdk-go/service/jobs.Job": {
				"name": "v0.228.0",
			},
		}

		// Only CLI types
		added := updateAnnotationsWithVersions(annotations, sinceVersions, nil, true)
		assert.Equal(t, 1, added)
		assert.Contains(t, annotations, "github.com/databricks/cli/bundle/config.Bundle")
		assert.NotContains(t, annotations, "github.com/databricks/databricks-sdk-go/service/jobs.Job")
	})

	t.Run("skips if in other file", func(t *testing.T) {
		annotations := annotation.File{}
		skipIfIn := annotation.File{
			"github.com/databricks/cli/bundle/config.Bundle": {
				"name": annotation.Descriptor{},
			},
		}

		sinceVersions := map[string]map[string]string{
			"github.com/databricks/cli/bundle/config.Bundle": {
				"name":       "v0.228.0",
				"cluster_id": "v0.229.0",
			},
		}

		added := updateAnnotationsWithVersions(annotations, sinceVersions, skipIfIn, true)

		assert.Equal(t, 1, added) // Only cluster_id added, name skipped
		assert.NotContains(t, annotations["github.com/databricks/cli/bundle/config.Bundle"], "name")
		assert.Equal(t, "v0.229.0", annotations["github.com/databricks/cli/bundle/config.Bundle"]["cluster_id"].SinceVersion)
	})
}

func TestWalkDefs(t *testing.T) {
	defs := map[string]any{
		"github.com": map[string]any{
			"databricks": map[string]any{
				"cli": map[string]any{
					"bundle": map[string]any{
						"config.Bundle": map[string]any{
							"properties": map[string]any{
								"name": map[string]any{"type": "string"},
							},
						},
					},
				},
			},
		},
	}

	result := walkDefs(defs, "")

	assert.Contains(t, result, "github.com/databricks/cli/bundle/config.Bundle")
	assert.Contains(t, result["github.com/databricks/cli/bundle/config.Bundle"], "name")
}
