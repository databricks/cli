package dmsmigration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/databricks/cli/bundle/deploy/dmsmigration"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestMigrate(t *testing.T) {
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

			require.NoError(t, dmsmigration.Migrate(ctx, &db))

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
