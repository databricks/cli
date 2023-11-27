package labs_test

import (
	"context"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/libs/env"
)

func TestListsInstalledProjects(t *testing.T) {
	ctx := context.Background()
	ctx = env.WithUserHomeDir(ctx, "project/testdata/installed-in-home")
	r := internal.NewCobraTestRunnerWithContext(t, ctx, "labs", "installed")
	r.RunAndExpectOutput(`
	Name       Description        Version
	blueprint  Blueprint Project  v0.3.15
	`)
}
