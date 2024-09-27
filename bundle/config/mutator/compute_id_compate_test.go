package mutator_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/mutator"
	"github.com/databricks/cli/libs/diag"
	"github.com/stretchr/testify/assert"
)

func TestComputeIdToClusterId(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Bundle: config.Bundle{
				ComputeId: "compute-id",
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, mutator.ComputeIdToClusterId())
	assert.NoError(t, diags.Error())
	assert.Equal(t, "compute-id", b.Config.Bundle.ClusterId)
	assert.Empty(t, b.Config.Bundle.ComputeId)

	assert.Len(t, diags, 1)
	assert.Equal(t, "compute_id is deprecated, please use cluster_id instead", diags[0].Summary)
	assert.Equal(t, diag.Warning, diags[0].Severity)
}

func TestComputeIdToClusterIdInTargetOverride(t *testing.T) {
	b := &bundle.Bundle{
		Config: config.Root{
			Targets: map[string]*config.Target{
				"dev": {
					ComputeId: "compute-id-dev",
				},
			},
		},
	}

	diags := bundle.Apply(context.Background(), b, mutator.ComputeIdToClusterId())
	assert.NoError(t, diags.Error())
	assert.Empty(t, b.Config.Targets["dev"].ComputeId)

	diags = diags.Extend(bundle.Apply(context.Background(), b, mutator.SelectTarget("dev")))
	assert.NoError(t, diags.Error())

	assert.Equal(t, "compute-id-dev", b.Config.Bundle.ClusterId)
	assert.Empty(t, b.Config.Bundle.ComputeId)

	assert.Len(t, diags, 1)
	assert.Equal(t, "compute_id is deprecated, please use cluster_id instead", diags[0].Summary)
	assert.Equal(t, diag.Warning, diags[0].Severity)
}
