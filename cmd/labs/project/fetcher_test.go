package project_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/labs/github"
	"github.com/databricks/cli/internal/testcli"
	"github.com/stretchr/testify/assert"
)

func TestInstallerFailsWhenLabsYmlMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/labs.yml") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		t.Errorf("unexpected request: %s", r.URL.Path)
		http.Error(w, "unexpected request", http.StatusInternalServerError)
	}))
	defer server.Close()
	ctx := installerContext(t, server)

	// A 404 on labs.yml has two causes: the repository ships no labs.yml (reached
	// here via the only released version, v0.3.15), or the requested version does
	// not exist (v9.9.9). Both must produce the same actionable error.
	tests := []struct {
		name    string
		ref     string
		version string
	}{
		{name: "released version without labs.yml", ref: "blueprint", version: "v0.3.15"},
		{name: "nonexistent version", ref: "blueprint@v9.9.9", version: "v9.9.9"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			r := testcli.NewRunner(t, ctx, "labs", "install", tc.ref)
			_, _, err := r.Run()
			assert.ErrorIs(t, err, github.ErrNotFound)
			assert.ErrorContains(t, err, "no labs.yml at databrickslabs/blueprint@"+tc.version)
			assert.ErrorContains(t, err, "either this version does not exist or this project cannot be installed with the Databricks CLI")
		})
	}
}
