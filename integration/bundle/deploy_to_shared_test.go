package bundle_test

import (
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeployBasicToSharedWorkspacePath(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	uniqueId := uuid.New().String()

	currentUser, err := wt.W.CurrentUser.Me(ctx)
	require.NoError(t, err)

	bundleRoot := initTestTemplate(t, ctx, "basic", map[string]any{
		"unique_id":     uniqueId,
		"node_type_id":  nodeTypeId,
		"spark_version": defaultSparkVersion,
		"root_path":     "/Shared/" + currentUser.UserName,
	})

	t.Cleanup(func() {
		destroyBundle(wt, ctx, bundleRoot)
	})

	deployBundle(wt, ctx, bundleRoot)
}
