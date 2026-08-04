package direct

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/databricks/cli/bundle/deployplan"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeOpClient struct {
	bundledeployments.BundleDeploymentsInterface

	mu       sync.Mutex
	requests []bundledeployments.CreateOperationRequest
	err      error
}

func (f *fakeOpClient) CreateOperation(ctx context.Context, req bundledeployments.CreateOperationRequest) (*bundledeployments.Operation, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.requests = append(f.requests, req)
	return &bundledeployments.Operation{}, f.err
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
	// State is sent as an opaque JSON string carrying the serialized config.
	assert.JSONEq(t, `{"state":{"name":"foo"}}`, req.Operation.State)
}

func TestOperationRecorderToleratesResponseDeserializationError(t *testing.T) {
	// DMS returns sequence_id as a JSON string the SDK cannot parse into its
	// int64 field, so CreateOperation can fail to deserialize a response the
	// server accepted. The CLI discards the response, so this must not fail the
	// deploy.
	f := &fakeOpClient{err: errors.New("failed to unmarshal response body: invalid character '1' after top-level value")}
	r := NewOperationRecorder(f, "dep-1", 2)

	op, err := newRecordedOperation(deployplan.Create, "job-123", map[string]string{"name": "foo"}, nil)
	require.NoError(t, err)
	assert.NoError(t, r.upload(t.Context(), "resources.jobs.foo", op))
	assert.Len(t, f.requests, 1)
}

func TestOperationRecorderPropagatesAPIError(t *testing.T) {
	// A real API error (status >= 400) must still fail the deploy.
	f := &fakeOpClient{err: &apierr.APIError{StatusCode: 400, ErrorCode: "INVALID_PARAMETER_VALUE", Message: "bad request"}}
	r := NewOperationRecorder(f, "dep-1", 2)

	op, err := newRecordedOperation(deployplan.Create, "job-123", map[string]string{"name": "foo"}, nil)
	require.NoError(t, err)
	assert.Error(t, r.upload(t.Context(), "resources.jobs.foo", op))
}

func TestOperationRecorderPropagatesTransportError(t *testing.T) {
	// A transport error means the request may never have reached DMS, so it must
	// not be swallowed like a response-deserialization error.
	f := &fakeOpClient{err: errors.New("dial tcp: connection refused")}
	r := NewOperationRecorder(f, "dep-1", 2)

	op, err := newRecordedOperation(deployplan.Create, "job-123", map[string]string{"name": "foo"}, nil)
	require.NoError(t, err)
	assert.Error(t, r.upload(t.Context(), "resources.jobs.foo", op))
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
