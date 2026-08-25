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

func TestDeploymentIDFromName(t *testing.T) {
	id, err := deploymentIDFromName("deployments/abc-123")
	require.NoError(t, err)
	assert.Equal(t, "abc-123", id)

	_, err = deploymentIDFromName("abc-123")
	assert.Error(t, err)

	_, err = deploymentIDFromName("deployments/")
	assert.Error(t, err)
}

func TestUpdateRequestSendsExactlyTheMaskedFields(t *testing.T) {
	// The service rejects an update whose mask names a field the body leaves out, and treats a
	// field it does carry as written - so the body has to hold every masked field and nothing
	// else, empty values included. Both masks the CLI builds are asserted on the wire by
	// acceptance/bundle/dms; this pins the rule they both rely on.
	update := OperationUpdate{
		Fields:     DescribesResource,
		State:      json.RawMessage(`{"state":{"name":"foo"}}`),
		ResourceID: "job-1",
		Status:     bundledeployments.OperationStatusOperationStatusSucceeded,
	}

	// A successful write reports no error, and the empty value is what clears an earlier one.
	assert.Equal(t, map[string]any{
		"state":         `{"state":{"name":"foo"}}`,
		"resource_id":   "job-1",
		"error_message": "",
		"status":        bundledeployments.OperationStatusOperationStatusSucceeded,
		"sequence_id":   "3",
	}, newUpdateRequest(update, "3"))

	// A failure keeps the recorded state, so state is absent rather than empty: naming it
	// would clear what the resource last recorded.
	assert.Equal(t, map[string]any{
		"error_message": "boom",
		"status":        bundledeployments.OperationStatusOperationStatusFailed,
		"sequence_id":   "3",
	}, newUpdateRequest(NewFailureUpdate("job-1", errors.New("boom")), "3"))
}
