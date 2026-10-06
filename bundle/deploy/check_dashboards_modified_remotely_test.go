package deploy

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/dashboards"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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
							DisplayName: "My Special Dashboard",
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

	diags := bundle.Apply(t.Context(), b, CheckDashboardsModifiedRemotely(false))
	assert.Empty(t, diags)
}

func TestCheckDashboardsModifiedRemotely_FirstDeployment(t *testing.T) {
	b := mockDashboardBundle(t)
	openDashboardState(t, b, dstate.NewDatabase("lineage", 1))
	diags := bundle.Apply(t.Context(), b, CheckDashboardsModifiedRemotely(false))
	assert.Empty(t, diags)
}

func TestCheckDashboardsModifiedRemotely_ExistingStateNoChange(t *testing.T) {
	ctx := t.Context()

	b := mockDashboardBundle(t)
	writeFakeDashboardState(t, b)

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
	diags := bundle.Apply(ctx, b, CheckDashboardsModifiedRemotely(false))
	assert.Empty(t, diags)
}

func TestCheckDashboardsModifiedRemotely_ExistingStateChange(t *testing.T) {
	ctx := t.Context()

	b := mockDashboardBundle(t)
	writeFakeDashboardState(t, b)

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
	diags := bundle.Apply(ctx, b, CheckDashboardsModifiedRemotely(false))
	if assert.Len(t, diags, 1) {
		assert.Equal(t, diag.Error, diags[0].Severity)
		assert.Equal(t, `dashboard "dash1" has been modified remotely`, diags[0].Summary)
	}
}

func TestCheckDashboardsModifiedRemotely_ExistingStateFailureToGet(t *testing.T) {
	ctx := t.Context()

	b := mockDashboardBundle(t)
	writeFakeDashboardState(t, b)

	// Mock the call to the API.
	m := mocks.NewMockWorkspaceClient(t)
	dashboardsAPI := m.GetMockLakeviewAPI()
	dashboardsAPI.EXPECT().
		GetByDashboardId(mock.Anything, "id1").
		Return(nil, errors.New("failure")).
		Once()
	b.SetWorkpaceClient(m.WorkspaceClient)

	// Unable to get the dashboard, so expect an error.
	diags := bundle.Apply(ctx, b, CheckDashboardsModifiedRemotely(false))
	if assert.Len(t, diags, 1) {
		assert.Equal(t, diag.Error, diags[0].Severity)
		assert.Equal(t, `failed to get dashboard "dash1"`, diags[0].Summary)
	}
}

func TestCheckDashboardsModifiedRemotely_ExistingStateChangePlanMode(t *testing.T) {
	ctx := t.Context()

	b := mockDashboardBundle(t)
	writeFakeDashboardState(t, b)

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

	// The dashboard has changed, but in plan mode expect a warning instead of an error.
	diags := bundle.Apply(ctx, b, CheckDashboardsModifiedRemotely(true))
	if assert.Len(t, diags, 1) {
		assert.Equal(t, diag.Warning, diags[0].Severity)
		assert.Equal(t, `dashboard "dash1" has been modified remotely`, diags[0].Summary)
	}
}

func openDashboardState(t *testing.T, b *bundle.Bundle, data dstate.Database) {
	b.DeploymentBundle.StateDB.OpenWithData(filepath.Join(t.TempDir(), "resources.json"), data)
}

func writeFakeDashboardState(t *testing.T, b *bundle.Bundle) {
	data := dstate.NewDatabase("lineage", 1)
	data.State["resources.dashboards.dash1"] = dstate.ResourceEntry{ID: "id1", State: json.RawMessage(`{"etag": "1000"}`)}
	data.State["resources.jobs.job"] = dstate.ResourceEntry{ID: "1234", State: json.RawMessage(`{}`)}
	data.State["resources.dashboards.dash2"] = dstate.ResourceEntry{ID: "id2", State: json.RawMessage(`{"etag": "1001"}`)}
	openDashboardState(t, b, data)
}
