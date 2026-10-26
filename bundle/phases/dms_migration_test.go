package phases

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/apierr"
	sdkconfig "github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMigrateToFileHistoryRejectsTerraformBeforeLock(t *testing.T) {
	// 1. Try migration with Terraform state and locking enabled by default.
	b := &bundle.Bundle{}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	MigrateToFileHistory(ctx, b, &statemgmt.StateDesc{Engine: engine.EngineTerraform})

	// 2. Reject the state before creating a deployment lock.
	require.True(t, logdiag.HasError(ctx))
	assert.Nil(t, b.Locker, "Terraform state must be rejected before locking")
}

func TestMigrateToFileHistorySkipsAlreadyRecordedBeforeLock(t *testing.T) {
	// 1. File state already records history; no open database or client is needed.
	b := &bundle.Bundle{}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	state := &statemgmt.StateDesc{
		Features: map[string]struct{}{dstate.FeatureDeploymentHistoryShadow: {}},
	}

	MigrateToFileHistory(ctx, b, state)

	// 2. Return successfully without acquiring a migration lock.
	require.False(t, logdiag.HasError(ctx), "%v", logdiag.FlushCollected(ctx))
	assert.Nil(t, b.Locker, "Already-recorded state must not acquire a migration lock")
}

func TestRegisterMigrationStateVersionNumbers(t *testing.T) {
	// Acceptance tests cover registration and persistence. These cases cover unequal counters.
	tests := []struct {
		name           string
		stateSerial    int
		serviceVersion int
		wantVersion    int
	}{
		{
			name:           "service version is newer",
			stateSerial:    2,
			serviceVersion: 5,
			wantVersion:    6,
		},
		{
			name:           "state serial is newer",
			stateSerial:    8,
			serviceVersion: 3,
			wantVersion:    9,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 1. Open state with deliberately different file and service counters.
			statePath := filepath.Join(t.TempDir(), "resources.json")
			var db dstate.DeploymentState
			db.OpenWithData(statePath, dstate.NewDatabase("lineage", tt.stateSerial))
			db.Data.State["resources.jobs.job"] = dstate.ResourceEntry{ID: "123"}
			db.DeploymentID = "123"
			db.VersionID = tt.serviceVersion

			// 2. Registration uses max(file, service)+1, but its predecessor is the service version.
			m := mocks.NewMockWorkspaceClient(t)
			api := m.GetMockBundleDeploymentsAPI()
			api.EXPECT().CreateVersion(mock.Anything, mock.Anything).
				Run(func(_ context.Context, request bundledeployments.CreateVersionRequest) {
					assert.Equal(t, strconv.Itoa(tt.wantVersion), request.VersionId, "Migration must follow the larger counter")
					assert.Equal(t, strconv.Itoa(tt.serviceVersion), request.Version.PreviousVersionId, "Predecessor must be the actual DMS version")
				}).Return(&bundledeployments.Version{}, nil).Once()
			api.EXPECT().UpdateOperation(mock.Anything, mock.Anything).
				Return(&bundledeployments.Operation{SequenceId: 1}, nil).Once()
			api.EXPECT().CompleteVersion(mock.Anything, mock.Anything).Return(&bundledeployments.Version{}, nil).Once()
			ctx := cmdctx.SetWorkspaceClient(t.Context(), m.WorkspaceClient)

			// 3. Run registration; the mock checks the version numbers above.
			version, err := registerMigrationState(ctx, &db)
			require.NoError(t, err)
			assert.Equal(t, tt.wantVersion, version)
		})
	}
}

func TestMigrateToFileHistoryRejectsNewerRemoteState(t *testing.T) {
	b := &bundle.Bundle{BundleRootPath: t.TempDir()}
	b.Config.Bundle.Target = "default"
	b.Config.Bundle.Deployment.Lock.Enabled = new(false)
	b.Config.Workspace.StatePath = "/state"
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	_, localPath := b.StateFilenameDirect(ctx)
	b.DeploymentBundle.StateDB.OpenWithData(localPath, dstate.NewDatabase("lineage", 1))

	// The command opened serial 1, but another deploy committed serial 2 before migration.
	newer := `{"lineage":"lineage","serial":2,"state":{"resources.jobs.new":{"__id__":"new-id"}}}`
	m := mocks.NewMockWorkspaceClient(t)
	m.WorkspaceClient.Config = &sdkconfig.Config{Host: "https://workspace.test", Token: "token"}
	m.GetMockWorkspaceAPI().EXPECT().Download(mock.Anything, "/state/resources.json").
		Return(io.NopCloser(strings.NewReader(newer)), nil).Once()
	m.GetMockWorkspaceAPI().EXPECT().Download(mock.Anything, "/state/terraform.tfstate").
		Return(nil, apierr.ErrNotFound).Once()
	b.SetWorkpaceClient(m.WorkspaceClient)

	MigrateToFileHistory(ctx, b, &statemgmt.StateDesc{Engine: engine.EngineDirect})

	// No DMS writes are expected. Keep the newer file; never replace it with serial 1.
	require.True(t, logdiag.HasError(ctx))
	assert.Contains(t, fmt.Sprint(logdiag.FlushCollected(ctx)), "deployment state changed before enabling history")
	content, err := os.ReadFile(localPath)
	require.NoError(t, err)
	assert.JSONEq(t, newer, string(content))
}
