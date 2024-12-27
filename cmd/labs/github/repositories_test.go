package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepositories(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/users/databrickslabs/repos" {
			_, err := w.Write([]byte(`[{"name": "x"}]`))
			require.NoError(t, err)
			return
		}
		t.Logf("Requested: %s", r.URL.Path)
		panic("stub required")
	}))
	defer server.Close()

	ctx := context.Background()
	ctx = WithApiOverride(ctx, server.URL)

	r := NewRepositoryCache("databrickslabs", t.TempDir())
	all, err := r.Load(ctx)
	assert.NoError(t, err)
	assert.True(t, len(all) > 0)
}
