package mutator_test

import (
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/metrics"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMarkSelectUsed_Set checks that the telemetry flag is recorded when --select
// is used. Selector resolution is covered by TestFilterToSelected_* and the
// acceptance tests under acceptance/bundle/select.
func TestMarkSelectUsed_Set(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: map[string]*resources.Job{"my_job": {}},
			},
		},
	}
	b.Select = []string{"my_job"}
	diags := bundle.Apply(t.Context(), b, mutator.MarkSelectUsed())
	require.NoError(t, diags.Error())
	assert.Contains(t, b.Metrics.BoolValues, protos.BoolMapEntry{Key: metrics.SelectUsed, Value: true})
}

// TestMarkSelectUsed_Unset checks that the flag is left unset when --select is not
// used, so bundles that never select keep the metric out of their telemetry.
func TestMarkSelectUsed_Unset(t *testing.T) {
	b := &bundle.Bundle{}
	diags := bundle.Apply(t.Context(), b, mutator.MarkSelectUsed())
	require.NoError(t, diags.Error())
	assert.Empty(t, b.Metrics.BoolValues)
}
