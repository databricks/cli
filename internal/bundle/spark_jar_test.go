package bundle

import (
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func runSparkJarTest(t *testing.T, sparkVersion string) {
	env := internal.GetEnvOrSkipTest(t, "CLOUD_ENV")
	t.Log(env)

	var nodeTypeId string
	if env == "gcp" {
		nodeTypeId = "n1-standard-4"
	} else if env == "aws" {
		nodeTypeId = "i3.xlarge"
	} else {
		nodeTypeId = "Standard_DS4_v2"
	}

	tmpDir := t.TempDir()
	bundleRoot, err := initTestTemplateWithBundleRoot(t, "spark_jar_task", map[string]any{
		"node_type_id":  nodeTypeId,
		"unique_id":     uuid.New().String(),
		"spark_version": sparkVersion,
		"root": tmpDir,
	}, tmpDir)
	require.NoError(t, err)

	err = deployBundle(t, bundleRoot)
	require.NoError(t, err)

	t.Cleanup(func() {
		destroyBundle(t, bundleRoot)
	})

	out, err := runResource(t, bundleRoot, "jar_job")
	require.NoError(t, err)
	require.Contains(t, out, "Hello from Jar!")
}

func TestAccSparkJarTaskDeployAndRun(t *testing.T) {
	runSparkJarTest(t, "14.1.x-scala2.12")
}
