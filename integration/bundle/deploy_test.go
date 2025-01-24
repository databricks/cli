package bundle_test

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
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
	bundleRoot := initTestTemplate(t, ctx, "uc_schema", map[string]any{
		"unique_id": uniqueId,
	})

	deployBundle(t, ctx, bundleRoot)

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

func TestBundleDeployUcSchema(t *testing.T) {
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
	deployBundle(t, ctx, bundleRoot)

	// Assert the schema is deleted
	_, err = w.Schemas.GetByFullName(ctx, strings.Join([]string{catalogName, schemaName}, "."))
	apiErr := &apierr.APIError{}
	assert.ErrorAs(t, err, &apiErr)
	assert.Equal(t, "SCHEMA_DOES_NOT_EXIST", apiErr.ErrorCode)
}

func TestBundleDeployUcSchemaFailsWithoutAutoApprove(t *testing.T) {
	ctx, wt := acc.UcWorkspaceTest(t)
	w := wt.W

	uniqueId := uuid.New().String()
	bundleRoot := setupUcSchemaBundle(t, ctx, w, uniqueId)

	// Remove the UC schema from the resource configuration.
	err := os.Remove(filepath.Join(bundleRoot, "schema.yml"))
	require.NoError(t, err)

	// Redeploy the bundle
	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	ctx = env.Set(ctx, "TERM", "dumb")
	c := testcli.NewRunner(t, ctx, "bundle", "deploy", "--force-lock")
	stdout, stderr, err := c.Run()

	assert.EqualError(t, err, root.ErrAlreadyPrinted.Error())
	assert.Contains(t, stderr.String(), "The following UC schemas will be deleted or recreated. Any underlying data may be lost:\n  delete schema bar")
	assert.Contains(t, stdout.String(), "the deployment requires destructive actions, but current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")
}

func TestBundlePipelineDeleteWithoutAutoApprove(t *testing.T) {
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

	// Redeploy the bundle. Expect it to fail because deleting the pipeline requires --auto-approve.
	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	ctx = env.Set(ctx, "TERM", "dumb")
	c := testcli.NewRunner(t, ctx, "bundle", "deploy", "--force-lock")
	stdout, stderr, err := c.Run()

	assert.EqualError(t, err, root.ErrAlreadyPrinted.Error())
	assert.Contains(t, stderr.String(), `This action will result in the deletion or recreation of the following DLT Pipelines along with the
Streaming Tables (STs) and Materialized Views (MVs) managed by them. Recreating the Pipelines will
restore the defined STs and MVs through full refresh. Note that recreation is necessary when pipeline
properties such as the 'catalog' or 'storage' are changed:
  delete pipeline bar`)
	assert.Contains(t, stdout.String(), "the deployment requires destructive actions, but current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")
}

func TestBundlePipelineRecreateWithoutAutoApprove(t *testing.T) {
	ctx, wt := acc.UcWorkspaceTest(t)
	w := wt.W
	uniqueId := uuid.New().String()

	bundleRoot := initTestTemplate(t, ctx, "recreate_pipeline", map[string]any{
		"unique_id": uniqueId,
	})

	deployBundle(t, ctx, bundleRoot)

	t.Cleanup(func() {
		destroyBundle(t, ctx, bundleRoot)
	})

	// Assert the pipeline is created
	pipelineName := "test-pipeline-" + uniqueId
	pipeline, err := w.Pipelines.GetByName(ctx, pipelineName)
	require.NoError(t, err)
	require.Equal(t, pipelineName, pipeline.Name)

	// Redeploy the bundle, pointing the DLT pipeline to a different UC catalog.
	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	ctx = env.Set(ctx, "TERM", "dumb")
	c := testcli.NewRunner(t, ctx, "bundle", "deploy", "--force-lock", "--var=\"catalog=whatever\"")
	stdout, stderr, err := c.Run()

	assert.EqualError(t, err, root.ErrAlreadyPrinted.Error())
	assert.Contains(t, stderr.String(), `This action will result in the deletion or recreation of the following DLT Pipelines along with the
Streaming Tables (STs) and Materialized Views (MVs) managed by them. Recreating the Pipelines will
restore the defined STs and MVs through full refresh. Note that recreation is necessary when pipeline
properties such as the 'catalog' or 'storage' are changed:
  recreate pipeline foo`)
	assert.Contains(t, stdout.String(), "the deployment requires destructive actions, but current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")
}

