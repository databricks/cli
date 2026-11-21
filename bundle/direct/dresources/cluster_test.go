package dresources

import (
	"context"
	"testing"
	"time"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/experimental/mocks"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestClusterOverrideChangeDescLibraries(t *testing.T) {
	jobLib := []compute.Library{{Whl: "/Workspace/job.whl"}}

	tests := []struct {
		name   string
		path   string
		change ChangeDesc
		action deployplan.ActionType
		reason string
	}{
		{
			name:   "no section with remote libraries is unmanaged",
			path:   "libraries",
			change: ChangeDesc{Action: deployplan.Update, Old: nil, New: nil, Remote: jobLib},
			action: deployplan.Skip,
			reason: deployplan.ReasonUnmanaged,
		},
		{
			name:   "removed section is unmanaged",
			path:   "libraries",
			change: ChangeDesc{Action: deployplan.Update, Old: jobLib, New: nil, Remote: jobLib},
			action: deployplan.Skip,
			reason: deployplan.ReasonUnmanaged,
		},
		{
			name:   "empty list stays managed",
			path:   "libraries",
			change: ChangeDesc{Action: deployplan.Update, Old: nil, New: []compute.Library{}, Remote: jobLib},
			action: deployplan.Update,
		},
		{
			name:   "declared libraries stay managed",
			path:   "libraries",
			change: ChangeDesc{Action: deployplan.Update, Old: nil, New: jobLib, Remote: nil},
			action: deployplan.Update,
		},
		{
			// A per-library entry only exists when the config declares a list, so a missing New
			// there is a removal to apply, not an unmanaged section.
			name:   "element removal stays managed",
			path:   "libraries[whl='/Workspace/job.whl']",
			change: ChangeDesc{Action: deployplan.Update, Old: nil, New: nil, Remote: jobLib[0]},
			action: deployplan.Update,
		},
	}

	r := &ResourceCluster{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			change := tt.change
			require.NoError(t, r.OverrideChangeDesc(t.Context(), structpath.MustParsePath(tt.path), &change, nil))
			assert.Equal(t, tt.action, change.Action)
			assert.Equal(t, tt.reason, change.Reason)
		})
	}
}

const clusterDoReadID = "0101-000000-deleted00"

// doReadOrdered runs DoRead with both lookups in flight. The lookup named by
// first returns immediately. The other waits until that call has started and
// then long enough for its result to be the one recorded first.
func doReadOrdered(t *testing.T, id string, details *compute.ClusterDetails, clusterErr error, statuses *compute.ClusterLibraryStatuses, libraryErr error, first string) (*ClusterRemote, error) {
	t.Helper()
	m := mocks.NewMockWorkspaceClient(t)
	r := &ResourceCluster{client: m.WorkspaceClient}

	winnerStarted := make(chan struct{})
	release := func(name string) {
		if name == first {
			close(winnerStarted)
			return
		}
		select {
		case <-winnerStarted:
		case <-t.Context().Done():
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	m.GetMockClustersAPI().EXPECT().
		GetByClusterId(mock.Anything, id).
		RunAndReturn(func(context.Context, string) (*compute.ClusterDetails, error) {
			release("cluster")
			return details, clusterErr
		}).Once()
	m.GetMockLibrariesAPI().EXPECT().
		ClusterStatusByClusterId(mock.Anything, id).
		RunAndReturn(func(context.Context, string) (*compute.ClusterLibraryStatuses, error) {
			release("libraries")
			return statuses, libraryErr
		}).Once()

	return r.DoRead(t.Context(), id)
}

func TestClusterDoReadDeletedClusterIgnoresLibraryStatusOrder(t *testing.T) {
	clusterMissing := &apierr.APIError{
		StatusCode: 400,
		ErrorCode:  "RESOURCE_DOES_NOT_EXIST",
		Message:    "Cluster " + clusterDoReadID + " does not exist",
	}
	// library-status uses this code for the same missing cluster. The SDK remaps
	// it to "not found" only on GET /clusters/get, so IsMissing is false here.
	libraryMissing := &apierr.APIError{
		StatusCode: 400,
		ErrorCode:  "INVALID_PARAMETER_VALUE",
		Message:    "Cluster " + clusterDoReadID + " does not exist",
	}

	for _, first := range []string{"libraries", "cluster"} {
		t.Run(first+" finishes first", func(t *testing.T) {
			_, err := doReadOrdered(t, clusterDoReadID, nil, clusterMissing, nil, libraryMissing, first)
			require.ErrorIs(t, err, apierr.ErrResourceDoesNotExist)
			require.NotErrorIs(t, err, apierr.ErrInvalidParameterValue)
		})
	}
}

func TestClusterDoReadLibraryErrorSurfacesWhenClusterExists(t *testing.T) {
	details := &compute.ClusterDetails{ClusterId: clusterDoReadID, State: compute.StateRunning}
	libraryErr := &apierr.APIError{
		StatusCode: 500,
		ErrorCode:  "UNKNOWN",
		Message:    "library status unavailable",
	}

	for _, first := range []string{"libraries", "cluster"} {
		t.Run(first+" finishes first", func(t *testing.T) {
			_, err := doReadOrdered(t, clusterDoReadID, details, nil, nil, libraryErr, first)
			require.ErrorIs(t, err, libraryErr)
			require.NotErrorIs(t, err, apierr.ErrNotFound)
		})
	}
}

func TestClusterDoReadReturnsInstalledLibraries(t *testing.T) {
	details := &compute.ClusterDetails{
		ClusterId: clusterDoReadID,
		State:     compute.StateRunning,
		Spec:      &compute.ClusterSpec{ApplyPolicyDefaultValues: true},
	}
	statuses := &compute.ClusterLibraryStatuses{
		LibraryStatuses: []compute.LibraryFullStatus{
			{Library: &compute.Library{Whl: "/Workspace/a.whl"}, Status: compute.LibraryInstallStatusInstalled},
			{Library: &compute.Library{Jar: "/Workspace/all.jar"}, IsLibraryForAllClusters: true, Status: compute.LibraryInstallStatusInstalled},
		},
	}

	remote, err := doReadOrdered(t, clusterDoReadID, details, nil, statuses, nil, "libraries")
	require.NoError(t, err)
	require.NotNil(t, remote.Lifecycle)
	require.NotNil(t, remote.Lifecycle.Started)
	assert.True(t, *remote.Lifecycle.Started)
	assert.True(t, remote.ApplyPolicyDefaultValues)
	require.Len(t, remote.Libraries, 1)
	assert.Equal(t, "/Workspace/a.whl", remote.Libraries[0].Whl)
}
