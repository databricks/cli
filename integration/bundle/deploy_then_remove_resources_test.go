package bundle_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBundleDeployThenRemoveResources(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	uniqueId := uuid.New().String()
	bundleRoot := initTestTemplate(t, ctx, "deploy_then_remove_resources", map[string]any{
		"unique_id":     uniqueId,
		"node_type_id":  nodeTypeId,
		"spark_version": defaultSparkVersion,
	})

	// deploy pipeline
	deployBundle(t, ctx, bundleRoot)

	// assert pipeline is created
	pipelineName := "test-bundle-pipeline-" + uniqueId
	pipeline, err := w.Pipelines.GetByName(ctx, pipelineName)
	require.NoError(t, err)
	assert.Equal(t, pipeline.Name, pipelineName)

	// assert job is created
	jobName := "test-bundle-job-" + uniqueId
	job, err := w.Jobs.GetBySettingsName(ctx, jobName)
	require.NoError(t, err)
	assert.Equal(t, job.Settings.Name, jobName)

	// delete resources.yml
	err = os.Remove(filepath.Join(bundleRoot, "resources.yml"))
	require.NoError(t, err)

	// deploy again
	deployBundle(t, ctx, bundleRoot)

	// assert pipeline is deleted
	_, err = w.Pipelines.GetByName(ctx, pipelineName)
	assert.ErrorContains(t, err, "does not exist")

	// assert job is deleted
	_, err = w.Jobs.GetBySettingsName(ctx, jobName)
	assert.ErrorContains(t, err, "does not exist")

	t.Cleanup(func() {
		destroyBundle(t, ctx, bundleRoot)
	})
}
