package configsync

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

func TestApplyChangesToYAML_PreserveBlankLines(t *testing.T) {
	ctx := logdiag.InitContext(context.Background())

	tmpDir := t.TempDir()

	yamlContent := `# Comment at top
resources:
  jobs:
    test_job:
      name: "Test Job"

      # Comment before timeout
      timeout_seconds: 3600

      tasks:
        - task_key: main
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

	modified := fileChanges[0].ModifiedContent

	assert.Equal(t, 2, strings.Count(modified, "\n\n"), "both blank lines should be preserved")
	assert.Contains(t, modified, "timeout_seconds: 7200")
	assert.NotContains(t, modified, blankLineMarker)
}

func TestPreserveBlankLines(t *testing.T) {
	input := "key1: value1\n\nkey2: value2\n"
	expected := "key1: value1\n" + blankLineMarker + "\nkey2: value2\n"
	assert.Equal(t, expected, string(preserveBlankLines([]byte(input))))
}

func TestPreserveBlankLines_BlockScalar(t *testing.T) {
	input := "key: |\n  line1\n\n  line2\nother: value\n"
	// Blank line inside block scalar should NOT be replaced
	assert.Equal(t, input, string(preserveBlankLines([]byte(input))))
}

func TestPreserveBlankLines_BlockScalarTrailing(t *testing.T) {
	// Trailing blank line after block scalar content should be replaced with marker
	// (yaml.v3 clips trailing newlines, so we must preserve them as markers).
	input := "key: |\n  line1\n  line2\n\nnext: value\n"
	expected := "key: |\n  line1\n  line2\n" + blankLineMarker + "\nnext: value\n"
	assert.Equal(t, expected, string(preserveBlankLines([]byte(input))))
}

func TestPreserveBlankLines_FoldedBlockScalar(t *testing.T) {
	input := "key: >-\n  line1\n  line2\n\nnext: value\n"
	expected := "key: >-\n  line1\n  line2\n" + blankLineMarker + "\nnext: value\n"
	assert.Equal(t, expected, string(preserveBlankLines([]byte(input))))
}

func TestPreserveBlankLines_ConsecutiveBlanks(t *testing.T) {
	input := "key1: value1\n\n\nkey2: value2\n"
	expected := "key1: value1\n" + blankLineMarker + "\n" + blankLineMarker + "\nkey2: value2\n"
	assert.Equal(t, expected, string(preserveBlankLines([]byte(input))))
}

func TestRestoreBlankLines(t *testing.T) {
	input := "key1: value1\n" + blankLineMarker + "\nkey2: value2\n"
	expected := "key1: value1\n\nkey2: value2\n"
	assert.Equal(t, expected, string(restoreBlankLines([]byte(input))))
}

func TestRestoreBlankLines_Indented(t *testing.T) {
	input := "  key1: value1\n  " + blankLineMarker + "\n  key2: value2\n"
	expected := "  key1: value1\n\n  key2: value2\n"
	assert.Equal(t, expected, string(restoreBlankLines([]byte(input))))
}

func TestPreserveAndRestoreRoundTrip(t *testing.T) {
	input := "key1: value1\n\nkey2: value2\n\n\nkey3: value3\n"
	assert.Equal(t, input, string(restoreBlankLines(preserveBlankLines([]byte(input)))))
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
