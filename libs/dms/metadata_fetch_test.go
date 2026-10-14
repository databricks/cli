package dms_test

import (
	"testing"

	"github.com/databricks/cli/libs/dms"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestFetchDeploymentFromStatePath(t *testing.T) {
	tests := []struct {
		name        string
		nodeError   error
		recordError error
		wantError   bool
	}{
		{name: "confirmed deployment"},
		{name: "missing node", nodeError: apierr.ErrNotFound},
		{name: "absent node", nodeError: apierr.ErrResourceDoesNotExist},
		{name: "node permission denied", nodeError: apierr.ErrPermissionDenied, wantError: true},
		{name: "missing service record", recordError: apierr.ErrNotFound, wantError: true},
		{name: "service permission denied", recordError: apierr.ErrPermissionDenied, wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := mocks.NewMockWorkspaceClient(t)
			m.GetMockWorkspaceAPI().EXPECT().GetStatusByPath(mock.Anything, "/Workspace/state/"+dms.DeploymentNodeName).
				Return(&workspace.ObjectInfo{ObjectId: 123}, tt.nodeError).Once()
			deployment := &bundledeployments.Deployment{Name: dms.DeploymentName("123"), LastVersionId: "7"}
			if tt.nodeError == nil {
				m.GetMockBundleDeploymentsAPI().EXPECT().GetDeployment(mock.Anything, bundledeployments.GetDeploymentRequest{Name: deployment.Name}).
					Return(deployment, tt.recordError).Once()
			}
			id, found, err := dms.FetchDeploymentFromStatePath(t.Context(), m.WorkspaceClient, "/Workspace/state")
			if tt.wantError {
				require.Error(t, err)
				if tt.nodeError != nil {
					assert.ErrorIs(t, err, tt.nodeError)
				} else {
					assert.ErrorIs(t, err, tt.recordError)
				}
				return
			}
			require.NoError(t, err)
			if tt.nodeError != nil {
				assert.Empty(t, id)
				assert.Nil(t, found)
				return
			}
			assert.Equal(t, "123", id)
			assert.Same(t, deployment, found)
		})
	}
}
