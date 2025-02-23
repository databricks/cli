package bundle_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/testdiff"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBasicBundleDeployWithFailOnActiveRuns(t *testing.T) {
	ctx, _ := acc.WorkspaceTest(t)

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	uniqueId := uuid.New().String()
	root := initTestTemplate(t, ctx, "basic", map[string]any{
		"unique_id":     uniqueId,
		"node_type_id":  nodeTypeId,
		"spark_version": defaultSparkVersion,
	})

	t.Cleanup(func() {
		destroyBundle(t, ctx, root)
	})

	// deploy empty bundle
	deployBundleWithFlags(t, ctx, root, []string{"--fail-on-active-runs"})

	// Remove .databricks directory to simulate a fresh deployment
	require.NoError(t, os.RemoveAll(filepath.Join(root, ".databricks")))

	// deploy empty bundle again
	deployBundleWithFlags(t, ctx, root, []string{"--fail-on-active-runs"})
}

func TestBasicBundleDeployWithDoubleUnderscoreVariables(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	uniqueId := uuid.New().String()
	root := initTestTemplate(t, ctx, "basic_with_variables", map[string]any{
		"unique_id":     uniqueId,
		"node_type_id":  nodeTypeId,
		"spark_version": defaultSparkVersion,
	})

	currentUser, err := wt.W.CurrentUser.Me(ctx)
	require.NoError(t, err)

	ctx, replacements := testdiff.WithReplacementsMap(ctx)
	replacements.Set(uniqueId, "$UNIQUE_PRJ")
	replacements.Set(currentUser.UserName, "$USERNAME")

	t.Cleanup(func() {
		destroyBundle(t, ctx, root)
	})

	testutil.Chdir(t, root)
	testcli.AssertOutput(
		t,
		ctx,
		[]string{"bundle", "validate"},
		testutil.TestData("testdata/basic_with_variables/bundle_validate.txt"),
	)
	testcli.AssertOutput(
		t,
		ctx,
		[]string{"bundle", "deploy", "--force-lock", "--auto-approve"},
		testutil.TestData("testdata/basic_with_variables/bundle_deploy.txt"),
	)
}
