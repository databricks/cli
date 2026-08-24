package dms

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	sdkclient "github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// sentRequest is one request the CLI put on the wire.
type sentRequest struct {
	method string
	path   string
	query  string
	body   string
}

// reply is what the test server answers with.
type reply struct {
	status int
	body   string
}

// fakeAPI serves the two requests the CLI writes by hand and records what arrived, so a test
// can assert the path, the query and the body it built.
type fakeAPI struct {
	mu       sync.Mutex
	requests []sentRequest

	// replies are answered in order; the last one repeats once they run out.
	replies []reply
}

func (f *fakeAPI) handle(w http.ResponseWriter, r *http.Request) {
	// The SDK probes for host metadata before its first call; that is not a request the CLI
	// made, so it neither records nor consumes a reply.
	if !strings.HasPrefix(r.URL.Path, "/api/2.0/bundle") {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	body, _ := io.ReadAll(r.Body)

	f.mu.Lock()
	f.requests = append(f.requests, sentRequest{
		method: r.Method,
		path:   r.URL.Path,
		query:  r.URL.RawQuery,
		body:   string(body),
	})
	answer := f.replies[min(len(f.requests)-1, len(f.replies)-1)]
	f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(answer.status)
	_, _ = w.Write([]byte(answer.body))
}

func (f *fakeAPI) sent() []sentRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]sentRequest(nil), f.requests...)
}

// newFakeAPI returns a Client whose hand-written requests reach a server answering replies in
// order, and the record of what it received.
func newFakeAPI(t *testing.T, replies ...reply) (*Client, *fakeAPI) {
	t.Helper()

	api := &fakeAPI{replies: replies}
	srv := httptest.NewServer(http.HandlerFunc(api.handle))
	t.Cleanup(srv.Close)

	raw, err := sdkclient.New(&config.Config{
		Host:        srv.URL,
		Token:       "token",
		Credentials: config.PatCredentials{},
		// The SDK retries some of the codes these tests inject - RESOURCE_EXHAUSTED among
		// them - so bound the window instead of waiting out the default.
		RetryTimeoutSeconds: 1,
	})
	require.NoError(t, err)

	return &Client{api: raw}, api
}

func TestClientNamesEveryResourceTheSameWay(t *testing.T) {
	// One format each, so a call only ever passes ids.
	assert.Equal(t, "deployments/dep-1", deploymentName("dep-1"))
	assert.Equal(t, "deployments/dep-1/versions/2", versionName("dep-1", 2))
}

func TestClientCreateVersionSendsTheNumberAsTheVersionID(t *testing.T) {
	// The version is a number everywhere in the CLI; the request carries it as a string, and
	// in the query rather than the body.
	c, api := newFakeAPI(t, reply{status: http.StatusOK, body: `{"version_id":"5"}`})

	version, err := c.CreateVersion(t.Context(), "dep-1", 5, CreateVersionRequest{
		VersionType:       VersionTypeDeploy,
		PreviousVersionId: "4",
		Operations:        []StagedOperation{{ResourceKey: "jobs.foo"}},
	})
	require.NoError(t, err)
	assert.Equal(t, "5", version.VersionId)

	sent := api.sent()
	require.Len(t, sent, 1)
	assert.Equal(t, http.MethodPost, sent[0].method)
	assert.Equal(t, "/api/2.0/bundle/deployments/dep-1/versions", sent[0].path)
	assert.Equal(t, "version_id=5", sent[0].query)

	// previous_version_id is the whole reason this request is written by hand.
	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(sent[0].body), &body))
	assert.Equal(t, "4", body["previous_version_id"])
	assert.Len(t, body["operations"], 1)
}

func TestClientUpdateOperationSendsTheMaskAndReadsTheSequenceBack(t *testing.T) {
	// The service answers with sequence_id as a string, which the generated client cannot
	// read - the reason this request is written by hand.
	c, api := newFakeAPI(t, reply{status: http.StatusOK, body: `{"sequence_id":"7"}`})

	next, err := c.UpdateOperation(t.Context(), "dep-1", 2, "jobs.foo", "0", NewFailureUpdate("job-1", assert.AnError))
	require.NoError(t, err)
	assert.Equal(t, "7", next)

	sent := api.sent()
	require.Len(t, sent, 1)
	assert.Equal(t, http.MethodPatch, sent[0].method)
	assert.Equal(t, "/api/2.0/bundle/deployments/dep-1/versions/2/operations/jobs.foo", sent[0].path)
	assert.Equal(t, "update_mask=error_message%2Cstatus", sent[0].query)

	var body map[string]any
	require.NoError(t, json.Unmarshal([]byte(sent[0].body), &body))
	assert.Equal(t, "0", body["sequence_id"])
	assert.Equal(t, string(bundledeployments.OperationStatusOperationStatusFailed), body["status"])
	// The mask names neither, so what an earlier write recorded stands.
	assert.NotContains(t, body, "state")
	assert.NotContains(t, body, "resource_id")
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
