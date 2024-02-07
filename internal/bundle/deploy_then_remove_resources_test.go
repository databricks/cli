package bundle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccBundleDeployThenRemoveResources(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	uniqueId := uuid.New().String()
	bundleRoot, err := initTestTemplate(t, ctx, "deploy_then_remove_resources", map[string]any{
		"unique_id": uniqueId,
	})
	require.NoError(t, err)

	// deploy pipeline
	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	// assert pipeline is created
	pipelineName := "test-bundle-pipeline-" + uniqueId
	pipeline, err := w.Pipelines.GetByName(ctx, pipelineName)
	require.NoError(t, err)
	assert.Equal(t, pipeline.Name, pipelineName)

	// delete resources.yml
	err = os.Remove(filepath.Join(bundleRoot, "resources.yml"))
	require.NoError(t, err)

	// deploy again
	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	// assert pipeline is deleted
	_, err = w.Pipelines.GetByName(ctx, pipelineName)
	assert.ErrorContains(t, err, "does not exist")

	t.Cleanup(func() {
		err = destroyBundle(t, ctx, bundleRoot)
		require.NoError(t, err)
	})
}
