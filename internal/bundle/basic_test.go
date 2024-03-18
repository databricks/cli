package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/libs/env"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestAccBasicBundleDeployWithFailOnActiveRuns(t *testing.T) {
	ctx, _ := acc.WorkspaceTest(t)

	nodeTypeId := internal.GetNodeTypeId(env.Get(ctx, "CLOUD_ENV"))
	uniqueId := uuid.New().String()
	root, err := initTestTemplate(t, ctx, "basic", map[string]any{
		"unique_id":     uniqueId,
		"node_type_id":  nodeTypeId,
		"spark_version": "13.2.x-snapshot-scala2.12",
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = destroyBundle(t, ctx, root)
		require.NoError(t, err)
	})

	// deploy empty bundle
	err = deployBundleWithFlags(t, ctx, root, []string{"--fail-on-active-runs"})
	require.NoError(t, err)

	// Remove .databricks directory to simulate a fresh deployment
	err = os.RemoveAll(filepath.Join(root, ".databricks"))
	require.NoError(t, err)

	// deploy empty bundle again
	err = deployBundleWithFlags(t, ctx, root, []string{"--fail-on-active-runs"})
	require.NoError(t, err)
}
