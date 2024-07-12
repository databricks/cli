package bundle

import (
	"os"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/acc"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func runSparkJarTest(t *testing.T, sparkVersion string) {
	t.Skip("Temporarily skipping the test until auth / permission issues for UC volumes are resolved.")

	env := internal.GetEnvOrSkipTest(t, "CLOUD_ENV")
	t.Log(env)

	if os.Getenv("TEST_METASTORE_ID") == "" {
		t.Skip("Skipping tests that require a UC Volume when metastore id is not set.")
	}

	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W
	volumePath := internal.TemporaryUcVolume(t, w)

	nodeTypeId := internal.GetNodeTypeId(env)
	tmpDir := t.TempDir()
	bundleRoot, err := initTestTemplateWithBundleRoot(t, ctx, "spark_jar_task", map[string]any{
		"node_type_id":  nodeTypeId,
		"unique_id":     uuid.New().String(),
		"spark_version": sparkVersion,
		"root":          tmpDir,
		"artifact_path": volumePath,
	}, tmpDir)
	require.NoError(t, err)

	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	t.Cleanup(func() {
		destroyBundle(t, ctx, bundleRoot)
	})

	out, err := runResource(t, ctx, bundleRoot, "jar_job")
	require.NoError(t, err)
	require.Contains(t, out, "Hello from Jar!")
}

func TestAccSparkJarTaskDeployAndRunOnVolumes(t *testing.T) {
	runSparkJarTest(t, "14.3.x-scala2.12")
}
