package aircmd

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateTagPolicy(t *testing.T) {
	require.NoError(t, validateTagPolicy(""))
	require.NoError(t, validateTagPolicy("latest"))
	require.NoError(t, validateTagPolicy("  LATEST  "))

	err := validateTagPolicy("auto")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no longer supported")

	err = validateTagPolicy("bogus")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "only supported value is latest")
}

// imageServer serves the image API. Each :get call returns the next body in
// getBodies (repeating the last), where an empty string means 404; POST returns
// postBody. Sequencing the :get bodies lets a test set distinct before/after
// digests for the two :get calls resolveImage makes.
func imageServer(t *testing.T, postBody string, getBodies ...string) string {
	t.Helper()
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case imagesAPIPath + ":get":
			body := getBodies[min(call, len(getBodies)-1)]
			call++
			if body == "" {
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error_code":"NOT_FOUND","message":"not registered"}`))
				return
			}
			_, _ = w.Write([]byte(body))
		case imagesAPIPath:
			_, _ = w.Write([]byte(postBody))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestResolveImageFreshRegistration(t *testing.T) {
	url := imageServer(t, `{"image":{"state":"AVAILABLE","manifest_sha256":"newsha"}}`, "")
	updated, sha, err := resolveImage(t.Context(), newTestImageClient(t, url), "ubuntu", time.Second)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, "newsha", sha)
}

func TestResolveImageDigestUnchanged(t *testing.T) {
	body := `{"state":"AVAILABLE","manifest_sha256":"samesha"}`
	url := imageServer(t, `{"image":`+body+`}`, body)
	updated, sha, err := resolveImage(t.Context(), newTestImageClient(t, url), "ubuntu", time.Second)
	require.NoError(t, err)
	assert.False(t, updated)
	assert.Equal(t, "samesha", sha)
}

func TestResolveImageDigestChanged(t *testing.T) {
	// First :get is the pre-existing (old) digest; the re-read after POST returns
	// the new digest, so the image reports updated.
	url := imageServer(t, `{"image":{"state":"AVAILABLE","manifest_sha256":"newsha"}}`,
		`{"state":"AVAILABLE","manifest_sha256":"oldsha"}`,
		`{"state":"AVAILABLE","manifest_sha256":"newsha"}`)
	updated, sha, err := resolveImage(t.Context(), newTestImageClient(t, url), "ubuntu", time.Second)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, "newsha", sha)
}
