package lsp_test

import (
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/lsp"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindCompletionContextAfterDollarBrace(t *testing.T) {
	lines := []string{`  name: "${"`}
	// Cursor is right after "${" which starts at index 9 ($), so partial starts at 11.
	ctx, ok := lsp.FindCompletionContext(lines, lsp.Position{Line: 0, Character: 11})
	require.True(t, ok)
	assert.Equal(t, 9, ctx.Start)
	assert.Equal(t, "", ctx.PartialPath)
}

func TestFindCompletionContextPartialPath(t *testing.T) {
	lines := []string{`  name: "${var.clust"`}
	ctx, ok := lsp.FindCompletionContext(lines, lsp.Position{Line: 0, Character: 20})
	require.True(t, ok)
	assert.Equal(t, "var.clust", ctx.PartialPath)
}

func TestFindCompletionContextAfterDot(t *testing.T) {
	//                0123456789012345678901234567
	lines := []string{`  name: "${resources.jobs."`}
	// "${" starts at 9, partial path starts at 11, cursor at 26 (after trailing dot)
	ctx, ok := lsp.FindCompletionContext(lines, lsp.Position{Line: 0, Character: 26})
	require.True(t, ok)
	assert.Equal(t, "resources.jobs.", ctx.PartialPath)
}

func TestFindCompletionContextNotInInterpolation(t *testing.T) {
	lines := []string{`  name: "hello world"`}
	_, ok := lsp.FindCompletionContext(lines, lsp.Position{Line: 0, Character: 15})
	assert.False(t, ok)
}

func TestFindCompletionContextClosedBrace(t *testing.T) {
	//                01234567890123456789
	lines := []string{`  name: "${var.foo}"`}
	// Cursor at 19, after the closing "}"
	_, ok := lsp.FindCompletionContext(lines, lsp.Position{Line: 0, Character: 19})
	assert.False(t, ok)
}

func TestCompleteInterpolationTopLevelKeys(t *testing.T) {
	yaml := `
bundle:
  name: test
variables:
  foo:
    default: bar
resources:
  jobs:
    my_job:
      name: hello
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	items := lsp.CompleteInterpolation(v, "", nil)
	labels := extractLabels(items)
	assert.Contains(t, labels, "bundle")
	assert.Contains(t, labels, "variables")
	assert.Contains(t, labels, "resources")
}

func TestCompleteInterpolationVarShorthand(t *testing.T) {
	yaml := `
variables:
  cluster_id:
    default: "abc"
  cluster_name:
    default: "my-cluster"
  warehouse_id:
    default: "def"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	items := lsp.CompleteInterpolation(v, "var.", nil)
	labels := extractLabels(items)
	assert.Contains(t, labels, "var.cluster_id")
	assert.Contains(t, labels, "var.cluster_name")
	assert.Contains(t, labels, "var.warehouse_id")
}

func TestCompleteInterpolationVarShorthandWithPrefix(t *testing.T) {
	yaml := `
variables:
  cluster_id:
    default: "abc"
  cluster_name:
    default: "my-cluster"
  warehouse_id:
    default: "def"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	items := lsp.CompleteInterpolation(v, "var.cluster", nil)
	labels := extractLabels(items)
	assert.Contains(t, labels, "var.cluster_id")
	assert.Contains(t, labels, "var.cluster_name")
	assert.NotContains(t, labels, "var.warehouse_id")
}

func TestCompleteInterpolationVarWithoutTrailingDot(t *testing.T) {
	yaml := `
variables:
  cluster_id:
    default: "abc"
  cluster_name:
    default: "my-cluster"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	// Typing "${var" (no trailing dot) should produce "var.cluster_id", not ".cluster_id".
	items := lsp.CompleteInterpolation(v, "var", nil)
	labels := extractLabels(items)
	assert.Contains(t, labels, "var.cluster_id")
	assert.Contains(t, labels, "var.cluster_name")
	assert.NotContains(t, labels, ".cluster_id")
	assert.NotContains(t, labels, "variables")
}

func TestCompleteInterpolationResourceJobs(t *testing.T) {
	yaml := `
resources:
  jobs:
    etl_job:
      name: "ETL"
    report_job:
      name: "Report"
  pipelines:
    dlt:
      name: "DLT"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	items := lsp.CompleteInterpolation(v, "resources.jobs.", nil)
	labels := extractLabels(items)
	assert.Contains(t, labels, "resources.jobs.etl_job")
	assert.Contains(t, labels, "resources.jobs.report_job")
	assert.Len(t, items, 2)
}

func TestCompleteInterpolationResourceTypes(t *testing.T) {
	yaml := `
resources:
  jobs:
    my_job:
      name: "hello"
  pipelines:
    my_pipeline:
      name: "world"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	items := lsp.CompleteInterpolation(v, "resources.", nil)
	labels := extractLabels(items)
	assert.Contains(t, labels, "resources.jobs")
	assert.Contains(t, labels, "resources.pipelines")
}

func TestCompleteInterpolationNoMatch(t *testing.T) {
	yaml := `
bundle:
  name: test
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	items := lsp.CompleteInterpolation(v, "nonexistent.", nil)
	assert.Empty(t, items)
}

func TestTopLevelCompletionsIncludesVarShorthand(t *testing.T) {
	yaml := `
bundle:
  name: test
variables:
  foo:
    default: bar
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	items := lsp.TopLevelCompletions(v, nil)
	labels := extractLabels(items)
	assert.Contains(t, labels, "var")
	assert.Contains(t, labels, "bundle")
	assert.Contains(t, labels, "variables")
}

