package validate

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequiredSkipsPureReference(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
resources:
  jobs:
    viaref:
      schedule: ${resources.jobs.base.schedule}
    missing:
      schedule:
        timezone_id: UTC
`))
	require.Empty(t, diags)
	b := &bundle.Bundle{Config: *root}

	diags = Required().Apply(t.Context(), b)
	require.Len(t, diags, 1)
	assert.Equal(t, `required field "quartz_cron_expression" is not set`, diags[0].Summary)
	assert.Equal(t, "resources.jobs.missing.schedule", diags[0].Paths[0].String())
}
