package bundle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal"
	"github.com/databricks/cli/internal/acc"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/files"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUcSchemaBundle(t *testing.T, ctx context.Context, w *databricks.WorkspaceClient, uniqueId string) string {
	bundleRoot, err := initTestTemplate(t, ctx, "uc_schema", map[string]any{
		"unique_id": uniqueId,
	})
	require.NoError(t, err)

	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	t.Cleanup(func() {
		destroyBundle(t, ctx, bundleRoot)
	})

	// Assert the schema is created
	catalogName := "main"
	schemaName := "test-schema-" + uniqueId
	schema, err := w.Schemas.GetByFullName(ctx, strings.Join([]string{catalogName, schemaName}, "."))
	require.NoError(t, err)
	require.Equal(t, strings.Join([]string{catalogName, schemaName}, "."), schema.FullName)
	require.Equal(t, "This schema was created from DABs", schema.Comment)

	// Assert the pipeline is created
	pipelineName := "test-pipeline-" + uniqueId
	pipeline, err := w.Pipelines.GetByName(ctx, pipelineName)
	require.NoError(t, err)
	require.Equal(t, pipelineName, pipeline.Name)
	id := pipeline.PipelineId

	// Assert the pipeline uses the schema
	i, err := w.Pipelines.GetByPipelineId(ctx, id)
	require.NoError(t, err)
	require.Equal(t, catalogName, i.Spec.Catalog)
	require.Equal(t, strings.Join([]string{catalogName, schemaName}, "."), i.Spec.Target)

	// Create a volume in the schema, and add a file to it. This ensures that the
	// schema has some data in it and deletion will fail unless the generated
	// terraform configuration has force_destroy set to true.
	volumeName := "test-volume-" + uniqueId
	volume, err := w.Volumes.Create(ctx, catalog.CreateVolumeRequestContent{
		CatalogName: catalogName,
		SchemaName:  schemaName,
		Name:        volumeName,
		VolumeType:  catalog.VolumeTypeManaged,
	})
	require.NoError(t, err)
	require.Equal(t, volume.Name, volumeName)

	fileName := "test-file-" + uniqueId
	err = w.Files.Upload(ctx, files.UploadRequest{
		Contents: io.NopCloser(strings.NewReader("Hello, world!")),
		FilePath: fmt.Sprintf("/Volumes/%s/%s/%s/%s", catalogName, schemaName, volumeName, fileName),
	})
	require.NoError(t, err)

	return bundleRoot
}

func TestAccBundleDeployUcSchema(t *testing.T) {
	ctx, wt := acc.UcWorkspaceTest(t)
	w := wt.W

	uniqueId := uuid.New().String()
	schemaName := "test-schema-" + uniqueId
	catalogName := "main"

	bundleRoot := setupUcSchemaBundle(t, ctx, w, uniqueId)

	// Remove the UC schema from the resource configuration.
	err := os.Remove(filepath.Join(bundleRoot, "schema.yml"))
	require.NoError(t, err)

	// Redeploy the bundle
	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	// Assert the schema is deleted
	_, err = w.Schemas.GetByFullName(ctx, strings.Join([]string{catalogName, schemaName}, "."))
	apiErr := &apierr.APIError{}
	assert.True(t, errors.As(err, &apiErr))
	assert.Equal(t, "SCHEMA_DOES_NOT_EXIST", apiErr.ErrorCode)
}

func TestAccBundleDeployUcSchemaFailsWithoutAutoApprove(t *testing.T) {
	ctx, wt := acc.UcWorkspaceTest(t)
	w := wt.W

	uniqueId := uuid.New().String()
	bundleRoot := setupUcSchemaBundle(t, ctx, w, uniqueId)

	// Remove the UC schema from the resource configuration.
	err := os.Remove(filepath.Join(bundleRoot, "schema.yml"))
	require.NoError(t, err)

	// Redeploy the bundle
	t.Setenv("BUNDLE_ROOT", bundleRoot)
	t.Setenv("TERM", "dumb")
	c := internal.NewCobraTestRunnerWithContext(t, ctx, "bundle", "deploy", "--force-lock")
	stdout, stderr, err := c.Run()

	assert.EqualError(t, err, root.ErrAlreadyPrinted.Error())
	assert.Contains(t, stderr.String(), "The following UC schemas will be deleted or recreated. Any underlying data may be lost:\n  delete schema bar")
	assert.Contains(t, stdout.String(), "the deployment requires destructive actions, but current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")
}

