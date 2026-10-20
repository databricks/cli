package phases

import (
	"context"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/engine"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/statemgmt"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/logdiag"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMigrateToDMSRejectsTerraformBeforeLock(t *testing.T) {
	// 1. Try migration with Terraform state and locking enabled by default.
	b := &bundle.Bundle{}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)

	MigrateToDMS(ctx, b, &statemgmt.StateDesc{Engine: engine.EngineTerraform})

	// 2. Reject the state before creating a deployment lock.
	require.True(t, logdiag.HasError(ctx))
	assert.Nil(t, b.Locker, "Terraform state must be rejected before locking")
}

func TestMigrateToDMSSkipsAlreadyRecordedBeforeLock(t *testing.T) {
	// 1. State already uses DMS; no open database or workspace client is needed.
	b := &bundle.Bundle{}
	ctx := logdiag.InitContext(t.Context())
	logdiag.SetCollect(ctx, true)
	state := &statemgmt.StateDesc{
		Features: map[string]struct{}{dstate.FeatureDeploymentHistory: {}},
	}

	MigrateToDMS(ctx, b, state)

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
			api.EXPECT().CompleteVersion(mock.Anything, mock.Anything).Return(&bundledeployments.Version{}, nil).Once()
			ctx := cmdctx.SetWorkspaceClient(t.Context(), m.WorkspaceClient)

			// 3. Run registration; the mock checks the version numbers above.
			require.NoError(t, registerMigrationState(ctx, &db))
		})
	}
}
