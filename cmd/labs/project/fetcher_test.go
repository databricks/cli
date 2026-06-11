package project_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/databricks/cli/cmd/labs/github"
	"github.com/databricks/cli/internal/testcli"
	"github.com/stretchr/testify/assert"
)

func TestInstallerFailsForProjectWithoutLabsYml(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/databrickslabs/blueprint/v0.3.15/labs.yml" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		t.Logf("Requested: %s", r.URL.Path)
		t.FailNow()
	}))
	defer server.Close()
	ctx := installerContext(t, server)

	r := testcli.NewRunner(t, ctx, "labs", "install", "blueprint")
	_, _, err := r.Run()
	assert.ErrorIs(t, err, github.ErrNotFound)
	assert.ErrorContains(t, err, "databrickslabs/blueprint@v0.3.15 does not provide labs.yml")
	assert.ErrorContains(t, err, "cannot be installed with the Databricks CLI")
}
