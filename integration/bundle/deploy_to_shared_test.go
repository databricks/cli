//go:build integration

package bundle_integration

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccDeployBasicToSharedWorkspacePath(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	uniqueId := uuid.New().String()

	currentUser, err := wt.W.CurrentUser.Me(ctx)
	require.NoError(t, err)

	bundleRoot, err := initTestTemplate(t, ctx, "basic", map[string]any{
		"unique_id":     uniqueId,
		"node_type_id":  nodeTypeId,
		"spark_version": defaultSparkVersion,
		"root_path":     fmt.Sprintf("/Shared/%s", currentUser.UserName),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = destroyBundle(wt, ctx, bundleRoot)
		require.NoError(wt, err)
	})

	err = deployBundle(wt, ctx, bundleRoot)
	require.NoError(wt, err)
}
