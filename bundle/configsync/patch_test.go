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

// for readability of test cases
func nl(s string) string {
	return strings.TrimPrefix(s, "\n")
}

func TestPreserveBlankLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "basic",
			input: nl(`
key1: value1

key2: value2
`),
			expected: nl(`
key1: value1
# __YAMLPATCH_BLANK_LINE__
key2: value2
`),
		},
		{
			name: "no blanks",
			input: nl(`
key1: value1
key2: value2
`),
			expected: nl(`
key1: value1
key2: value2
`),
		},
		{
			name: "consecutive blanks",
			input: nl(`
key1: value1


key2: value2
`),
			expected: nl(`
key1: value1
# __YAMLPATCH_BLANK_LINE__
# __YAMLPATCH_BLANK_LINE__
key2: value2
`),
		},
		{
			name: "block scalar mid-content blank preserved",
			input: nl(`
key: |
  line1

  line2
other: value
`),
			expected: nl(`
key: |
  line1

  line2
other: value
`),
		},
		{
			name: "block scalar trailing blank becomes marker",
			input: nl(`
key: |
  line1
  line2

next: value
`),
			expected: nl(`
key: |
  line1
  line2
# __YAMLPATCH_BLANK_LINE__
next: value
`),
		},
		{
			name: "folded block scalar trailing blank",
			input: nl(`
key: >-
  line1
  line2

next: value
`),
			expected: nl(`
key: >-
  line1
  line2
# __YAMLPATCH_BLANK_LINE__
next: value
`),
		},
		{
			name: "block scalar as list item",
			input: nl(`
items:
  - |
    line1

    line2

next: value
`),
			expected: nl(`
items:
  - |
    line1

    line2
# __YAMLPATCH_BLANK_LINE__
next: value
`),
		},
		{
			name: "block scalar at EOF",
			input: nl(`
key: |
  content

`),
			expected: nl(`
key: |
  content
# __YAMLPATCH_BLANK_LINE__
`),
		},
		{
			name: "consecutive blanks inside block scalar",
			input: nl(`
key: |
  line1


  line2
next: value
`),
			expected: nl(`
key: |
  line1


  line2
next: value
`),
		},
		{
			name: "back-to-back block scalars",
			input: nl(`
key1: |
  content1

key2: |
  content2
`),
			expected: nl(`
key1: |
  content1
# __YAMLPATCH_BLANK_LINE__
key2: |
  content2
`),
		},
		{
			name: "block scalar with indent indicator",
			input: nl(`
key: |2
  line1

  line2
next: value
`),
			expected: nl(`
key: |2
  line1

  line2
next: value
`),
		},
		{
			name: "indented content",
			input: nl(`
resources:
  jobs:
    my_job:
      name: test

      tasks:
        - task_key: main

      tags:
        env: dev
`),
			expected: nl(`
resources:
  jobs:
    my_job:
      name: test
# __YAMLPATCH_BLANK_LINE__
      tasks:
        - task_key: main
# __YAMLPATCH_BLANK_LINE__
      tags:
        env: dev
`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(preserveBlankLines([]byte(tt.input))))
		})
	}
}

func TestRestoreBlankLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "basic",
			input: nl(`
key1: value1
# __YAMLPATCH_BLANK_LINE__
key2: value2
`),
			expected: nl(`
key1: value1

key2: value2
`),
		},
		{
			name: "indented marker",
			input: nl(`
  key1: value1
  # __YAMLPATCH_BLANK_LINE__
  key2: value2
`),
			expected: nl(`
  key1: value1

  key2: value2
`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(restoreBlankLines([]byte(tt.input))))
		})
	}
}

func TestPreserveAndRestoreRoundTrip(t *testing.T) {
	input := nl(`
key1: value1

key2: value2


key3: value3
`)
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
