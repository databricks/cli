package phases_test

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/phases"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrateToDMSAlreadyRecorded(t *testing.T) {
	b := &bundle.Bundle{
		BundleRootPath:    t.TempDir(),
		MigratingToDirect: true,
		Config: config.Root{
			Bundle: config.Bundle{
				Target:     "default",
				Deployment: config.Deployment{Lock: config.Lock{Enabled: new(true)}},
			},
			Workspace: config.Workspace{StatePath: "/Workspace/state"},
		},
	}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	b.DeploymentBundle.StateDB.OpenWithData(filepath.Join(b.BundleRootPath, "resources.json"), dstate.NewDatabase("lineage", 2))
	state := &statemgmt.StateDesc{
		Serial:   2,
		Features: map[string]struct{}{dstate.FeatureDeploymentHistory: {}},
	}

	phases.MigrateToDMS(ctx, b, state)

	require.False(t, logdiag.HasError(ctx), "%v", logdiag.FlushCollected(ctx))
	assert.True(t, state.IsDMS())
	assert.Equal(t, 2, state.Serial)
	assert.False(t, b.MigratingToDirect)
	assert.Nil(t, b.Locker)
	assert.True(t, b.DeploymentBundle.StateDB.IsOpen())
}
