package aircmd

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeDockerImageURL(t *testing.T) {
	cases := map[string]string{
		// Bare Docker Hub official images get the library/ namespace.
		"ubuntu":       "docker.io/library/ubuntu:latest",
		"ubuntu:22.04": "docker.io/library/ubuntu:22.04",
		"  ubuntu  ":   "docker.io/library/ubuntu:latest",
		// User/org Docker Hub images get the registry prefix but no library/.
		"pytorch/pytorch:2.0.0": "docker.io/pytorch/pytorch:2.0.0",
		"pytorch/pytorch":       "docker.io/pytorch/pytorch:latest",
		// Explicit registries (host has a dot) are left as-is aside from a default tag.
		"nvcr.io/nvidia/pytorch:24.01":    "nvcr.io/nvidia/pytorch:24.01",
		"registry.gitlab.com/org/repo":    "registry.gitlab.com/org/repo:latest",
		"docker.io/library/ubuntu:latest": "docker.io/library/ubuntu:latest",
		// A digest takes precedence over any tag per the OCI spec.
		"ubuntu@sha256:abc":                 "docker.io/library/ubuntu@sha256:abc",
		"pytorch/pytorch:2.0.0@sha256:def":  "docker.io/pytorch/pytorch@sha256:def",
		"nvcr.io/nvidia/pytorch@sha256:xyz": "nvcr.io/nvidia/pytorch@sha256:xyz",
	}
	for in, want := range cases {
		t.Run(in, func(t *testing.T) {
			assert.Equal(t, want, normalizeDockerImageURL(in))
		})
	}
}

func TestNormalizeStatus(t *testing.T) {
	cases := map[string]imageStatus{
		"AVAILABLE": imageStatusAvailable,
		"PENDING":   imageStatusPending,
		"IMPORTING": imageStatusImporting,
		"FAILED":    imageStatusFailed,
		// Absent or unrecognized states degrade to PENDING.
		"":        imageStatusPending,
		"UNKNOWN": imageStatusPending,
	}
	for state, want := range cases {
		t.Run(state, func(t *testing.T) {
			reg := imageRegistration{State: state}
			reg.normalizeStatus()
			assert.Equal(t, want, reg.Status)
		})
	}
}

// newTestImageClient builds an imageClient pointed at srv.
func newTestImageClient(t *testing.T, host string) *imageClient {
	t.Helper()
	c, err := newImageClient(newTestWorkspaceClient(t, host))
	require.NoError(t, err)
	return c
}

func TestImageClientCreateImageUnwrapsResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == imagesAPIPath && r.Method == http.MethodPost {
			// The backend wraps the registration under "image".
			_, _ = w.Write([]byte(`{"image":{"state":"PENDING","manifest_sha256":"abc"}}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	reg, err := newTestImageClient(t, srv.URL).createImage(t.Context(), "ubuntu", "", "")
	require.NoError(t, err)
	assert.Equal(t, imageStatusPending, reg.Status)
	assert.Equal(t, "abc", reg.ManifestSHA256)
}

func TestImageClientGetImageNotFoundReturnsNil(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error_code":"NOT_FOUND","message":"not registered"}`))
	}))
	t.Cleanup(srv.Close)

	reg, err := newTestImageClient(t, srv.URL).getImage(t.Context(), "ubuntu")
	require.NoError(t, err)
	assert.Nil(t, reg)
}

func TestImageClientCheckImageAccess(t *testing.T) {
	var hit bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == imagesAPIPath+":checkImageAccess" {
			hit = true
			_, _ = w.Write([]byte(`{"publicly_accessible":true}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	got := newTestImageClient(t, srv.URL).checkImageAccess(t.Context(), "ubuntu")
	require.True(t, hit)
	require.NotNil(t, got)
	assert.True(t, *got)
}

func TestImageClientCheckImageAccessUnknownOnError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(srv.Close)

	assert.Nil(t, newTestImageClient(t, srv.URL).checkImageAccess(t.Context(), "ubuntu"))
}

func TestImageClientWaitForImageReady(t *testing.T) {
	call := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == imagesAPIPath+":get" {
			// First poll is still PENDING; second reports AVAILABLE.
			call++
			if call == 1 {
				_, _ = w.Write([]byte(`{"state":"PENDING"}`))
				return
			}
			_, _ = w.Write([]byte(`{"state":"AVAILABLE","manifest_sha256":"done"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	reg, err := newTestImageClient(t, srv.URL).waitForImageReady(t.Context(), "ubuntu", 5*time.Second, time.Millisecond)
	require.NoError(t, err)
	assert.Equal(t, imageStatusAvailable, reg.Status)
	assert.Equal(t, "done", reg.ManifestSHA256)
}

func TestImageClientWaitForImageReadyFailed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == imagesAPIPath+":get" {
			_, _ = w.Write([]byte(`{"state":"FAILED","status_message":"boom"}`))
			return
		}
		_, _ = w.Write([]byte(`{}`))
	}))
	t.Cleanup(srv.Close)

	_, err := newTestImageClient(t, srv.URL).waitForImageReady(t.Context(), "ubuntu", 5*time.Second, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "boom")
}
