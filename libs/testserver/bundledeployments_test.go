package testserver_test

import (
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateVersionAcceptsFeatureResource(t *testing.T) {
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	client, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)
	ctx := t.Context()
	const statePath = "/Workspace/dms-feature-test"
	require.NoError(t, client.Workspace.MkdirsByPath(ctx, statePath))
	deployment, err := client.BundleDeployments.CreateDeployment(ctx, bundledeployments.CreateDeploymentRequest{
		Deployment: bundledeployments.Deployment{InitialParentPath: statePath},
	})
	require.NoError(t, err)

	version, err := client.BundleDeployments.CreateVersion(ctx, bundledeployments.CreateVersionRequest{
		Parent:    deployment.Name,
		VersionId: "1",
		Version: bundledeployments.Version{
			VersionType: bundledeployments.VersionTypeVersionTypeDeploy,
			Operations: []bundledeployments.StagedOperation{{
				ResourceKey: "features.feature1",
				ActionType:  bundledeployments.OperationActionTypeOperationActionTypeBind,
			}},
		},
	})
	require.NoError(t, err)
	operations, err := client.BundleDeployments.ListOperationsAll(ctx, bundledeployments.ListOperationsRequest{Parent: version.Name})
	require.NoError(t, err)
	require.Len(t, operations, 1)
	assert.Equal(t, "features.feature1", operations[0].ResourceKey)
	assert.Equal(t, bundledeployments.OperationActionTypeOperationActionTypeBind, operations[0].ActionType)
}
