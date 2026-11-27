package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoPathSyntaxInResourceKey(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
resources:
  jobs:
    a.b: {}
    a[0]: {}
    # These do not break path parsing in the direct engine.
    job_a: {}
    job-b: {}
    a]: {}
    a,b: {}
  pipelines:
    p.q: {}
  schemas:
    s.x: {}
`))
	require.Empty(t, diags)
	b := &bundle.Bundle{Config: *root}

	diags = NoPathSyntaxInResourceKey().Apply(t.Context(), b)

	var summaries, paths []string
	for _, d := range diags {
		assert.Equal(t, diag.Error, d.Severity)
		require.Len(t, d.Locations, 1)
		assert.Equal(t, "databricks.yml", d.Locations[0].File)
		summaries = append(summaries, d.Summary)
		paths = append(paths, d.Paths[0].String())
	}
	assert.Equal(t, []string{
		`resource key "a.b" must not contain "." or "["`,
		`resource key "a[0]" must not contain "." or "["`,
		`resource key "p.q" must not contain "." or "["`,
		`resource key "s.x" must not contain "." or "["`,
	}, summaries)
	assert.Equal(t, []string{
		"resources.jobs['a.b']",
		"resources.jobs['a[0]']",
		"resources.pipelines['p.q']",
		"resources.schemas['s.x']",
	}, paths)
}