func TestTopLevelCompletionsNoVarWithoutVariables(t *testing.T) {
	yaml := `
bundle:
  name: test
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	items := lsp.TopLevelCompletions(v, nil)
	labels := extractLabels(items)
	assert.NotContains(t, labels, "var")
	assert.Contains(t, labels, "bundle")
}

func TestCompleteInterpolationSequenceExpandedInline(t *testing.T) {
	yaml := `
resources:
  pipelines:
    dlt:
      name: "DLT"
      libraries:
        - notebook:
            path: ./a.py
        - notebook:
            path: ./b.py
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	// Typing "${resources.pipelines.dlt." should expand libraries inline as [0], [1].
	items := lsp.CompleteInterpolation(v, "resources.pipelines.dlt.", nil)
	labels := extractLabels(items)
	assert.Contains(t, labels, "resources.pipelines.dlt.name")
	assert.Contains(t, labels, "resources.pipelines.dlt.libraries[0]")
	assert.Contains(t, labels, "resources.pipelines.dlt.libraries[1]")
}

func TestCompleteInterpolationSequenceWithPartialIndex(t *testing.T) {
	yaml := `
resources:
  pipelines:
    dlt:
      libraries:
        - notebook:
            path: ./a.py
        - notebook:
            path: ./b.py
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	// Typing "${resources.pipelines.dlt.libraries[" should still suggest indices.
	items := lsp.CompleteInterpolation(v, "resources.pipelines.dlt.libraries[", nil)
	labels := extractLabels(items)
	assert.Contains(t, labels, "resources.pipelines.dlt.libraries[0]")
	assert.Contains(t, labels, "resources.pipelines.dlt.libraries[1]")
	assert.Len(t, items, 2)
}

func TestCompleteInterpolationAfterIndex(t *testing.T) {
	yaml := `
resources:
  jobs:
    my_job:
      tasks:
        - task_key: ingest
          notebook_task:
            notebook_path: ./a.py
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	// Typing "${resources.jobs.my_job.tasks[0]." should show keys of the first task.
	items := lsp.CompleteInterpolation(v, "resources.jobs.my_job.tasks[0].", nil)
	labels := extractLabels(items)
	assert.Contains(t, labels, "resources.jobs.my_job.tasks[0].task_key")
	assert.Contains(t, labels, "resources.jobs.my_job.tasks[0].notebook_task")
}

func TestCompleteInterpolationComputedBundleKeys(t *testing.T) {
	yaml := `
bundle:
  name: test
  git:
    origin_url: "https://example.com"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	items := lsp.CompleteInterpolation(v, "bundle.", nil)
	labels := extractLabels(items)

	// Tree-based keys.
	assert.Contains(t, labels, "bundle.name")
	assert.Contains(t, labels, "bundle.git")

	// Computed keys.
	assert.Contains(t, labels, "bundle.target")
	assert.Contains(t, labels, "bundle.environment")

	// bundle.git.commit should not appear at this depth; "bundle.git" is the intermediate.
	assert.NotContains(t, labels, "bundle.git.commit")

	// Verify computed items have the right detail.
	for _, item := range items {
		if item.Label == "bundle.target" {
			assert.Equal(t, "computed", item.Detail)
			assert.Equal(t, 6, item.Kind) // completionKindVariable
		}
	}
}

func TestCompleteInterpolationComputedGitSubkeys(t *testing.T) {
	yaml := `
bundle:
  name: test
  git:
    origin_url: "https://example.com"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	items := lsp.CompleteInterpolation(v, "bundle.git.", nil)
	labels := extractLabels(items)

	// Tree-based key.
	assert.Contains(t, labels, "bundle.git.origin_url")

	// Computed keys at this depth.
	assert.Contains(t, labels, "bundle.git.commit")
	assert.Contains(t, labels, "bundle.git.actual_branch")
	assert.Contains(t, labels, "bundle.git.bundle_root_path")
}

func TestCompleteInterpolationComputedWorkspaceCurrentUser(t *testing.T) {
	yaml := `
bundle:
  name: test
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	// workspace doesn't exist in the tree, but computed keys should still appear.
	items := lsp.CompleteInterpolation(v, "workspace.current_user.", nil)
	labels := extractLabels(items)
	assert.Contains(t, labels, "workspace.current_user.short_name")
	assert.Contains(t, labels, "workspace.current_user.user_name")
}

func TestCompleteInterpolationComputedFilterByPrefix(t *testing.T) {
	yaml := `
bundle:
  name: test
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	items := lsp.CompleteInterpolation(v, "bundle.t", nil)
	labels := extractLabels(items)

	// "bundle.target" starts with "bundle.t".
	assert.Contains(t, labels, "bundle.target")

	// "bundle.environment" and "bundle.git.*" do not start with "bundle.t".
	assert.NotContains(t, labels, "bundle.environment")
	assert.NotContains(t, labels, "bundle.git.commit")
	assert.NotContains(t, labels, "bundle.git")
}

func TestCompleteInterpolationComputedTopLevel(t *testing.T) {
	yaml := `
bundle:
  name: test
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	// At top level, "workspace" should appear as a computed intermediate.
	items := lsp.CompleteInterpolation(v, "", nil)
	labels := extractLabels(items)
	assert.Contains(t, labels, "bundle")
	assert.Contains(t, labels, "workspace")
}

func TestCompleteInterpolationComputedNoDuplicates(t *testing.T) {
	yaml := `
bundle:
  name: test
  git:
    origin_url: "https://example.com"
`
	v, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	items := lsp.CompleteInterpolation(v, "bundle.", nil)
	// "bundle.git" should appear only once even though it exists in the tree
	// and would also be generated as a computed intermediate.
	count := 0
	for _, item := range items {
		if item.Label == "bundle.git" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func extractLabels(items []lsp.CompletionItem) []string {
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = item.Label
	}
	return labels
}
