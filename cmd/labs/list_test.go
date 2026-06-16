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
	// blueprint is in the repositories cache fixture but lacks the
	// databricks-cli-installable topic, proving the topic filter is applied.
	require.NotContains(t, stdout.String(), "blueprint")
}

func TestListingFiltersReposWithoutTopic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/users/databrickslabs/repos", r.URL.Path)
		_, err := w.Write([]byte(`[
			{"name": "ucx", "description": "Unity Catalog Migrations", "topics": ["databricks-cli-installable"]},
			{"name": "brickster", "description": "R interface to Databricks", "topics": []}
		]`))
		assert.NoError(t, err)
	}))
	defer server.Close()
	ctx := t.Context()
	ctx = github.WithApiOverride(ctx, server.URL)
	ctx = env.WithUserHomeDir(ctx, t.TempDir())

	c := testcli.NewRunner(t, ctx, "labs", "list")
	stdout, _, err := c.Run()
	require.NoError(t, err)
	require.Contains(t, stdout.String(), "ucx")
	require.NotContains(t, stdout.String(), "brickster")
}
