package dms

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/bundledeployments"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient returns a workspace client pointed at a server that serves a
// single get-status response.
func newTestClient(t *testing.T, statusCode int, body string) *databricks.WorkspaceClient {
	t.Helper()

	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Query().Get("path")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() {
		assert.Equal(t, nodePath, gotPath)
	})

	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:        srv.URL,
		Token:       "token",
		Credentials: config.PatCredentials{},
	})
	require.NoError(t, err)
	return w
}

const nodePath = "/Workspace/state/" + DeploymentNodeName

func TestResolveDeploymentIDReturnsNodeID(t *testing.T) {
	w := newTestClient(t, http.StatusOK, `{"object_type":"FILE","object_id":123456789,"path":"/Workspace/state/`+DeploymentNodeName+`"}`)

	// The workspace node ID is the deployment ID, so no local state is consulted.
	id, err := ResolveDeploymentID(t.Context(), w, "/Workspace/state")
	require.NoError(t, err)
	assert.Equal(t, "123456789", id)
}

func TestResolveDeploymentIDAbsentWhenNodeMissing(t *testing.T) {
	w := newTestClient(t, http.StatusNotFound, `{"error_code":"RESOURCE_DOES_NOT_EXIST","message":"Path (/Workspace/state/`+DeploymentNodeName+`) doesn't exist."}`)

	// A bundle that never recorded a deployment, or whose deployment was
	// destroyed (the service trashes the node), has no ID rather than an error.
	id, err := ResolveDeploymentID(t.Context(), w, "/Workspace/state")
	require.NoError(t, err)
	assert.Empty(t, id)
}

func TestResolveDeploymentIDPropagatesOtherErrors(t *testing.T) {
	w := newTestClient(t, http.StatusForbidden, `{"error_code":"PERMISSION_DENIED","message":"nope"}`)

	// Anything other than a missing node is fatal: silently treating it as absent
	// would create a second deployment for a bundle that already has one.
	_, err := ResolveDeploymentID(t.Context(), w, "/Workspace/state")
	require.Error(t, err)
	assert.ErrorContains(t, err, "looking up deployment at /Workspace/state/"+DeploymentNodeName)
}

func TestReadDeploymentReportsWhatTheReadRefused(t *testing.T) {
	tests := []struct {
		name        string
		getErr      error
		wantMessage string
	}{
		{
			name:        "the deployment cannot be read",
			getErr:      errors.New("boom"),
			wantMessage: "failed to get deployment",
		},
		{
			// The service has a deployment for every BUNDLE_DEPLOYMENT node, so a not-found for
			// a node get-status just returned is a broken invariant, not anything the user did.
			name:        "the deployment the workspace node names is gone",
			getErr:      fmt.Errorf("deployment: %w", apierr.ErrNotFound),
			wantMessage: "internal error: no deployment found for the file with object id stored-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeDMS{getDeployment: func(string) (*bundledeployments.Deployment, error) {
				return nil, tt.getErr
			}}

			_, err := ReadDeployment(t.Context(), &Client{Service: f}, "stored-id")

			require.Error(t, err)
			assert.ErrorContains(t, err, tt.wantMessage)
		})
	}
}

func TestReadDeploymentWithoutOneReadsNothing(t *testing.T) {
	// Nothing to read before the first recorded deploy, so it must not call the service: the
	// fake would panic on GetDeployment.
	dep, err := ReadDeployment(t.Context(), &Client{Service: &fakeDMS{}}, "")

	require.NoError(t, err)
	assert.Nil(t, dep)
}

func TestReadDeploymentReturnsTheRecord(t *testing.T) {
	dep, err := ReadDeployment(t.Context(), &Client{Service: &fakeDMS{getDeployment: deploymentAt("7")}}, "stored-id")

	require.NoError(t, err)
	assert.Equal(t, "7", dep.LastVersionId)
}
