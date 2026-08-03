package dms

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testStatePath is the bundle state directory the recorder registers the
// deployment node under; several tests assert it round-trips to the service.
const testStatePath = "/Workspace/Users/me/.bundle/proj/dev/state"

// fakeDMS records the calls the recorder makes and lets a test script the
// server-side responses. It embeds the SDK interface so it satisfies it while
// only overriding the methods the recorder uses.
type fakeDMS struct {
	bundledeployments.BundleDeploymentsInterface

	// scripted behavior
	getDeployment func(id string) (*bundledeployments.Deployment, error)

	// assigned deployment ID for CreateDeployment (server-generated flow)
	assignedID string

	// captured requests
	created   []bundledeployments.CreateDeploymentRequest
	versions  []bundledeployments.CreateVersionRequest
	completed []bundledeployments.CompleteVersionRequest
	deleted   []string
}

func (f *fakeDMS) CreateDeployment(ctx context.Context, req bundledeployments.CreateDeploymentRequest) (*bundledeployments.Deployment, error) {
	f.created = append(f.created, req)
	// The server always assigns the ID; it is the ID of the workspace node it
	// creates under initial_parent_path.
	return &bundledeployments.Deployment{Name: "deployments/" + f.assignedID}, nil
}

func (f *fakeDMS) GetDeployment(ctx context.Context, req bundledeployments.GetDeploymentRequest) (*bundledeployments.Deployment, error) {
	id := req.Name[len("deployments/"):]
	return f.getDeployment(id)
}

func (f *fakeDMS) CreateVersion(ctx context.Context, req bundledeployments.CreateVersionRequest) (*bundledeployments.Version, error) {
	f.versions = append(f.versions, req)
	return &bundledeployments.Version{VersionId: req.VersionId}, nil
}

func (f *fakeDMS) CompleteVersion(ctx context.Context, req bundledeployments.CompleteVersionRequest) (*bundledeployments.Version, error) {
	f.completed = append(f.completed, req)
	return &bundledeployments.Version{}, nil
}

func (f *fakeDMS) DeleteDeployment(ctx context.Context, req bundledeployments.DeleteDeploymentRequest) error {
	f.deleted = append(f.deleted, req.Name)
	return nil
}

func (f *fakeDMS) Heartbeat(ctx context.Context, req bundledeployments.HeartbeatRequest) (*bundledeployments.HeartbeatResponse, error) {
	return &bundledeployments.HeartbeatResponse{}, nil
}

func TestRecorderFirstDeployCreatesDeploymentWithServerAssignedID(t *testing.T) {
	f := &fakeDMS{assignedID: "server-generated-id"}
	// A first deploy resolves no deployment ID from the workspace.
	r := NewRecorder(f, "", testStatePath, "dev", VersionTypeDeploy, nil)

	require.NoError(t, r.CreateVersion(t.Context()))

	// The server assigned the ID, and the recorder exposes it for the rest of the
	// deploy (it parents the operations recorded under this version).
	require.Len(t, f.created, 1)
	assert.Equal(t, "server-generated-id", r.DeploymentID())
	// initial_parent_path is required: the service creates the deployment node
	// under it, and that node is what ResolveDeploymentID looks up later.
	assert.Equal(t, testStatePath, f.created[0].Deployment.InitialParentPath)

	// The first version is 1, parented under the assigned deployment.
	require.Len(t, f.versions, 1)
	assert.Equal(t, "1", f.versions[0].VersionId)
	assert.Equal(t, "deployments/server-generated-id", f.versions[0].Parent)
	assert.Empty(t, f.versions[0].Version.PreviousVersionId)
	assert.Equal(t, int64(1), r.Version())

	require.NoError(t, r.CompleteVersion(t.Context(), true))
	require.Len(t, f.completed, 1)
	assert.Equal(t, bundledeployments.VersionCompleteVersionCompleteSuccess, f.completed[0].CompletionReason)
	assert.Empty(t, f.deleted)
}

func TestRecorderSubsequentDeployReusesDeploymentAndIncrementsVersion(t *testing.T) {
	f := &fakeDMS{
		getDeployment: func(id string) (*bundledeployments.Deployment, error) {
			return &bundledeployments.Deployment{Name: "deployments/" + id, LastVersionId: "4"}, nil
		},
	}
	// A subsequent deploy passes the stored deployment ID.
	r := NewRecorder(f, "stored-id", testStatePath, "dev", VersionTypeDeploy, nil)

	require.NoError(t, r.CreateVersion(t.Context()))

	// No new deployment is created; the version increments to last_version_id + 1
	// and names its predecessor for the server's concurrency check.
	assert.Empty(t, f.created)
	require.Len(t, f.versions, 1)
	assert.Equal(t, "5", f.versions[0].VersionId)
	assert.Equal(t, "4", f.versions[0].Version.PreviousVersionId)
	assert.Equal(t, "stored-id", r.DeploymentID())
}

