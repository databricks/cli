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

func TestFindInterpolationAtPositionBasic(t *testing.T) {
	lines := []string{`      name: "${resources.jobs.my_job.name}"`}
	// Cursor inside the interpolation.
	ref, ok := lsp.FindInterpolationAtPosition(lines, lsp.Position{Line: 0, Character: 15})
	require.True(t, ok)
	assert.Equal(t, "resources.jobs.my_job.name", ref.Path)
}

func TestFindInterpolationAtPositionMultiple(t *testing.T) {
	lines := []string{`value: "${a.b} and ${c.d}"`}
	// Cursor on the second interpolation.
	ref, ok := lsp.FindInterpolationAtPosition(lines, lsp.Position{Line: 0, Character: 21})
	require.True(t, ok)
	assert.Equal(t, "c.d", ref.Path)
}

func TestFindInterpolationAtPositionOutside(t *testing.T) {
	lines := []string{`value: "${a.b} plain text ${c.d}"`}
	// Cursor on "plain text" between the two interpolations.
	_, ok := lsp.FindInterpolationAtPosition(lines, lsp.Position{Line: 0, Character: 16})
	assert.False(t, ok)
}

func TestFindInterpolationAtPositionAtDollar(t *testing.T) {
	lines := []string{`name: "${var.foo}"`}
	// Cursor on the "$" character.
	idx := strings.Index(lines[0], "$")
	ref, ok := lsp.FindInterpolationAtPosition(lines, lsp.Position{Line: 0, Character: idx})
	require.True(t, ok)
	assert.Equal(t, "var.foo", ref.Path)
}

func TestFindInterpolationAtPositionNone(t *testing.T) {
	lines := []string{`name: "plain string"`}
	_, ok := lsp.FindInterpolationAtPosition(lines, lsp.Position{Line: 0, Character: 10})
	assert.False(t, ok)
}

func TestResolveDefinition(t *testing.T) {
	yaml := `
resources:
  jobs:
    my_job:
      name: "ETL"
`
	tree, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	loc, ok := lsp.ResolveDefinition(tree, "resources.jobs.my_job")
	require.True(t, ok)
	assert.Equal(t, "test.yml", loc.File)
	assert.Positive(t, loc.Line)
}

func TestResolveDefinitionVarShorthand(t *testing.T) {
	yaml := `
variables:
  foo:
    default: "bar"
`
	tree, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	loc, ok := lsp.ResolveDefinition(tree, "var.foo")
	require.True(t, ok)
	assert.Equal(t, "test.yml", loc.File)
}

func TestResolveDefinitionInvalid(t *testing.T) {
	yaml := `
resources:
  jobs:
    my_job:
      name: "ETL"
`
	tree, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	_, ok := lsp.ResolveDefinition(tree, "resources.jobs.nonexistent")
	assert.False(t, ok)
}

func TestFindInterpolationReferences(t *testing.T) {
	yaml := `
resources:
  jobs:
    my_job:
      name: "ETL"
  pipelines:
    my_pipeline:
      name: "${resources.jobs.my_job.name}"
      settings:
        target: "${resources.jobs.my_job.id}"
`
	tree, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	refs := lsp.FindInterpolationReferences(tree, "resources.jobs.my_job")
	require.Len(t, refs, 2)
	assert.Contains(t, refs[0].RefStr, "resources.jobs.my_job")
	assert.Contains(t, refs[1].RefStr, "resources.jobs.my_job")
}

func TestFindInterpolationReferencesNoMatch(t *testing.T) {
	yaml := `
resources:
  jobs:
    my_job:
      name: "${var.name}"
`
	tree, err := yamlloader.LoadYAML("test.yml", strings.NewReader(yaml))
	require.NoError(t, err)

	refs := lsp.FindInterpolationReferences(tree, "resources.jobs.my_job")
	assert.Empty(t, refs)
}

func TestDynLocationToLSPLocation(t *testing.T) {
	loc := dyn.Location{
		File:   "/path/to/file.yml",
		Line:   5,
		Column: 10,
	}

	lspLoc := lsp.DynLocationToLSPLocation(loc)
	assert.Equal(t, "file:///path/to/file.yml", lspLoc.URI)
	assert.Equal(t, 4, lspLoc.Range.Start.Line)
	assert.Equal(t, 9, lspLoc.Range.Start.Character)
}
