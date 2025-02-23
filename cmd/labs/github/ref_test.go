package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileFromRef(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/databrickslabs/ucx/main/README.md" {
			_, err := w.Write([]byte(`abc`))
			if !assert.NoError(t, err) {
				return
			}
			return
		}
		t.Logf("Requested: %s", r.URL.Path)
		panic("stub required")
	}))
	defer server.Close()

	ctx := context.Background()
	ctx = WithUserContentOverride(ctx, server.URL)

	raw, err := ReadFileFromRef(ctx, "databrickslabs", "ucx", "main", "README.md")
	assert.NoError(t, err)
	assert.Equal(t, []byte("abc"), raw)
}

func TestDownloadZipball(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/databrickslabs/ucx/zipball/main" {
			_, err := w.Write([]byte(`abc`))
			if !assert.NoError(t, err) {
				return
			}
			return
		}
		t.Logf("Requested: %s", r.URL.Path)
		panic("stub required")
	}))
	defer server.Close()

	ctx := context.Background()
	ctx = WithApiOverride(ctx, server.URL)

	raw, err := DownloadZipball(ctx, "databrickslabs", "ucx", "main")
	assert.NoError(t, err)
	assert.Equal(t, []byte("abc"), raw)
}
