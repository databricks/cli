package bundle

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccBundleDestroy(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	nodeTypeId := testutil.GetCloud(t).NodeTypeID()
	uniqueId := uuid.New().String()
	bundleRoot, err := initTestTemplate(t, ctx, "deploy_then_remove_resources", map[string]any{
		"unique_id":     uniqueId,
		"node_type_id":  nodeTypeId,
		"spark_version": defaultSparkVersion,
	})
	require.NoError(t, err)

	snapshotsDir := filepath.Join(bundleRoot, ".databricks", "bundle", "default", "sync-snapshots")

	// Assert the snapshot file does not exist
	_, err = os.ReadDir(snapshotsDir)
	assert.ErrorIs(t, err, os.ErrNotExist)

	// deploy resources
	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	// Assert the snapshot file exists
	entries, err := os.ReadDir(snapshotsDir)
	assert.NoError(t, err)
	assert.Len(t, entries, 1)

	// Assert bundle deployment path is created
	remoteRoot := getBundleRemoteRootPath(w, t, uniqueId)
	_, err = w.Workspace.GetStatusByPath(ctx, remoteRoot)
	assert.NoError(t, err)

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

	// destroy bundle
	err = destroyBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	// assert pipeline is deleted
	_, err = w.Pipelines.GetByName(ctx, pipelineName)
	assert.ErrorContains(t, err, "does not exist")

	// assert job is deleted
	_, err = w.Jobs.GetBySettingsName(ctx, jobName)
	assert.ErrorContains(t, err, "does not exist")

	// Assert snapshot file is deleted
	entries, err = os.ReadDir(snapshotsDir)
	require.NoError(t, err)
	assert.Len(t, entries, 0)

	// Assert bundle deployment path is deleted
	_, err = w.Workspace.GetStatusByPath(ctx, remoteRoot)
	apiErr := &apierr.APIError{}
	assert.True(t, errors.As(err, &apiErr))
	assert.Equal(t, "RESOURCE_DOES_NOT_EXIST", apiErr.ErrorCode)
}
