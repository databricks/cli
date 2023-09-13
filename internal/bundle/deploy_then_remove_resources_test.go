package bundle

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/databricks/databricks-sdk-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccBundleDeployThenRemoveResources(t *testing.T) {
	env := internal.GetEnvOrSkipTest(t, "CLOUD_ENV")
	t.Log(env)

	uniqueId := uuid.New().String()
	bundleRoot, err := initTestTemplate(t, "deploy_then_remove_resources", map[string]any{
		"unique_id": uniqueId,
	})
	require.NoError(t, err)

	// deploy pipeline
	err = deployBundle(t, bundleRoot)
	require.NoError(t, err)

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	// assert pipeline is created
	pipelineName := "test-bundle-pipeline-" + uniqueId
	pipeline, err := w.Pipelines.GetByName(context.Background(), pipelineName)
	require.NoError(t, err)
	assert.Equal(t, pipeline.Name, pipelineName)

	// delete resources.yml
	err = os.Remove(filepath.Join(bundleRoot, "resources.yml"))
	require.NoError(t, err)

	// deploy again
	err = deployBundle(t, bundleRoot)
	require.NoError(t, err)

	// assert pipeline is deleted
	_, err = w.Pipelines.GetByName(context.Background(), pipelineName)
	assert.ErrorContains(t, err, "does not exist")

	t.Cleanup(func() {
		err = destroyBundle(t, bundleRoot)
		require.NoError(t, err)
	})
}
