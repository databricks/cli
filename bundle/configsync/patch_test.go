package configsync

import (
	"fmt"
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

func TestApplyChangesToYAML_PreserveFormatting(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())

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

	fieldChanges, skipped, err := ResolveChanges(ctx, b, changes)
	require.NoError(t, err)
	require.Empty(t, skipped)

	fileChanges, unwritable, err := ApplyChangesToYAML(ctx, b, fieldChanges)
	require.NoError(t, err)
	require.Zero(t, unwritable)
	require.Len(t, fileChanges, 1)

	modified := fileChanges[0].ModifiedContent

	assert.Contains(t, modified, "# Comment at top")
	assert.Contains(t, modified, "# Comment before timeout")
	assert.Contains(t, modified, "timeout_seconds: 7200")
	assert.Equal(t, 2, strings.Count(modified, "\n\n"), "both blank lines should be preserved")
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
			name: "trailing blank line",
			input: `key: value

`,
			expected: `key: value
# __YAMLPATCH_BLANK_LINE__
`,
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
		{
			name: "yaml.v3-added blank next to marker is deduplicated",
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
			name: "yaml.v3-added standalone blank is kept",
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
			name: "yaml.v3-added blank near block scalar",
			input: nl(`
key: |
  line1

  line2

# __YAMLPATCH_BLANK_LINE__
next: value
`),
			expected: nl(`
key: |
  line1

  line2

next: value
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
	// Tabs are not valid for YAML indentation but can appear in block scalar values.
	input := nl(`
resources:
  jobs:
    my_job:
      name: "test	job"

      description: |
        Multi-line description.

        Second paragraph.

      tasks:
        - task_key: main
          description: >-
            Folded text
            on two lines.

          notebook_task:
            notebook_path: /notebook

      tags:
        env: dev
        team: data-eng
`)
	assert.Equal(t, input, string(restoreBlankLines(preserveBlankLines([]byte(input)))))
}

// The classification decides whether a change is applied, has its parent
// materialized first, or is left unapplied, so each node shape is asserted
// directly rather than through the error the patcher happens to produce.
func TestResolveParentNode(t *testing.T) {
	content := nl(`
resources:
  jobs:
    absent:
      job_clusters:
        - new_cluster:
            num_workers: 1
    null_placeholder:
      job_clusters:
        - new_cluster:
            spark_env_vars:
    reference:
      job_clusters:
        - new_cluster:
            spark_env_vars: ${var.env_vars}
    empty_mapping:
      job_clusters:
        - new_cluster:
            spark_env_vars: {}
    anchored:
      job_clusters:
        - new_cluster: &cluster
            spark_env_vars:
              FOO: bar
    aliased:
      job_clusters:
        - new_cluster: *cluster
    plain_scalar:
      job_clusters:
        - new_cluster:
            spark_env_vars: disabled
`)

	tests := []struct {
		name         string
		job          string
		wantKind     parentKind
		wantAncestor string
	}{
		{
			name:         "absent parent",
			job:          "absent",
			wantKind:     parentAbsent,
			wantAncestor: "/resources/jobs/absent/job_clusters/0/new_cluster/spark_env_vars",
		},
		{
			name:         "null placeholder",
			job:          "null_placeholder",
			wantKind:     parentNull,
			wantAncestor: "/resources/jobs/null_placeholder/job_clusters/0/new_cluster/spark_env_vars",
		},
		{
			name:         "variable reference",
			job:          "reference",
			wantKind:     parentVariable,
			wantAncestor: "/resources/jobs/reference/job_clusters/0/new_cluster/spark_env_vars",
		},
		{
			name:     "empty mapping",
			job:      "empty_mapping",
			wantKind: parentContainer,
		},
		{
			name:     "mapping reached through an alias",
			job:      "aliased",
			wantKind: parentContainer,
		},
		{
			name:         "plain scalar",
			job:          "plain_scalar",
			wantKind:     parentScalar,
			wantAncestor: "/resources/jobs/plain_scalar/job_clusters/0/new_cluster/spark_env_vars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := yamlpatch.ParsePath("/resources/jobs/" + tt.job + "/job_clusters/0/new_cluster/spark_env_vars/MY_KEY")
			require.NoError(t, err)

			kind, ancestor, err := resolveParentNode([]byte(content), path)
			require.NoError(t, err)
			assert.Equal(t, tt.wantKind, kind)
			assert.Equal(t, tt.wantAncestor, ancestor.String())
		})
	}
}

// A parent that is not a mapping is what config-remote-sync hits in production when
// a cluster policy adds a spark_env_vars entry: the three "empty or not writable"
// spellings must behave the way each of them means.
func TestApplyChangesToYAML_ParentNotAMapping(t *testing.T) {
	const wantAdded = `spark_env_vars:
              MY_KEY: my-value`

	tests := []struct {
		name       string
		envVars    string
		want       string
		unwritable int
	}{
		{
			name:    "absent parent is created",
			envVars: "",
			want:    wantAdded,
		},
		{
			name:    "empty mapping is filled in",
			envVars: "            spark_env_vars: {}",
			want:    wantAdded,
		},
		{
			// "key:" parses as a null scalar rather than the empty mapping "key: {}"
			// produces, so it needs materializing before anything can be added to it.
			name:    "null placeholder is materialized",
			envVars: "            spark_env_vars:",
			want:    wantAdded,
		},
		{
			// Overwriting the reference would resolve it to one target's value for
			// every target that shares the file.
			name:       "variable reference is left unapplied",
			envVars:    "            spark_env_vars: ${var.env_vars}",
			unwritable: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := logdiag.InitContext(t.Context())
			tmpDir := t.TempDir()

			content := fmt.Sprintf(nl(`
resources:
  jobs:
    test_job:
      job_clusters:
        - job_cluster_key: shared
          new_cluster:
            num_workers: 1
%s
`), tt.envVars)

			err := os.WriteFile(filepath.Join(tmpDir, "databricks.yml"), []byte(content), 0o644)
			require.NoError(t, err)

			b, err := bundle.Load(ctx, tmpDir)
			require.NoError(t, err)
			mutator.DefaultMutators(ctx, b)

			changes := Changes{
				"resources.jobs.test_job": ResourceChanges{
					"job_clusters[0].new_cluster.spark_env_vars.MY_KEY": &ConfigChangeDesc{
						Operation: OperationAdd,
						Value:     "my-value",
					},
				},
			}

			fieldChanges, skipped, err := ResolveChanges(ctx, b, changes)
			require.NoError(t, err)
			require.Zero(t, skipped)

			fileChanges, unwritable, err := ApplyChangesToYAML(ctx, b, fieldChanges)
			require.NoError(t, err)
			assert.Equal(t, tt.unwritable, unwritable)

			if tt.want == "" {
				assert.Empty(t, fileChanges)
				return
			}
			require.Len(t, fileChanges, 1)
			assert.Contains(t, fileChanges[0].ModifiedContent, tt.want)
		})
	}
}

// The patcher dereferences aliases, so a change addressed through one is written to
// the anchored node instead of failing.
func TestApplyChangesToYAML_AliasedParent(t *testing.T) {
	ctx := logdiag.InitContext(t.Context())
	tmpDir := t.TempDir()

	content := nl(`
resources:
  jobs:
    job_a:
      job_clusters:
        - job_cluster_key: shared
          new_cluster: &cluster
            num_workers: 1
            spark_env_vars:
              FOO: bar
    job_b:
      job_clusters:
        - job_cluster_key: shared
          new_cluster: *cluster
`)

	err := os.WriteFile(filepath.Join(tmpDir, "databricks.yml"), []byte(content), 0o644)
	require.NoError(t, err)

	b, err := bundle.Load(ctx, tmpDir)
	require.NoError(t, err)
	mutator.DefaultMutators(ctx, b)

	changes := Changes{
		"resources.jobs.job_b": ResourceChanges{
			"job_clusters[0].new_cluster.spark_env_vars.MY_KEY": &ConfigChangeDesc{
				Operation: OperationAdd,
				Value:     "my-value",
			},
		},
	}

	fieldChanges, skipped, err := ResolveChanges(ctx, b, changes)
	require.NoError(t, err)
	require.Zero(t, skipped)

	fileChanges, unwritable, err := ApplyChangesToYAML(ctx, b, fieldChanges)
	require.NoError(t, err)
	require.Zero(t, unwritable)
	require.Len(t, fileChanges, 1)
	assert.Contains(t, fileChanges[0].ModifiedContent, "MY_KEY: my-value")
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
