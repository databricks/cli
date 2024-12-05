package bundle

import (
	"fmt"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/libs/env"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDeployBasicToSharedWorkspacePath(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	nodeTypeId := internal.GetNodeTypeId(env.Get(ctx, "CLOUD_ENV"))
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
		err = destroyBundle(wt.T, ctx, bundleRoot)
		require.NoError(wt.T, err)
	})

	err = deployBundle(wt.T, ctx, bundleRoot)
	require.NoError(wt.T, err)
}
