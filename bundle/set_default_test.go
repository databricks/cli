package bundle

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetDefaultSkipsPureReference(t *testing.T) {
	root, diags := config.LoadFromBytes("databricks.yml", []byte(`
resources:
  jobs:
    base:
      schedule:
        quartz_cron_expression: "0 0 * * * ?"
    viaref:
      schedule: ${resources.jobs.base.schedule}
`))
	require.Empty(t, diags)
	b := &Bundle{Config: *root}

	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	SetDefault(ctx, b, "resources.jobs.*.schedule.pause_status", "UNPAUSED")
	require.Empty(t, logdiag.FlushCollected(ctx))

	v := b.Config.Value()
	base, err := dyn.GetByPath(v, dyn.MustPathFromString("resources.jobs.base.schedule.pause_status"))
	require.NoError(t, err)
	assert.Equal(t, "UNPAUSED", base.MustString())

	viaref, err := dyn.GetByPath(v, dyn.MustPathFromString("resources.jobs.viaref.schedule"))
	require.NoError(t, err)
	assert.Equal(t, "${resources.jobs.base.schedule}", viaref.MustString())
}
