package lsp_test

import (
	"strings"
	"testing"

	"github.com/databricks/cli/bundle/lsp"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiagnoseInterpolationsValidReferences(t *testing.T) {
	yaml := `
resources:
  jobs:
    my_job:
      name: "ETL"
variables:
  env:
    default: "dev"
`
	tree, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	lines := []string{
		`name: "${resources.jobs.my_job.name}"`,
		`env: "${var.env}"`,
	}

	diags := lsp.DiagnoseInterpolations(lines, tree)
	assert.Empty(t, diags)
}

func TestDiagnoseInterpolationsUnresolvableVar(t *testing.T) {
	yaml := `
variables:
  env:
    default: "dev"
`
	tree, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	lines := []string{`name: "${var.nonexistent}"`}

	diags := lsp.DiagnoseInterpolations(lines, tree)
	require.Len(t, diags, 1)
	assert.Equal(t, lsp.DiagnosticSeverityWarning, diags[0].Severity)
	assert.Contains(t, diags[0].Message, "var.nonexistent")
	assert.Equal(t, 0, diags[0].Range.Start.Line)
}

func TestDiagnoseInterpolationsComputedKeysSkipped(t *testing.T) {
	tree := dyn.NewValue(map[string]dyn.Value{}, []dyn.Location{})

	lines := []string{
		`target: "${bundle.target}"`,
		`env: "${bundle.environment}"`,
		`user: "${workspace.current_user.short_name}"`,
		`name: "${workspace.current_user.user_name}"`,
		`commit: "${bundle.git.commit}"`,
		`branch: "${bundle.git.actual_branch}"`,
	}

	diags := lsp.DiagnoseInterpolations(lines, tree)
	assert.Empty(t, diags)
}

func TestDiagnoseInterpolationsMissingResource(t *testing.T) {
	yaml := `
resources:
  jobs:
    my_job:
      name: "ETL"
`
	tree, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	lines := []string{`ref: "${resources.jobs.missing}"`}

	diags := lsp.DiagnoseInterpolations(lines, tree)
	require.Len(t, diags, 1)
	assert.Equal(t, lsp.DiagnosticSeverityWarning, diags[0].Severity)
	assert.Contains(t, diags[0].Message, "resources.jobs.missing")
}

func TestDiagnoseInterpolationsMultipleOnSameLine(t *testing.T) {
	yaml := `
resources:
  jobs:
    my_job:
      name: "ETL"
`
	tree, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	lines := []string{`value: "${resources.jobs.my_job.name} ${resources.jobs.bad}"`}

	diags := lsp.DiagnoseInterpolations(lines, tree)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "resources.jobs.bad")
}

func TestDiagnoseInterpolationsEmptyTree(t *testing.T) {
	lines := []string{`name: "${var.something}"`}

	diags := lsp.DiagnoseInterpolations(lines, dyn.InvalidValue)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "var.something")
}

func TestDiagnoseInterpolationsDiagnosticRange(t *testing.T) {
	yaml := `
variables:
  env:
    default: "dev"
`
	tree, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	lines := []string{`  name: "${var.missing_var}"`}

	diags := lsp.DiagnoseInterpolations(lines, tree)
	require.Len(t, diags, 1)

	// The range should cover the "${var.missing_var}" text.
	start := strings.Index(lines[0], "${")
	end := strings.Index(lines[0], "}") + 1
	assert.Equal(t, start, diags[0].Range.Start.Character)
	assert.Equal(t, end, diags[0].Range.End.Character)
	assert.Equal(t, "databricks-bundle-lsp", diags[0].Source)
}

func TestDiagnoseInterpolationsTypoWithIndex(t *testing.T) {
	yaml := `
resources:
  pipelines:
    dlt_pipeline:
      libraries:
        - notebook:
            path: ./a.py
`
	tree, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	lines := []string{`ref: "${resources.pipelines.dlt_pipeline.librarids[0]}"`}
	diags := lsp.DiagnoseInterpolations(lines, tree)
	require.Len(t, diags, 1)
	assert.Contains(t, diags[0].Message, "librarids")
}

func TestDiagnoseInterpolationsNoInterpolations(t *testing.T) {
	tree := dyn.NewValue(map[string]dyn.Value{}, []dyn.Location{})
	lines := []string{`name: "plain text"`, `value: 42`}

	diags := lsp.DiagnoseInterpolations(lines, tree)
	assert.Empty(t, diags)
}
