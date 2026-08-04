package aircmd

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDigestDisplay(t *testing.T) {
	assert.Equal(t, "digest unknown", digestDisplay(""))
	assert.Equal(t, "abc", digestDisplay("abc"))
	assert.Equal(t, "0123456789abcdef...", digestDisplay("0123456789abcdefghij"))
}

func TestWaitForRegisteredImageAvailable(t *testing.T) {
	url := imageServer(t, `{}`, `{"state":"AVAILABLE","manifest_sha256":"abc"}`)
	require.NoError(t, waitForRegisteredImage(cmdio.MockDiscard(t.Context()), newTestImageClient(t, url), "nvcr.io/org/img:1.0"))
}

func TestWaitForRegisteredImageNotRegistered(t *testing.T) {
	// :get 404s, so the run must stop with registration guidance.
	url := imageServer(t, `{}`, "")
	err := waitForRegisteredImage(cmdio.MockDiscard(t.Context()), newTestImageClient(t, url), "nvcr.io/org/img:1.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker image not registered")
	assert.Contains(t, err.Error(), "air register-image nvcr.io/org/img:1.0")
}

func TestWaitForRegisteredImageFailed(t *testing.T) {
	url := imageServer(t, `{}`, `{"state":"FAILED","status_message":"manifest not found"}`)
	err := waitForRegisteredImage(cmdio.MockDiscard(t.Context()), newTestImageClient(t, url), "nvcr.io/org/img:1.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registration failed: manifest not found")
	assert.Contains(t, err.Error(), "re-register")
}

func TestWaitForRegisteredImageWaitsWhileImporting(t *testing.T) {
	// First poll is still importing; the next reports AVAILABLE.
	url := imageServer(t, `{}`,
		`{"state":"IMPORTING"}`,
		`{"state":"AVAILABLE","manifest_sha256":"abc"}`)
	require.NoError(t, waitForRegisteredImage(cmdio.MockDiscard(t.Context()), newTestImageClient(t, url), "nvcr.io/org/img:1.0"))
}

// latestImageServer serves a registry where the image is registered and
// AVAILABLE, counting POSTs so a test can assert whether a re-registration
// happened.
func latestImageServer(t *testing.T, posts *int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case imagesAPIPath + ":get":
			_, _ = w.Write([]byte(`{"state":"AVAILABLE","manifest_sha256":"abc"}`))
		case imagesAPIPath:
			*posts++
			_, _ = w.Write([]byte(`{"image":{"state":"AVAILABLE","manifest_sha256":"abc"}}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestPrepareDockerImageAutoDoesNotReregister(t *testing.T) {
	var posts int
	w := newTestWorkspaceClient(t, latestImageServer(t, &posts))
	img := &dockerImageConfig{URL: "nvcr.io/org/img:1.0"}
	require.NoError(t, prepareDockerImage(cmdio.MockDiscard(t.Context()), w, img))
	assert.Zero(t, posts, "default tag policy must not re-register")
}

// TestPrepareDockerImageLatestDiscoversCredentials covers the config-without-
// credentials path: creds come from the local Docker config and ride the POST.
func TestPrepareDockerImageLatestDiscoversCredentials(t *testing.T) {
	var postBodies []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case imagesAPIPath + ":get":
			_, _ = w.Write([]byte(`{"state":"AVAILABLE","manifest_sha256":"abc"}`))
		case imagesAPIPath + ":checkImageAccess":
			_, _ = w.Write([]byte(`{"publicly_accessible": false}`))
		case imagesAPIPath:
			body, _ := io.ReadAll(r.Body)
			postBodies = append(postBodies, string(body))
			_, _ = w.Write([]byte(`{"image":{"state":"AVAILABLE","manifest_sha256":"abc"}}`))
		case "/api/2.0/secrets/scopes/list":
			_, _ = w.Write([]byte(`{"scopes":[]}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)

	ctx := cmdio.MockDiscard(writeDockerConfig(t, `{"auths":{"nvcr.io":{"auth":"`+b64(t, "bob:secret")+`"}}}`))
	w := newTestWorkspaceClient(t, srv.URL)
	img := &dockerImageConfig{URL: "nvcr.io/org/img:1.0", TagPolicy: "latest"}

	require.NoError(t, prepareDockerImage(ctx, w, img))
	require.Len(t, postBodies, 1)
	assert.Contains(t, postBodies[0], `"credentials_key":"nvcr.io-bob-local"`)
	assert.Contains(t, postBodies[0], `"credentials_scope":"docker-credentials-`)
}

// TestPrepareDockerImageLatestStorageDeniedReportsCause covers the journey where
// creds exist locally but can't be stored: the error must name that cause rather
// than tell the user to `docker login`.
func TestPrepareDockerImageLatestStorageDeniedReportsCause(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case imagesAPIPath + ":get":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_code":"NOT_FOUND","message":"not registered"}`))
		case imagesAPIPath + ":checkImageAccess":
			_, _ = w.Write([]byte(`{"publicly_accessible": false}`))
		case "/api/2.0/secrets/scopes/create":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error_code":"PERMISSION_DENIED","message":"denied"}`))
		case imagesAPIPath:
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error_code":"PERMISSION_DENIED","message":"cannot pull: unauthorized"}`))
		case "/api/2.0/secrets/scopes/list":
			_, _ = w.Write([]byte(`{"scopes":[]}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)

	ctx := cmdio.MockDiscard(writeDockerConfig(t, `{"auths":{"nvcr.io":{"auth":"`+b64(t, "bob:secret")+`"}}}`))
	w := newTestWorkspaceClient(t, srv.URL)
	img := &dockerImageConfig{URL: "nvcr.io/org/img:1.0", TagPolicy: "latest"}

	err := prepareDockerImage(ctx, w, img)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "could not be stored")
	assert.NotContains(t, err.Error(), "run `docker login`")
}

// TestPrepareDockerImageLatestExplicitCredsRejected asserts a rejected secret the
// user named reports that secret, and is not retried anonymously or blamed on a
// missing `docker login`.
func TestPrepareDockerImageLatestExplicitCredsRejected(t *testing.T) {
	var credentialedPOSTs int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case imagesAPIPath + ":get":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_code":"NOT_FOUND","message":"not registered"}`))
		case imagesAPIPath:
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), "credentials_scope") {
				credentialedPOSTs++
			}
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"error_code":"PERMISSION_DENIED","message":"denied"}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)

	w := newTestWorkspaceClient(t, srv.URL)
	img := &dockerImageConfig{
		URL:              "nvcr.io/org/img:1.0",
		TagPolicy:        "latest",
		CredentialsScope: "myscope",
		CredentialsKey:   "mykey",
	}

	err := prepareDockerImage(cmdio.MockDiscard(t.Context()), w, img)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "credentials in secret myscope/mykey were rejected")
	assert.NotContains(t, err.Error(), "docker login")
	assert.Equal(t, 1, credentialedPOSTs, "explicit credentials must not be retried anonymously")
}

func TestPrepareDockerImageLatestReregisters(t *testing.T) {
	var posts int
	w := newTestWorkspaceClient(t, latestImageServer(t, &posts))
	img := &dockerImageConfig{
		URL:              "nvcr.io/org/img:1.0",
		TagPolicy:        "latest",
		CredentialsScope: "scope",
		CredentialsKey:   "key",
	}
	require.NoError(t, prepareDockerImage(cmdio.MockDiscard(t.Context()), w, img))
	assert.Equal(t, 1, posts, "tag_policy=latest must re-register against the source registry")
}
