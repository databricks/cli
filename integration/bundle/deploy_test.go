package bundle_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/databricks/cli/integration/internal/acc"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
