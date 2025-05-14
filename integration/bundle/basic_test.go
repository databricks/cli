package bundle_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"
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