func TestRecorderRecordsGitInfo(t *testing.T) {
	f := &fakeDMS{
		getDeployment: func(id string) (*bundledeployments.Deployment, error) {
			return &bundledeployments.Deployment{Name: "deployments/" + id, LastVersionId: "1"}, nil
		},
	}
	git := &bundledeployments.GitInfo{
		OriginUrl: "https://github.com/pavloKozlov/bundle-test.git",
		Branch:    "main",
		Commit:    "e62c64503a29f91af22ec6544ae9610490ad1fce",
	}
	r := NewRecorder(f, "stored-id", testStatePath, "dev", VersionTypeDeploy, git)

	require.NoError(t, r.CreateVersion(t.Context()))
	require.Len(t, f.versions, 1)
	assert.Equal(t, git, f.versions[0].Version.GitInfo)
}

func TestRecorderGitInfoNilWhenAbsent(t *testing.T) {
	f := &fakeDMS{
		getDeployment: func(id string) (*bundledeployments.Deployment, error) {
			return &bundledeployments.Deployment{Name: "deployments/" + id, LastVersionId: "1"}, nil
		},
	}
	r := NewRecorder(f, "stored-id", testStatePath, "dev", VersionTypeDeploy, nil)

	require.NoError(t, r.CreateVersion(t.Context()))
	require.Len(t, f.versions, 1)
	assert.Nil(t, f.versions[0].Version.GitInfo)
}

func TestRecorderGetDeploymentErrorFailsDeploy(t *testing.T) {
	f := &fakeDMS{
		getDeployment: func(id string) (*bundledeployments.Deployment, error) {
			return nil, errors.New("boom")
		},
	}
	r := NewRecorder(f, "stored-id", testStatePath, "dev", VersionTypeDeploy, nil)

	err := r.CreateVersion(t.Context())
	assert.ErrorContains(t, err, "failed to get deployment")
	assert.Empty(t, f.created)
}

func TestRecorderMissingDeploymentRecordStartsAtVersionOne(t *testing.T) {
	// The record is created by the first version, so a node can name a deployment
	// that has none yet - an earlier deploy registered it and then failed. Record
	// version 1 under that same ID instead of creating a second deployment, which
	// would collide on the node path.
	f := &fakeDMS{
		getDeployment: func(id string) (*bundledeployments.Deployment, error) {
			return nil, fmt.Errorf("deployment: %w", apierr.ErrNotFound)
		},
	}
	r := NewRecorder(f, "stored-id", testStatePath, "dev", VersionTypeDeploy, nil)

	require.NoError(t, r.CreateVersion(t.Context()))
	assert.Empty(t, f.created)
	require.Len(t, f.versions, 1)
	assert.Equal(t, "1", f.versions[0].VersionId)
	assert.Equal(t, "deployments/stored-id", f.versions[0].Parent)
	// The first version has no predecessor.
	assert.Empty(t, f.versions[0].Version.PreviousVersionId)
}

func TestRecorderDestroyDeletesDeploymentOnSuccess(t *testing.T) {
	f := &fakeDMS{
		getDeployment: func(id string) (*bundledeployments.Deployment, error) {
			return &bundledeployments.Deployment{Name: "deployments/" + id, LastVersionId: "2"}, nil
		},
	}
	r := NewRecorder(f, "stored-id", testStatePath, "dev", VersionTypeDestroy, nil)

	require.NoError(t, r.CreateVersion(t.Context()))
	assert.Equal(t, bundledeployments.VersionTypeVersionTypeDestroy, f.versions[0].Version.VersionType)

	require.NoError(t, r.CompleteVersion(t.Context(), true))
	// A successful destroy deletes the deployment record.
	require.Equal(t, []string{"deployments/stored-id"}, f.deleted)
}

func TestRecorderFailedDestroyKeepsDeployment(t *testing.T) {
	f := &fakeDMS{
		getDeployment: func(id string) (*bundledeployments.Deployment, error) {
			return &bundledeployments.Deployment{Name: "deployments/" + id, LastVersionId: "2"}, nil
		},
	}
	r := NewRecorder(f, "stored-id", testStatePath, "dev", VersionTypeDestroy, nil)

	require.NoError(t, r.CreateVersion(t.Context()))
	require.NoError(t, r.CompleteVersion(t.Context(), false))

	assert.Equal(t, bundledeployments.VersionCompleteVersionCompleteFailure, f.completed[0].CompletionReason)
	// A failed destroy leaves the deployment in place.
	assert.Empty(t, f.deleted)
}

func TestNilRecorderIsNoOp(t *testing.T) {
	var r *Recorder
	assert.NoError(t, r.CreateVersion(t.Context()))
	assert.NoError(t, r.CompleteVersion(t.Context(), true))
	assert.Empty(t, r.DeploymentID())
	assert.Zero(t, r.Version())
}

func TestRecorderCompleteVersionNoOpWithoutCreateVersion(t *testing.T) {
	f := &fakeDMS{}
	r := NewRecorder(f, "stored-id", testStatePath, "dev", VersionTypeDeploy, nil)
	// CompleteVersion before CreateVersion is a no-op (nothing was claimed).
	require.NoError(t, r.CompleteVersion(t.Context(), true))
	assert.Empty(t, f.completed)
}

func TestDeploymentIDFromName(t *testing.T) {
	id, err := deploymentIDFromName("deployments/abc-123")
	require.NoError(t, err)
	assert.Equal(t, "abc-123", id)

	_, err = deploymentIDFromName("abc-123")
	assert.Error(t, err)

	_, err = deploymentIDFromName("deployments/")
	assert.Error(t, err)
}
