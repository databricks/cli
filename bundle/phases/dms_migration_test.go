package phases

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMigrateToDMSRejectsTerraformBeforeLock(t *testing.T) {
	b := &bundle.Bundle{}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	MigrateToDMS(ctx, b, &statemgmt.StateDesc{Engine: engine.EngineTerraform})

	require.True(t, logdiag.HasError(ctx))
	diags := logdiag.FlushCollected(ctx)
	require.Len(t, diags, 1)
	assert.Equal(t, ErrDMSRequiresDirectState.Error(), diags[0].Summary)
	assert.Nil(t, b.Locker)
}

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

	MigrateToDMS(ctx, b, state)

	require.False(t, logdiag.HasError(ctx), "%v", logdiag.FlushCollected(ctx))
	assert.True(t, state.IsDMS())
	assert.Equal(t, 2, state.Serial)
	assert.False(t, b.MigratingToDirect)
	assert.Nil(t, b.Locker)
	assert.True(t, b.DeploymentBundle.StateDB.IsOpen())
}

func TestRegisterMigrationState(t *testing.T) {
	tests := []struct {
		name         string
		serial       int
		lastVersion  int
		registration int
	}{
		{
			name:         "first registration",
			serial:       2,
			registration: 3,
		},
		{
			name:         "service ahead",
			serial:       2,
			lastVersion:  5,
			registration: 6,
		},
		{
			name:         "file ahead",
			serial:       8,
			lastVersion:  3,
			registration: 9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statePath := filepath.Join(t.TempDir(), "resources.json")
			var db dstate.DeploymentState
			db.OpenWithData(statePath, dstate.NewDatabase("lineage", tt.serial))
			db.DeploymentID = "123"
			db.VersionID = tt.lastVersion
			m := mocks.NewMockWorkspaceClient(t)
			api := m.GetMockBundleDeploymentsAPI()
			previousVersion := ""
			if tt.lastVersion > 0 {
				previousVersion = strconv.Itoa(tt.lastVersion)
			}
			api.EXPECT().CreateVersion(mock.Anything, bundledeployments.CreateVersionRequest{
				Parent:    dms.DeploymentName("123"),
				VersionId: strconv.Itoa(tt.registration),
				Version: bundledeployments.Version{
					CliVersion:        build.GetInfo().Version,
					VersionType:       dms.VersionTypeDeploy,
					PreviousVersionId: previousVersion,
					Operations:        []bundledeployments.StagedOperation{},
				},
			}).Return(&bundledeployments.Version{}, nil).Once()
			api.EXPECT().CompleteVersion(mock.Anything, bundledeployments.CompleteVersionRequest{
				Name:             dms.VersionName("123", tt.registration),
				CompletionReason: bundledeployments.VersionCompleteVersionCompleteSuccess,
			}).Return(&bundledeployments.Version{}, nil).Once()
			ctx := cmdctx.SetWorkspaceClient(t.Context(), m.WorkspaceClient)

			require.NoError(t, registerMigrationState(ctx, &db))

			assert.True(t, db.IsDeploymentMetadataService())
			assert.Equal(t, tt.registration, db.GetSerial())
			content, err := os.ReadFile(statePath)
			require.NoError(t, err)
			var persisted dstate.Database
			require.NoError(t, json.Unmarshal(content, &persisted))
			assert.Contains(t, persisted.Features, dstate.FeatureDeploymentHistory)
			assert.Equal(t, tt.registration, persisted.Serial)
			assert.Empty(t, persisted.State)
		})
	}
}
