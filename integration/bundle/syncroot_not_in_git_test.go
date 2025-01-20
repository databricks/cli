package bundle

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/testdiff"
)

// TODO: We should not receive an error here
func TestSyncRootNotInGitError(t *testing.T) {
	ctx, _ := acc.WorkspaceTest(t)
	tmp := t.TempDir()
	root := filepath.Join(tmp, "bundle")
	ctx, replacements := testdiff.WithReplacementsMap(ctx)
	replacements.Set(tmp, "$TMP_DIR")

	testutil.WriteFile(t, filepath.Join(root, "databricks.yml"), `bundle:
  name: test-bundle

sync:
  paths:
    - ..`)

	testutil.Chdir(t, root)
	testcli.AssertOutput(
		t,
		ctx,
		[]string{"bundle", "deploy", "--force-lock", "--auto-approve"},
		testutil.TestData("testdata/syncroot_not_in_git/bundle_deploy.txt"),
	)
}
