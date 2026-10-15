package deploy

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/direct/dstate"
	"github.com/databricks/cli/bundle/internal/bundletest"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/testserver"
	"github.com/stretchr/testify/assert"
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

	// Serve the dashboard read by the check.
	w, server := bundletest.NewClients(t)
	calls := 0
	server.Handle("GET", "/api/2.0/lakeview/dashboards/id1", func(testserver.Request) any {
		calls++
		return map[string]string{"dashboard_id": "id1", "display_name": "My Special Dashboard", "etag": "1000"}
	})
	b.SetWorkpaceClient(w)
	defer func() { assert.Equal(t, 1, calls) }()

	// No changes, so no diags.
	diags := bundle.Apply(ctx, b, CheckDashboardsModifiedRemotely(false))
	assert.Empty(t, diags)
}

func TestCheckDashboardsModifiedRemotely_ExistingStateChange(t *testing.T) {
	ctx := t.Context()

	b := mockDashboardBundle(t)
	writeFakeDashboardState(t, b)

	// Serve the dashboard read by the check.
	w, server := bundletest.NewClients(t)
	calls := 0
	server.Handle("GET", "/api/2.0/lakeview/dashboards/id1", func(testserver.Request) any {
		calls++
		return map[string]string{"dashboard_id": "id1", "display_name": "My Special Dashboard", "etag": "1234"}
	})
	b.SetWorkpaceClient(w)
	defer func() { assert.Equal(t, 1, calls) }()

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

	// Serve the dashboard read by the check.
	w, server := bundletest.NewClients(t)
	calls := 0
	server.Handle("GET", "/api/2.0/lakeview/dashboards/id1", func(testserver.Request) any {
		calls++
		return testserver.Response{StatusCode: 500, Body: map[string]string{"error_code": "INTERNAL_ERROR", "message": "failure"}}
	})
	b.SetWorkpaceClient(w)
	defer func() { assert.Equal(t, 1, calls) }()

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

	// Serve the dashboard read by the check.
	w, server := bundletest.NewClients(t)
	calls := 0
	server.Handle("GET", "/api/2.0/lakeview/dashboards/id1", func(testserver.Request) any {
		calls++
		return map[string]string{"dashboard_id": "id1", "display_name": "My Special Dashboard", "etag": "1234"}
	})
	b.SetWorkpaceClient(w)
	defer func() { assert.Equal(t, 1, calls) }()

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
