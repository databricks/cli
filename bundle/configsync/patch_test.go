package configsync

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/palantir/pkg/yamlpatch/yamlpatch"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyChangesToYAML_PreserveComments(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	tmpDir := t.TempDir()

	yamlContent := `# Comment at top
resources:
  jobs:
    test_job:
      name: "Test Job"
      # Comment before timeout
      timeout_seconds: 3600
`

	yamlPath := filepath.Join(tmpDir, "databricks.yml")
	err := os.WriteFile(yamlPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)

	mutator.DefaultMutators(ctx, b)

	changes := Changes{
		"resources.jobs.test_job": ResourceChanges{
			"timeout_seconds": &ConfigChangeDesc{
				Operation: OperationReplace,
				Value:     7200,
			},
		},
	}

	fieldChanges, err := ResolveChanges(ctx, b, changes)
	require.NoError(t, err)

	fileChanges, err := ApplyChangesToYAML(ctx, b, fieldChanges)
	require.NoError(t, err)
	require.Len(t, fileChanges, 1)

	assert.Contains(t, fileChanges[0].ModifiedContent, "# Comment at top")
	assert.Contains(t, fileChanges[0].ModifiedContent, "# Comment before timeout")
	assert.Contains(t, fileChanges[0].ModifiedContent, "timeout_seconds: 7200")
}

func TestBuildNestedMaps(t *testing.T) {
	targetPath, err := yamlpatch.ParsePath("/targets/default/resources/pipelines/my_pipeline/tags/foo")
	require.NoError(t, err)

	missingPath, err := yamlpatch.ParsePath("/targets/default/resources")
	require.NoError(t, err)

	result := buildNestedMaps(targetPath, missingPath, "bar")

	expected := map[string]any{
		"pipelines": map[string]any{
			"my_pipeline": map[string]any{
				"tags": map[string]any{
					"foo": "bar",
				},
			},
		},
	}
	assert.Equal(t, expected, result)
}