func TestDeployBasicBundleLogs(t *testing.T) {
	ctx, wt := acc.WorkspaceTest(t)

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

	currentUser, err := wt.W.CurrentUser.Me(ctx)
	require.NoError(t, err)

	stdout, stderr := blackBoxRun(t, ctx, root, "bundle", "deploy")
	assert.Equal(t, strings.Join([]string{
		fmt.Sprintf("Uploading bundle files to /Workspace/Users/%s/.bundle/%s/files...", currentUser.UserName, uniqueId),
		"Deploying resources...",
		"Updating deployment state...",
		"Deployment complete!\n",
	}, "\n"), stderr)
	assert.Equal(t, "", stdout)
}

func TestDeployUcVolume(t *testing.T) {
	ctx, wt := acc.UcWorkspaceTest(t)
	w := wt.W

	uniqueId := uuid.New().String()
	bundleRoot := initTestTemplate(t, ctx, "volume", map[string]any{
		"unique_id": uniqueId,
	})

	deployBundle(t, ctx, bundleRoot)

	t.Cleanup(func() {
		destroyBundle(t, ctx, bundleRoot)
	})

	// Assert the volume is created successfully
	catalogName := "main"
	schemaName := "schema1-" + uniqueId
	volumeName := "my_volume"
	fullName := fmt.Sprintf("%s.%s.%s", catalogName, schemaName, volumeName)
	volume, err := w.Volumes.ReadByName(ctx, fullName)
	require.NoError(t, err)
	require.Equal(t, volume.Name, volumeName)
	require.Equal(t, catalogName, volume.CatalogName)
	require.Equal(t, schemaName, volume.SchemaName)

	// Assert that the grants were successfully applied.
	grants, err := w.Grants.GetBySecurableTypeAndFullName(ctx, catalog.SecurableTypeVolume, fullName)
	require.NoError(t, err)
	assert.Len(t, grants.PrivilegeAssignments, 1)
	assert.Equal(t, "account users", grants.PrivilegeAssignments[0].Principal)
	assert.Equal(t, []catalog.Privilege{catalog.PrivilegeWriteVolume}, grants.PrivilegeAssignments[0].Privileges)

	// Recreation of the volume without --auto-approve should fail since prompting is not possible
	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	ctx = env.Set(ctx, "TERM", "dumb")
	stdout, stderr, err := testcli.NewRunner(t, ctx, "bundle", "deploy", "--var=schema_name=${resources.schemas.schema2.name}").Run()
	assert.Error(t, err)
	assert.Contains(t, stderr.String(), `This action will result in the deletion or recreation of the following volumes.
For managed volumes, the files stored in the volume are also deleted from your
cloud tenant within 30 days. For external volumes, the metadata about the volume
is removed from the catalog, but the underlying files are not deleted:
  recreate volume foo`)
	assert.Contains(t, stdout.String(), "the deployment requires destructive actions, but current console does not support prompting. Please specify --auto-approve if you would like to skip prompts and proceed")

	// Successfully recreate the volume with --auto-approve
	ctx = env.Set(ctx, "BUNDLE_ROOT", bundleRoot)
	ctx = env.Set(ctx, "TERM", "dumb")
	_, _, err = testcli.NewRunner(t, ctx, "bundle", "deploy", "--var=schema_name=${resources.schemas.schema2.name}", "--auto-approve").Run()
	assert.NoError(t, err)

	// Assert the volume is updated successfully
	schemaName = "schema2-" + uniqueId
	fullName = fmt.Sprintf("%s.%s.%s", catalogName, schemaName, volumeName)
	volume, err = w.Volumes.ReadByName(ctx, fullName)
	require.NoError(t, err)
	require.Equal(t, volume.Name, volumeName)
	require.Equal(t, catalogName, volume.CatalogName)
	require.Equal(t, schemaName, volume.SchemaName)

	// assert that the grants were applied / retained on recreate.
	grants, err = w.Grants.GetBySecurableTypeAndFullName(ctx, catalog.SecurableTypeVolume, fullName)
	require.NoError(t, err)
	assert.Len(t, grants.PrivilegeAssignments, 1)
	assert.Equal(t, "account users", grants.PrivilegeAssignments[0].Principal)
	assert.Equal(t, []catalog.Privilege{catalog.PrivilegeWriteVolume}, grants.PrivilegeAssignments[0].Privileges)
}