func TestAccBundlePipelineDeleteWithoutAutoApprove(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)
	w := wt.W

	nodeTypeId := internal.GetNodeTypeId(env.Get(ctx, "CLOUD_ENV"))
	uniqueId := uuid.New().String()
	bundleRoot, err := initTestTemplate(t, ctx, "deploy_then_remove_resources", map[string]any{
		"unique_id":     uniqueId,
		"node_type_id":  nodeTypeId,
		"spark_version": defaultSparkVersion,
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

	// assert job is created
	jobName := "test-bundle-job-" + uniqueId
	job, err := w.Jobs.GetBySettingsName(ctx, jobName)
	require.NoError(t, err)
	assert.Equal(t, job.Settings.Name, jobName)

	// delete resources.yml
	err = os.Remove(filepath.Join(bundleRoot, "resources.yml"))
	require.NoError(t, err)

	// Redeploy the bundle. Expect it to fail because deleting the pipeline requires --auto-approve.
	t.Setenv("BUNDLE_ROOT", bundleRoot)
	t.Setenv("TERM", "dumb")
	c := internal.NewCobraTestRunnerWithContext(t, ctx, "bundle", "deploy", "--force-lock")
	stdout, stderr, err := c.Run()

	assert.EqualError(t, err, root.ErrAlreadyPrinted.Error())
	assert.Contains(t, stderr.String(), `This action will result in the deletion or recreation of the following DLT Pipelines along with the
Streaming Tables (STs) and Materialized Views (MVs) managed by them. Recreating the Pipelines will
restore the defined STs and MVs through full refresh. Note that recreation is necessary when pipeline
properties such as the 'catalog' or 'storage' are changed:
  delete pipeline bar`)
	assert.Contains(t, stdout.String(), "the deployment requires destructive actions, but current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")

}

func TestAccBundlePipelineRecreateWithoutAutoApprove(t *testing.T) {
	ctx, wt := acc.UcWorkspaceTest(t)
	w := wt.W
	uniqueId := uuid.New().String()

	bundleRoot, err := initTestTemplate(t, ctx, "recreate_pipeline", map[string]any{
		"unique_id": uniqueId,
	})
	require.NoError(t, err)

	err = deployBundle(t, ctx, bundleRoot)
	require.NoError(t, err)

	t.Cleanup(func() {
		destroyBundle(t, ctx, bundleRoot)
	})

	// Assert the pipeline is created
	pipelineName := "test-pipeline-" + uniqueId
	pipeline, err := w.Pipelines.GetByName(ctx, pipelineName)
	require.NoError(t, err)
	require.Equal(t, pipelineName, pipeline.Name)

	// Redeploy the bundle, pointing the DLT pipeline to a different UC catalog.
	t.Setenv("BUNDLE_ROOT", bundleRoot)
	t.Setenv("TERM", "dumb")
	c := internal.NewCobraTestRunnerWithContext(t, ctx, "bundle", "deploy", "--force-lock", "--var=\"catalog=whatever\"")
	stdout, stderr, err := c.Run()

	assert.EqualError(t, err, root.ErrAlreadyPrinted.Error())
	assert.Contains(t, stderr.String(), `This action will result in the deletion or recreation of the following DLT Pipelines along with the
Streaming Tables (STs) and Materialized Views (MVs) managed by them. Recreating the Pipelines will
restore the defined STs and MVs through full refresh. Note that recreation is necessary when pipeline
properties such as the 'catalog' or 'storage' are changed:
  recreate pipeline foo`)
	assert.Contains(t, stdout.String(), "the deployment requires destructive actions, but current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")
}

func TestAccDeployBasicBundleLogs(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

	nodeTypeId := internal.GetNodeTypeId(env.Get(ctx, "CLOUD_ENV"))
	uniqueId := uuid.New().String()
	root, err := initTestTemplate(t, ctx, "basic", map[string]any{
		"unique_id":     uniqueId,
		"node_type_id":  nodeTypeId,
		"spark_version": defaultSparkVersion,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		err = destroyBundle(t, ctx, root)
		require.NoError(t, err)
	})

	currentUser, err := wt.W.CurrentUser.Me(ctx)
	require.NoError(t, err)

	stdout, stderr := blackBoxRun(t, root, "bundle", "deploy")
	assert.Equal(t, strings.Join([]string{
		fmt.Sprintf("Uploading bundle files to /Workspace/Users/%s/.bundle/%s/files...", currentUser.UserName, uniqueId),
		"Deploying resources...",
		"Updating deployment state...",
		"Deployment complete!\n",
	}, "\n"), stderr)
	assert.Equal(t, "", stdout)
}
