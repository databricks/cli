package labs_test

import (
	"testing"

	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/env"
)

func TestListsInstalledProjects(t *testing.T) {
	ctx := t.Context()
	ctx = env.WithUserHomeDir(ctx, "project/testdata/installed-in-home")
	r := testcli.NewRunner(t, ctx, "labs", "installed")
	r.RunAndExpectOutput(`
	Name       Description        Version
	blueprint  Blueprint Project  v0.3.15
	`)
}
