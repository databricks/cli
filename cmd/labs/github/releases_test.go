package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
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

	ctx := context.Background()
	ctx = WithApiOverride(ctx, server.URL)

	r := NewReleaseCache("databricks", "cli", t.TempDir(), false)
	all, err := r.Load(ctx)
	assert.NoError(t, err)
	assert.Len(t, all, 2)

	// no call is made
	_, err = r.Load(ctx)
	assert.NoError(t, err)
}
