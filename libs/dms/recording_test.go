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

// What a recorded deploy puts on the wire - the version it claims, the operations it stages,
// the completion it reports - is asserted end to end by acceptance/bundle/dms. What is left
// here is what only an injected API error or a disabled recording can reach.

// fakeDMS answers the generated calls a Recording makes. It embeds the SDK interface so it
// satisfies it while only overriding those.
type fakeDMS struct {
	bundledeployments.BundleDeploymentsInterface

	getDeployment func(id string) (*bundledeployments.Deployment, error)

	created   []bundledeployments.CreateDeploymentRequest
	completed []bundledeployments.CompleteVersionRequest
	deleted   []string

	// raw captures what the hand-written half of the client sent; see fakeRaw.
	raw *fakeRaw
}

func (f *fakeDMS) CreateDeployment(ctx context.Context, req bundledeployments.CreateDeploymentRequest) (*bundledeployments.Deployment, error) {
	f.created = append(f.created, req)
	return &bundledeployments.Deployment{Name: "deployments/new-id"}, nil
}

func (f *fakeDMS) GetDeployment(ctx context.Context, req bundledeployments.GetDeploymentRequest) (*bundledeployments.Deployment, error) {
	return f.getDeployment(req.Name)
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

// deploymentAt answers GetDeployment with a deployment whose last version is lastVersion.
func deploymentAt(lastVersion string) func(string) (*bundledeployments.Deployment, error) {
	return func(name string) (*bundledeployments.Deployment, error) {
		return &bundledeployments.Deployment{Name: name, LastVersionId: lastVersion}, nil
	}
}

// testRecording records the stored deployment through f, failing CreateVersion with
// versionErr when set.
func testRecording(f *fakeDMS, versionType VersionType, versionErr error) Recording {
	f.raw = newFakeRaw("1")
	f.raw.versionErr = versionErr
	return NewRecording(RecordingOptions{
		Client:       &Client{Service: f, raw: f.raw},
		DeploymentID: "stored-id",
		VersionType:  versionType,
	})
}

func TestRecordingStartReportsWhatTheServiceRefused(t *testing.T) {
	aborted := &apierr.APIError{StatusCode: 409, ErrorCode: "ABORTED"}
	exhausted := &apierr.APIError{StatusCode: 429, ErrorCode: "RESOURCE_EXHAUSTED"}

	tests := []struct {
		name          string
		getDeployment func(string) (*bundledeployments.Deployment, error)
		versionErr    error
		wantMessages  []string
		wantCause     error
	}{
		{
			name: "the deployment cannot be read",
			getDeployment: func(string) (*bundledeployments.Deployment, error) {
				return nil, errors.New("boom")
			},
			wantMessages: []string{"failed to get deployment"},
		},
		{
			// The service has a deployment for every BUNDLE_DEPLOYMENT node, so a not-found for
			// a node get-status just returned is a broken invariant, not anything the user did.
			name: "the deployment the workspace node names is gone",
			getDeployment: func(string) (*bundledeployments.Deployment, error) {
				return nil, fmt.Errorf("deployment: %w", apierr.ErrNotFound)
			},
			wantMessages: []string{"internal error: no deployment found for the file with object id stored-id"},
		},
		{
			name:          "another deploy claimed the version number",
			getDeployment: deploymentAt("4"),
			versionErr:    aborted,
			wantMessages:  []string{"another deploy already claimed version 5", "try again"},
			wantCause:     aborted,
		},
		{
			name:          "the bundle stages more operations than a version holds",
			getDeployment: deploymentAt("4"),
			versionErr:    exhausted,
			wantMessages:  []string{"this bundle deploys 1 resources"},
			wantCause:     exhausted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeDMS{getDeployment: tt.getDeployment}
			r := testRecording(f, VersionTypeDeploy, tt.versionErr)

			_, err := r.Start(t.Context(), []StagedOperation{{ResourceKey: "jobs.foo"}})

			require.Error(t, err)
			for _, want := range tt.wantMessages {
				assert.ErrorContains(t, err, want)
			}
			if tt.wantCause != nil {
				// Wrapped, so a caller can still match on what the service said.
				assert.ErrorIs(t, err, tt.wantCause)
			}
			assert.Empty(t, f.created, "the deployment already exists")
		})
	}
}

func TestRecordingFinishDeletesTheDeploymentOnlyForACompletedDestroy(t *testing.T) {
	// A failed destroy leaves resources behind, so its deployment has to stay for the next
	// deploy to find.
	tests := []struct {
		name        string
		versionType VersionType
		success     bool
		wantDeleted bool
	}{
		{name: "a destroy that completed", versionType: VersionTypeDestroy, success: true, wantDeleted: true},
		{name: "a destroy that failed", versionType: VersionTypeDestroy, success: false},
		{name: "a deploy", versionType: VersionTypeDeploy, success: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeDMS{getDeployment: deploymentAt("2")}
			r := testRecording(f, tt.versionType, nil)

			_, err := r.Start(t.Context(), nil)
			require.NoError(t, err)
			require.NoError(t, r.Finish(t.Context(), tt.success))

			wantReason := bundledeployments.VersionCompleteVersionCompleteSuccess
			if !tt.success {
				wantReason = bundledeployments.VersionCompleteVersionCompleteFailure
			}
			require.Len(t, f.completed, 1)
			assert.Equal(t, wantReason, f.completed[0].CompletionReason)

			if tt.wantDeleted {
				assert.Equal(t, []string{"deployments/stored-id"}, f.deleted)
			} else {
				assert.Empty(t, f.deleted)
			}
		})
	}
}

func TestDisabledRecordingIsNoOp(t *testing.T) {
	r := Disabled()

	require.NoError(t, r.Prepare(t.Context()))
	sink, err := r.Start(t.Context(), []StagedOperation{{ResourceKey: "jobs.foo"}})
	require.NoError(t, err)
	require.NoError(t, r.Finish(t.Context(), true))

	assert.Empty(t, r.DeploymentID())
	assert.Zero(t, r.Version())
	// No sink, which is what leaves the state DB recording nothing and nothing to stamp.
	assert.Nil(t, sink)
}
