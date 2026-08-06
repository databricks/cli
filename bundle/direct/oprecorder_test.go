package direct

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOpClient struct {
	bundledeployments.BundleDeploymentsInterface

	mu       sync.Mutex
	requests []bundledeployments.CreateOperationRequest
}

func (f *fakeOpClient) CreateOperation(ctx context.Context, req bundledeployments.CreateOperationRequest) (*bundledeployments.Operation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)
	return &bundledeployments.Operation{}, nil
}

// uploadOne records a single operation through the given uploader, mirroring what
// an operationQueue worker does.
func uploadOne(t *testing.T, u operationUploader, resourceKey string, action deployplan.ActionType, resourceID string, state any) {
	t.Helper()
	op, err := newRecordedOperation(action, resourceID, state, nil)
	require.NoError(t, err)
	require.NoError(t, u.upload(t.Context(), resourceKey, op))
}

func TestOperationRecorderStripsResourcePrefix(t *testing.T) {
	f := &fakeOpClient{}
	r := NewOperationRecorder(f, "dep-1", 2)

	uploadOne(t, r, "resources.jobs.foo", deployplan.Create, "job-123", map[string]string{"name": "foo"})

	require.Len(t, f.requests, 1)
	req := f.requests[0]
	// The wire key drops the CLI-internal "resources." prefix, both in the query
	// param and the operation body.
	assert.Equal(t, "jobs.foo", req.ResourceKey)
	assert.Equal(t, "jobs.foo", req.Operation.ResourceKey)
	assert.Equal(t, "deployments/dep-1/versions/2", req.Parent)
	assert.Equal(t, bundledeployments.OperationActionTypeOperationActionTypeCreate, req.Operation.ActionType)
	assert.Equal(t, "job-123", req.Operation.ResourceId)
	require.NotNil(t, req.Operation.State)
}

func TestNewRecordedOperationRecordsStateAsIs(t *testing.T) {
	state := struct {
		Name  string `json:"name"`
		Token string `json:"token" bundle:"sensitive"`
	}{Name: "foo", Token: "super-secret"}

	op, err := newRecordedOperation(deployplan.Create, "job-123", state, nil)
	require.NoError(t, err)

	// The state is serialized as-is, including fields tagged bundle:"sensitive".
	assert.JSONEq(t,
		`{"state":{"name":"foo","token":"super-secret"}}`,
		string(op.state))
}

func TestNewRecordedOperationRecordsDependsOn(t *testing.T) {
	// depends_on rides in an envelope alongside the config: it cannot be
	// recomputed from the config, whose references are already resolved.
	dependsOn := []deployplan.DependsOnEntry{{Node: "resources.jobs.bar", Label: "${resources.jobs.bar.id}"}}

	op, err := newRecordedOperation(deployplan.Create, "job-123", map[string]string{"name": "foo"}, dependsOn)
	require.NoError(t, err)

	assert.JSONEq(t,
		`{"state":{"name":"foo"},"depends_on":[{"node":"resources.jobs.bar","label":"${resources.jobs.bar.id}"}]}`,
		string(op.state))
}

func TestNewRecordedOperationRejectsUnsupportedAction(t *testing.T) {
	_, err := newRecordedOperation(deployplan.Skip, "job-123", nil, nil)
	assert.Error(t, err)
}

func TestNewFailedOperationRecordsError(t *testing.T) {
	op, err := newFailedOperation(deployplan.Create, "", nil, errors.New("cluster spec is invalid"))
	require.NoError(t, err)

	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, op.status)
	assert.Equal(t, "cluster spec is invalid", op.errorMessage)
	// The resource was never written, so there is no state to serve back for it.
	assert.Nil(t, op.state)
}

func TestNewFailedOperationRecordsPriorStateWithID(t *testing.T) {
	// A failed recreate has already deleted the resource, so the id must come from
	// the pre-deploy record alongside the state: the service rejects state without
	// an id, since state describes a resource that exists.
	op, err := newFailedOperation(deployplan.Recreate, "main.some_schema", json.RawMessage(`{"state":{"catalog_name":"main"}}`), errors.New("Catalog 'mainx' does not exist"))
	require.NoError(t, err)

	assert.Equal(t, bundledeployments.OperationStatusOperationStatusFailed, op.status)
	assert.Equal(t, "main.some_schema", op.resourceID)
	assert.JSONEq(t, `{"state":{"catalog_name":"main"}}`, string(op.state))
}

func TestNewFailedOperationTruncatesLongError(t *testing.T) {
	// Truncated rather than rejected: a message over the limit would make recording
	// fail and hide the error it is reporting.
	op, err := newFailedOperation(deployplan.Update, "job-123", nil, errors.New(strings.Repeat("x", maxOperationErrorMessageSize+100)))
	require.NoError(t, err)

	assert.Len(t, op.errorMessage, maxOperationErrorMessageSize)
}

func TestDeployActionToSDK(t *testing.T) {
	cases := []struct {
		action deployplan.ActionType
		want   bundledeployments.OperationActionType
	}{
		{deployplan.Create, bundledeployments.OperationActionTypeOperationActionTypeCreate},
		{deployplan.Update, bundledeployments.OperationActionTypeOperationActionTypeUpdate},
		{deployplan.UpdateWithID, bundledeployments.OperationActionTypeOperationActionTypeUpdateWithId},
		{deployplan.Recreate, bundledeployments.OperationActionTypeOperationActionTypeRecreate},
		{deployplan.Resize, bundledeployments.OperationActionTypeOperationActionTypeResize},
		{deployplan.Delete, bundledeployments.OperationActionTypeOperationActionTypeDelete},
	}
	for _, c := range cases {
		got, err := deployActionToSDK(c.action)
		require.NoError(t, err)
		assert.Equal(t, c.want, got)
	}

	// Skip and Undefined never reach a recorder and are rejected.
	_, err := deployActionToSDK(deployplan.Skip)
	assert.Error(t, err)
	_, err = deployActionToSDK(deployplan.Undefined)
	assert.Error(t, err)
}
