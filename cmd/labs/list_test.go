package labs_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/cli/cmd/labs/github"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListingWorks(t *testing.T) {
	ctx := t.Context()
	ctx = env.WithUserHomeDir(ctx, "project/testdata/installed-in-home")
	c := testcli.NewRunner(t, ctx, "labs", "list")
	stdout, _, err := c.Run()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ucx")
	// blueprint is in the repositories cache fixture but not in the
	// installable-repositories cache fixture, proving the latter is rendered.
	require.NotContains(t, stdout.String(), "blueprint")
}

func TestListingFiltersReposWithoutLabsYml(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/users/databrickslabs/repos":
			_, err := w.Write([]byte(`[
				{"name": "ucx", "description": "Unity Catalog Migrations", "default_branch": "main"},
				{"name": "brickster", "description": "R interface to Databricks", "default_branch": "main"}
			]`))
			assert.NoError(t, err)
		case "/databrickslabs/ucx/main/labs.yml":
			_, err := w.Write([]byte("name: ucx"))
			assert.NoError(t, err)
		case "/databrickslabs/brickster/main/labs.yml":
			w.WriteHeader(http.StatusNotFound)
		default:
			t.Logf("Requested: %s", r.URL.Path)
			t.FailNow()
		}
	}))
	defer server.Close()
	ctx := t.Context()
	ctx = github.WithApiOverride(ctx, server.URL)
	ctx = github.WithUserContentOverride(ctx, server.URL)
	ctx = env.WithUserHomeDir(ctx, t.TempDir())

	c := testcli.NewRunner(t, ctx, "labs", "list")
	stdout, _, err := c.Run()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ucx")
	require.NotContains(t, stdout.String(), "brickster")
}
