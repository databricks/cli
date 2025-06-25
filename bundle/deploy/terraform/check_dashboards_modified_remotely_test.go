package terraform

import (
	"context"
	"errors"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func mockDashboardBundle(t *testing.T) *bundle.Bundle {
	dir := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: dir,
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "test",
			},
			Resources: config.Resources{
				Dashboards: map[string]*resources.Dashboard{
					"dash1": {
						DashboardConfig: resources.DashboardConfig{
							Dashboard: dashboards.Dashboard{
								DisplayName: "My Special Dashboard",
							},
						},
					},
				},
			},
		},
	}
	return b
}

func TestCheckDashboardsModifiedRemotely_NoDashboards(t *testing.T) {
	dir := t.TempDir()
	b := &bundle.Bundle{
		BundleRootPath: dir,
		Config: config.Root{
			Bundle: config.Bundle{
				Target: "test",
			},
			Resources: config.Resources{},
		},
	}

	diags := bundle.Apply(context.Background(), b, CheckDashboardsModifiedRemotely())
	assert.Empty(t, diags)
}

func TestCheckDashboardsModifiedRemotely_FirstDeployment(t *testing.T) {
	b := mockDashboardBundle(t)
	diags := bundle.Apply(context.Background(), b, CheckDashboardsModifiedRemotely())
	assert.Empty(t, diags)
}

func TestCheckDashboardsModifiedRemotely_ExistingStateNoChange(t *testing.T) {
	ctx := context.Background()

	b := mockDashboardBundle(t)
	writeFakeDashboardState(t, ctx, b)

	// Mock the call to the API.
	m := mocks.NewMockWorkspaceClient(t)
	dashboardsAPI := m.GetMockLakeviewAPI()
	dashboardsAPI.EXPECT().
		GetByDashboardId(mock.Anything, "id1").
		Return(&dashboards.Dashboard{
			DisplayName: "My Special Dashboard",
			Etag:        "1000",
		}, nil).
		Once()
	b.SetWorkpaceClient(m.WorkspaceClient)

	// No changes, so no diags.
	diags := bundle.Apply(ctx, b, CheckDashboardsModifiedRemotely())
	assert.Empty(t, diags)
}

func TestCheckDashboardsModifiedRemotely_ExistingStateChange(t *testing.T) {
	ctx := context.Background()

	b := mockDashboardBundle(t)
	writeFakeDashboardState(t, ctx, b)

	// Mock the call to the API.
	m := mocks.NewMockWorkspaceClient(t)
	dashboardsAPI := m.GetMockLakeviewAPI()
	dashboardsAPI.EXPECT().
		GetByDashboardId(mock.Anything, "id1").
		Return(&dashboards.Dashboard{
			DisplayName: "My Special Dashboard",
			Etag:        "1234",
		}, nil).
		Once()
	b.SetWorkpaceClient(m.WorkspaceClient)

	// The dashboard has changed, so expect an error.
	diags := bundle.Apply(ctx, b, CheckDashboardsModifiedRemotely())
	if assert.Len(t, diags, 1) {
		assert.Equal(t, diag.Error, diags[0].Severity)
		assert.Equal(t, `dashboard "dash1" has been modified remotely`, diags[0].Summary)
	}
}

func TestCheckDashboardsModifiedRemotely_ExistingStateFailureToGet(t *testing.T) {
	ctx := context.Background()

	b := mockDashboardBundle(t)
	writeFakeDashboardState(t, ctx, b)

	// Mock the call to the API.
	m := mocks.NewMockWorkspaceClient(t)
	dashboardsAPI := m.GetMockLakeviewAPI()
	dashboardsAPI.EXPECT().
		GetByDashboardId(mock.Anything, "id1").
		Return(nil, errors.New("failure")).
		Once()
	b.SetWorkpaceClient(m.WorkspaceClient)

	// Unable to get the dashboard, so expect an error.
	diags := bundle.Apply(ctx, b, CheckDashboardsModifiedRemotely())
	if assert.Len(t, diags, 1) {
		assert.Equal(t, diag.Error, diags[0].Severity)
		assert.Equal(t, `failed to get dashboard "dash1"`, diags[0].Summary)
	}
}

func writeFakeDashboardState(t *testing.T, ctx context.Context, b *bundle.Bundle) {
	path, err := b.StateLocalPath(ctx)
	require.NoError(t, err)

	// Write fake state file.
	testutil.WriteFile(t, path, `
    {
      "version": 4,
      "terraform_version": "1.5.5",
      "resources": [
        {
          "mode": "managed",
          "type": "databricks_dashboard",
          "name": "dash1",
          "instances": [
            {
              "schema_version": 0,
              "attributes": {
                "etag": "1000",
                "id": "id1"
              }
            }
          ]
        },
        {
          "mode": "managed",
          "type": "databricks_job",
          "name": "job",
          "instances": [
            {
              "schema_version": 0,
              "attributes": {
                "id": "1234"
              }
            }
          ]
        },
        {
          "mode": "managed",
          "type": "databricks_dashboard",
          "name": "dash2",
          "instances": [
            {
              "schema_version": 0,
              "attributes": {
                "etag": "1001",
                "id": "id2"
              }
            }
          ]
        }
      ]
    }
	`)
}
