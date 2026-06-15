package statemgmt

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	sdkbundle "github.com/databricks/databricks-sdk-go/service/bundle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// newTestBundle returns a Bundle wired with a mock workspace client and the
// workspace fields that StateFilenameDirect needs (CurrentUser, RootPath).
func newTestBundle(t *testing.T) (*bundle.Bundle, *mocks.MockWorkspaceClient) {
	b := &bundle.Bundle{
		BundleRootPath: t.TempDir(),
		Config: config.Root{
			Bundle: config.Bundle{Target: "default"},
			Workspace: config.Workspace{
				StatePath: "/Workspace/Users/me@example.com/.bundle/test/default/state",
			},
		},
	}
	m := mocks.NewMockWorkspaceClient(t)
	b.SetWorkpaceClient(m.WorkspaceClient)
	return b, m
}

// TestLoadStateFromDMS_NoDeploymentID covers the "DMS enabled but the
// workspace has no managed_service.json yet" case: the state DB should be
// initialised empty (no panic on later ExportState) and no API call should be
// made.
func TestLoadStateFromDMS_NoDeploymentID(t *testing.T) {
	b, _ := newTestBundle(t)
	// b.DeploymentID stays empty.

	err := LoadStateFromDMS(t.Context(), b)
	require.NoError(t, err)

	// State is initialised but empty — Export should succeed and return no entries.
	got := b.DeploymentBundle.ExportState(t.Context())
	assert.Empty(t, got)
}

// TestLoadStateFromDMS_PopulatesFromList confirms that resources reported by
// ListResources land in the in-memory state DB under the fully-qualified
// "resources.<key>" form, and that per-resource State payloads are preserved
// verbatim.
func TestLoadStateFromDMS_PopulatesFromList(t *testing.T) {
	b, m := newTestBundle(t)
	b.DeploymentID = "dep-123"

	jobState := json.RawMessage(`{"name":"my-job","max_concurrent_runs":1}`)
	mockBundle := m.GetMockBundleAPI()
	mockBundle.EXPECT().
		ListResourcesAll(mock.Anything, sdkbundle.ListResourcesRequest{
			Parent: "deployments/dep-123",
		}).
		Return([]sdkbundle.Resource{
			{ResourceKey: "jobs.foo", ResourceId: "1001", State: &jobState},
			// State omitted — exercises the nil-state path.
			{ResourceKey: "pipelines.bar", ResourceId: "p-1"},
		}, nil)

	err := LoadStateFromDMS(t.Context(), b)
	require.NoError(t, err)

	job, ok := b.DeploymentBundle.StateDB.GetResourceEntry("resources.jobs.foo")
	require.True(t, ok)
	assert.Equal(t, "1001", job.ID)
	assert.JSONEq(t, string(jobState), string(job.State))

	pipeline, ok := b.DeploymentBundle.StateDB.GetResourceEntry("resources.pipelines.bar")
	require.True(t, ok)
	assert.Equal(t, "p-1", pipeline.ID)
	assert.Empty(t, pipeline.State)
}

// TestLoadStateFromDMS_ListError checks that an underlying API failure is
// wrapped and surfaced rather than swallowed (otherwise the deploy would
// proceed against an empty in-memory view and treat everything as a create).
func TestLoadStateFromDMS_ListError(t *testing.T) {
	b, m := newTestBundle(t)
	b.DeploymentID = "dep-456"

	mockBundle := m.GetMockBundleAPI()
	mockBundle.EXPECT().
		ListResourcesAll(mock.Anything, mock.Anything).
		Return(nil, errors.New("boom"))

	err := LoadStateFromDMS(t.Context(), b)
	require.Error(t, err)
	assert.ErrorContains(t, err, "boom")
	assert.ErrorContains(t, err, "failed to list resources from deployment metadata service")
}
