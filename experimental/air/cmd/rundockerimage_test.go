package aircmd

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
	require.NoError(t, waitForRegisteredImage(t.Context(), newTestImageClient(t, url), "nvcr.io/org/img:1.0"))
}

func TestWaitForRegisteredImageNotRegistered(t *testing.T) {
	// :get 404s, so the run must stop with registration guidance.
	url := imageServer(t, `{}`, "")
	err := waitForRegisteredImage(t.Context(), newTestImageClient(t, url), "nvcr.io/org/img:1.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docker image not registered")
	assert.Contains(t, err.Error(), "air register-image nvcr.io/org/img:1.0")
}

func TestWaitForRegisteredImageFailed(t *testing.T) {
	url := imageServer(t, `{}`, `{"state":"FAILED","status_message":"manifest not found"}`)
	err := waitForRegisteredImage(t.Context(), newTestImageClient(t, url), "nvcr.io/org/img:1.0")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registration failed: manifest not found")
	assert.Contains(t, err.Error(), "re-register")
}

func TestWaitForRegisteredImageWaitsWhileImporting(t *testing.T) {
	// First poll is still importing; the next reports AVAILABLE.
	url := imageServer(t, `{}`,
		`{"state":"IMPORTING"}`,
		`{"state":"AVAILABLE","manifest_sha256":"abc"}`)
	require.NoError(t, waitForRegisteredImage(t.Context(), newTestImageClient(t, url), "nvcr.io/org/img:1.0"))
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
	require.NoError(t, prepareDockerImage(t.Context(), w, img))
	assert.Zero(t, posts, "default tag policy must not re-register")
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
	require.NoError(t, prepareDockerImage(t.Context(), w, img))
	assert.Equal(t, 1, posts, "tag_policy=latest must re-register against the source registry")
}
