package aircmd

import (
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDockerCredentials(t *testing.T) {
	got := encodeDockerCredentials("alice", "pat")
	decoded, err := base64.StdEncoding.DecodeString(got)
	require.NoError(t, err)
	assert.Equal(t, "alice:pat", string(decoded))
}

// credServer records secret puts and lets a test choose the scope list and the
// create-scope failure. me is the current-user name returned to the client.
type credServer struct {
	existingScopes []string
	createStatus   int    // 0 → 200
	createCode     string // error_code for a failed create; defaults to the quota code
	putBodies      []string
}

func (cs *credServer) start(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/scim/v2/Me"):
			_, _ = w.Write([]byte(`{"userName":"user@example.com"}`))
		case r.URL.Path == "/api/2.0/secrets/scopes/list":
			var scopes []map[string]string
			for _, s := range cs.existingScopes {
				scopes = append(scopes, map[string]string{"name": s})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"scopes": scopes})
		case r.URL.Path == "/api/2.0/secrets/scopes/create":
			if cs.createStatus != 0 {
				code := cs.createCode
				if code == "" {
					code = "RESOURCE_LIMIT_EXCEEDED"
				}
				w.WriteHeader(cs.createStatus)
				_, _ = w.Write([]byte(`{"error_code":"` + code + `","message":"denied"}`))
				return
			}
			_, _ = w.Write([]byte(`{}`))
		case r.URL.Path == "/api/2.0/secrets/put":
			body, _ := io.ReadAll(r.Body)
			cs.putBodies = append(cs.putBodies, string(body))
			_, _ = w.Write([]byte(`{}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestStoreDockerCredentialsCreatesScope(t *testing.T) {
	cs := &credServer{}
	w := newTestWorkspaceClient(t, cs.start(t))

	scope, key, err := storeDockerCredentials(t.Context(), w, "docker.io/library/ubuntu:latest", "bob", "secret")
	require.NoError(t, err)
	assert.Equal(t, "docker-credentials-user@example.com", scope)
	assert.Equal(t, "docker.io-bob-local", key)
	require.Len(t, cs.putBodies, 1)
	assert.Contains(t, cs.putBodies[0], base64.StdEncoding.EncodeToString([]byte("bob:secret")))
}

// TestStoreDockerCredentialsKeyIsPerRegistry guards against the same username on
// two registries colliding on one secret key.
func TestStoreDockerCredentialsKeyIsPerRegistry(t *testing.T) {
	cases := map[string]string{
		"docker.io/library/ubuntu:latest": "docker.io-bob-local",
		"nvcr.io/nvidia/pytorch:24.01":    "nvcr.io-bob-local",
		"ghcr.io/org/img:1.0":             "ghcr.io-bob-local",
	}
	for imageURL, wantKey := range cases {
		t.Run(imageURL, func(t *testing.T) {
			cs := &credServer{}
			w := newTestWorkspaceClient(t, cs.start(t))
			_, key, err := storeDockerCredentials(t.Context(), w, imageURL, "bob", "secret")
			require.NoError(t, err)
			assert.Equal(t, wantKey, key)
		})
	}
}

func TestStoreDockerCredentialsScopeExists(t *testing.T) {
	// When the scope already exists, create is not required; storage still succeeds.
	cs := &credServer{existingScopes: []string{"docker-credentials-user@example.com"}}
	w := newTestWorkspaceClient(t, cs.start(t))

	_, _, err := storeDockerCredentials(t.Context(), w, "nvcr.io/org/img:1.0", "bob", "secret")
	require.NoError(t, err)
	assert.Len(t, cs.putBodies, 1)
}

// TestStoreDockerCredentialsPermissionDenied covers the workspace where the user
// may not create secret scopes: the failure must surface with admin guidance
// rather than be swallowed into a misleading "run docker login" error.
func TestStoreDockerCredentialsPermissionDenied(t *testing.T) {
	cs := &credServer{createStatus: http.StatusForbidden, createCode: "PERMISSION_DENIED"}
	w := newTestWorkspaceClient(t, cs.start(t))

	_, _, err := storeDockerCredentials(t.Context(), w, "nvcr.io/org/img:1.0", "bob", "secret")
	require.Error(t, err)
	assert.NotErrorIs(t, err, errSecretScopeQuota)
	assert.Contains(t, err.Error(), `creating secret scope "docker-credentials-user@example.com" was denied`)
	assert.Contains(t, err.Error(), "Ask a workspace admin")
	assert.Empty(t, cs.putBodies, "must not attempt to store the secret when the scope could not be created")
}

func TestStoreDockerCredentialsQuotaError(t *testing.T) {
	cs := &credServer{createStatus: http.StatusForbidden}
	w := newTestWorkspaceClient(t, cs.start(t))

	_, _, err := storeDockerCredentials(t.Context(), w, "nvcr.io/org/img:1.0", "bob", "secret")
	require.Error(t, err)
	assert.ErrorIs(t, err, errSecretScopeQuota)
}
