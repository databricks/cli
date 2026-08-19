package dms

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeVersionRequest is a CreateVersion call captured by fakeRaw.
type fakeVersionRequest struct {
	deploymentID string
	versionID    string
	body         CreateVersionRequest
}

// updaterCall is an UpdateOperation call captured by fakeRaw.
type updaterCall struct {
	deploymentID string
	version      int64
	key          ResourceKey
	sequenceID   string
	update       OperationUpdate
}

// fakeRaw stands in for the hand-written half of a Client, capturing what the CLI would put
// on the wire.
type fakeRaw struct {
	mu sync.Mutex

	// versions collects CreateVersion calls, and versionErr fails them.
	versions   []fakeVersionRequest
	versionErr error

	// updates collects UpdateOperation calls. sequence is what the service reports back, and
	// the call at index failOn fails instead.
	updates  []updaterCall
	sequence string
	failOn   int
}

func newFakeRaw(sequence string) *fakeRaw {
	return &fakeRaw{sequence: sequence, failOn: -1}
}

func (f *fakeRaw) CreateVersion(ctx context.Context, deploymentID, versionID string, body CreateVersionRequest) (*bundledeployments.Version, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.versions = append(f.versions, fakeVersionRequest{deploymentID: deploymentID, versionID: versionID, body: body})
	if f.versionErr != nil {
		return nil, f.versionErr
	}
	return &bundledeployments.Version{VersionId: versionID}, nil
}

func (f *fakeRaw) UpdateOperation(ctx context.Context, deploymentID string, version int64, key ResourceKey, sequenceID string, update OperationUpdate) (string, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	callNum := len(f.updates)
	f.updates = append(f.updates, updaterCall{
		deploymentID: deploymentID,
		version:      version,
		key:          key,
		sequenceID:   sequenceID,
		update:       update,
	})
	if callNum == f.failOn {
		return "", errors.New("injected error")
	}
	return f.sequence, nil
}

func TestClientNamesEveryResourceTheSameWay(t *testing.T) {
	// One format each, so a call only ever passes ids.
	assert.Equal(t, "deployments/dep-1", deploymentName("dep-1"))
	assert.Equal(t, "deployments/dep-1/versions/2", versionName("dep-1", 2))
}

func TestClientFormatsTheVersionIDForCreateVersion(t *testing.T) {
	// The version is a number everywhere in the CLI; the request wants it as a string.
	raw := newFakeRaw("1")
	c := &Client{raw: raw}

	_, err := c.CreateVersion(t.Context(), "dep-1", 5, CreateVersionRequest{})
	require.NoError(t, err)

	require.Len(t, raw.versions, 1)
	assert.Equal(t, "5", raw.versions[0].versionID)
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

func TestUpdateRequestSendsAFieldOnlyWhenTheMaskNamesIt(t *testing.T) {
	// Every case carries the same values, so what reaches the body is decided by the mask
	// alone. A failure sending state would drop the resource from the deployment, and
	// resource_id does not ride along with state.
	update := OperationUpdate{
		State:        json.RawMessage(`{"state":{"name":"foo"}}`),
		ResourceID:   "job-1",
		Status:       bundledeployments.OperationStatusOperationStatusSucceeded,
		ErrorMessage: "boom",
	}

	tests := []struct {
		name   string
		fields Fields
		want   updateOperationRequest
	}{
		{
			name:   "a write that describes the resource",
			fields: DescribesResource,
			want: updateOperationRequest{
				State:        `{"state":{"name":"foo"}}`,
				ResourceId:   "job-1",
				Status:       bundledeployments.OperationStatusOperationStatusSucceeded,
				ErrorMessage: "boom",
				SequenceId:   "3",
			},
		},
		{
			name:   "a failure that keeps the recorded state",
			fields: KeepsState,
			want: updateOperationRequest{
				Status:       bundledeployments.OperationStatusOperationStatusSucceeded,
				ErrorMessage: "boom",
				SequenceId:   "3",
			},
		},
		{
			name:   "resource_id without state",
			fields: FieldResourceID,
			want: updateOperationRequest{
				ResourceId: "job-1",
				SequenceId: "3",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			update.Fields = tt.fields
			assert.Equal(t, tt.want, newUpdateRequest(update, "3"))
		})
	}
}
