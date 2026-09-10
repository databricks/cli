package aircmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClassifyRegistrationError(t *testing.T) {
	cases := []struct {
		name      string
		err       error
		kind      string
		retryable bool
	}{
		{"auth", apierr.ErrUnauthenticated, "PERMANENT", false},
		{"permission", apierr.ErrPermissionDenied, "PERMANENT", false},
		{"not found", apierr.ErrNotFound, "PERMANENT", false},
		{"bad request", apierr.ErrBadRequest, "PERMANENT", false},
		{"conflict", apierr.ErrResourceConflict, "PERMANENT", false},
		{"canceled", context.Canceled, "PERMANENT", false},
		{"upload failed", fmt.Errorf("%w: boom", errImageUploadFailed), "PERMANENT", false},
		{"unknown error", errors.New("something odd"), "PERMANENT", false},
		{"wait timeout", fmt.Errorf("%w within 1m0s", errImageWaitTimeout), "TRANSIENT", true},
		{"rate limited", apierr.ErrTooManyRequests, "TRANSIENT", true},
		{"server error", apierr.ErrInternalError, "TRANSIENT", true},
		{"unavailable", apierr.ErrTemporarilyUnavailable, "TRANSIENT", true},
		{"deadline exceeded", apierr.ErrDeadlineExceeded, "TRANSIENT", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kind, retryable := classifyRegistrationError(tc.err)
			assert.Equal(t, tc.kind, kind)
			assert.Equal(t, tc.retryable, retryable)
		})
	}
}

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

func TestRegistrationError(t *testing.T) {
	authErr := apierr.ErrPermissionDenied
	credErr := errors.New(`creating secret scope "docker-credentials-you@example.com" was denied`)

	// Credentials were found but couldn't be stored: report that as the cause,
	// not "run docker login" — the user already has working credentials.
	err := registrationError("nvcr.io/org/img:1.0", authErr, credErr)
	assert.Contains(t, err.Error(), "requires credentials, and the credentials found in your local Docker config could not be stored")
	assert.Contains(t, err.Error(), "was denied")
	assert.NotContains(t, err.Error(), "run `docker login`")

	// No credential-storage problem: the docker login hint is the right guidance.
	err = registrationError("nvcr.io/org/img:1.0", authErr, nil)
	assert.Contains(t, err.Error(), "run `docker login`")

	// A non-auth failure passes through untouched.
	other := errors.New("boom")
	assert.Equal(t, other, registrationError("nvcr.io/org/img:1.0", other, credErr))
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
	updated, sha, err := resolveImage(t.Context(), newTestImageClient(t, url), "ubuntu", "", "", time.Second)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, "newsha", sha)
}

func TestResolveImageDigestUnchanged(t *testing.T) {
	body := `{"state":"AVAILABLE","manifest_sha256":"samesha"}`
	url := imageServer(t, `{"image":`+body+`}`, body)
	updated, sha, err := resolveImage(t.Context(), newTestImageClient(t, url), "ubuntu", "", "", time.Second)
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
	updated, sha, err := resolveImage(t.Context(), newTestImageClient(t, url), "ubuntu", "", "", time.Second)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, "newsha", sha)
}

// credRejectingImageServer 401s a POST that carries credentials and returns
// AVAILABLE for an anonymous POST, so a test can exercise the stale-credential
// anonymous retry.
func credRejectingImageServer(t *testing.T, credentialedPOSTs *int) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case imagesAPIPath + ":get":
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"error_code":"NOT_FOUND","message":"not registered"}`))
		case imagesAPIPath:
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), "credentials_scope") {
				*credentialedPOSTs++
				w.WriteHeader(http.StatusForbidden)
				_, _ = w.Write([]byte(`{"error_code":"PERMISSION_DENIED","message":"denied"}`))
				return
			}
			_, _ = w.Write([]byte(`{"image":{"state":"AVAILABLE","manifest_sha256":"pubsha"}}`))
		default:
			_, _ = w.Write([]byte(`{}`))
		}
	}))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestRegisterWithCredentialFallbackRetriesAnonymously(t *testing.T) {
	var credentialedPOSTs int
	url := credRejectingImageServer(t, &credentialedPOSTs)
	updated, sha, err := registerWithCredentialFallback(t.Context(), newTestImageClient(t, url), "nvcr.io/org/img:1.0", imageCredentials{scope: "scope", key: "key", discovered: true}, time.Second)
	require.NoError(t, err)
	assert.True(t, updated)
	assert.Equal(t, "pubsha", sha)
	assert.Equal(t, 1, credentialedPOSTs, "should try once with creds, then retry anonymously")
}

func TestRegisterWithCredentialFallbackNoRetryWithoutCreds(t *testing.T) {
	// Without credentials there is nothing stale to fall back from, so an auth
	// failure surfaces directly.
	var credentialedPOSTs int
	url := credRejectingImageServer(t, &credentialedPOSTs)
	_, _, err := registerWithCredentialFallback(t.Context(), newTestImageClient(t, url), "nvcr.io/org/img:1.0", imageCredentials{}, time.Second)
	require.NoError(t, err) // anonymous POST succeeds on this server
	assert.Equal(t, 0, credentialedPOSTs)
}

func TestRegisterWithCredentialFallbackNoRetryForExplicitCreds(t *testing.T) {
	// The user named these credentials, so a rejection is the real answer: do not
	// silently retry without them.
	var credentialedPOSTs int
	url := credRejectingImageServer(t, &credentialedPOSTs)
	_, _, err := registerWithCredentialFallback(t.Context(), newTestImageClient(t, url), "nvcr.io/org/img:1.0", imageCredentials{scope: "scope", key: "key"}, time.Second)
	require.Error(t, err)
	assert.Equal(t, 1, credentialedPOSTs, "should try once with creds and stop")
}
