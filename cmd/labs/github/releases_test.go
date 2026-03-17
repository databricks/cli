package github

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadsReleasesForCLI(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/databricks/cli/releases" {
			_, err := w.Write([]byte(`[{"tag_name": "v1.2.3"}, {"tag_name": "v1.2.2"}]`))
			if !assert.NoError(t, err) {
				return
			}
			return
		}
		t.Logf("Requested: %s", r.URL.Path)
		panic("stub required")
	}))
	defer server.Close()

	ctx := t.Context()
	ctx = WithApiOverride(ctx, server.URL)

	r := NewReleaseCache("databricks", "cli", t.TempDir(), false)
	all, err := r.Load(ctx)
	assert.NoError(t, err)
	assert.Len(t, all, 2)

	// no call is made
	_, err = r.Load(ctx)
	assert.NoError(t, err)
}

func TestLoadsReleasesWhenOffline(t *testing.T) {
	cacheDir := t.TempDir()
	cacheFile := filepath.Join(cacheDir, "databricks-cli-releases.json")
	cache := `{"data":[{"tag_name":"v1.2.3"},{"tag_name":"v1.2.2"}]}`
	err := os.WriteFile(cacheFile, []byte(cache), 0o644)
	require.NoError(t, err)

	ctx := t.Context()
	r := NewReleaseCache("databricks", "cli", cacheDir, true)
	all, err := r.Load(ctx)
	assert.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestLoadingErrorWhenOffline(t *testing.T) {
	ctx := t.Context()
	r := NewReleaseCache("databricks", "cli", t.TempDir(), true)
	all, err := r.Load(ctx)
	assert.Error(t, err)
	assert.Nil(t, all)
}
