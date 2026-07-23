package dms

import (
	"context"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	id := req.DeploymentId
	if id == "" {
		id = f.assignedID
	}
	return &bundledeployments.Deployment{Name: "deployments/" + id}, nil
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
	// A first deploy has no stored deployment ID.
	r := NewRecorder(f, "", "dev", VersionTypeDeploy)

	require.NoError(t, r.CreateVersion(t.Context()))

	// The deployment was created with an empty ID so the server assigns one, and
	// the recorder exposes the assigned ID for the caller to persist.
	require.Len(t, f.created, 1)
	assert.Empty(t, f.created[0].DeploymentId)
	assert.Equal(t, "server-generated-id", r.DeploymentID())

	// The first version is 1, parented under the assigned deployment.
	require.Len(t, f.versions, 1)
	assert.Equal(t, "1", f.versions[0].VersionId)
	assert.Equal(t, "deployments/server-generated-id", f.versions[0].Parent)
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
	r := NewRecorder(f, "stored-id", "dev", VersionTypeDeploy)

	require.NoError(t, r.CreateVersion(t.Context()))

	// No new deployment is created; the version increments to last_version_id + 1.
	assert.Empty(t, f.created)
	require.Len(t, f.versions, 1)
	assert.Equal(t, "5", f.versions[0].VersionId)
	assert.Equal(t, "stored-id", r.DeploymentID())
}

func TestRecorderDestroyDeletesDeploymentOnSuccess(t *testing.T) {
	f := &fakeDMS{
		getDeployment: func(id string) (*bundledeployments.Deployment, error) {
			return &bundledeployments.Deployment{Name: "deployments/" + id, LastVersionId: "2"}, nil
		},
	}
	r := NewRecorder(f, "stored-id", "dev", VersionTypeDestroy)

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
	r := NewRecorder(f, "stored-id", "dev", VersionTypeDestroy)

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
	r := NewRecorder(f, "stored-id", "dev", VersionTypeDeploy)
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
