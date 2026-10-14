package acc

import (
	"context"
	"fmt"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/workspaceapi"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/files"
	sdkscim "github.com/databricks/sdk-go/scim/v1"
	"github.com/stretchr/testify/require"
)

func TemporaryWorkspaceDir(t *WorkspaceT, name ...string) string {
	ctx := t.ctx
	me, err := t.W.Scim.Me(ctx, sdkscim.MeRequest{})
	require.NoError(t, err)

	// Prefix the name with "integration-test-" to make it easier to identify.
	name = append([]string{"integration-test-"}, name...)
	basePath := fmt.Sprintf("/Users/%s/%s", me.UserName, testutil.RandomName(name...))

	t.Logf("Creating workspace directory %s", basePath)
	err = t.W.Workspace.MkdirsByPath(ctx, basePath)
	require.NoError(t, err)

	// Remove test directory on test completion.
	t.Cleanup(func() {
		t.Logf("Removing workspace directory %s", basePath)
		err := t.W.Workspace.Delete(context.WithoutCancel(ctx), workspaceapi.Delete{
			Path:      basePath,
			Recursive: true,
		})
		if err == nil || apierrs.IsMissing(err) {
			return
		}
		t.Logf("Unable to remove temporary workspace directory %s: %#v", basePath, err)
	})

	return basePath
}

func TemporaryDbfsDir(t *WorkspaceT, name ...string) string {
	ctx := t.ctx

	// Prefix the name with "integration-test-" to make it easier to identify.
	name = append([]string{"integration-test-"}, name...)
	path := "/tmp/" + testutil.RandomName(name...)

	t.Logf("Creating DBFS directory %s", path)
	err := t.W.Dbfs.MkdirsByPath(ctx, path)
	require.NoError(t, err)

	t.Cleanup(func() {
		t.Logf("Removing DBFS directory %s", path)
		err := t.W.Dbfs.Delete(context.WithoutCancel(ctx), files.Delete{
			Path:      path,
			Recursive: true,
		})
		if err == nil || apierrs.IsMissing(err) {
			return
		}
		t.Logf("Unable to remove temporary DBFS directory %s: %#v", path, err)
	})

	return path
}

func TemporaryRepo(t *WorkspaceT, url string) string {
	ctx := t.ctx
	me, err := t.W.Scim.Me(ctx, sdkscim.MeRequest{})
	require.NoError(t, err)

	// Prefix the path with "integration-test-" to make it easier to identify.
	path := fmt.Sprintf("/Repos/%s/%s", me.UserName, testutil.RandomName("integration-test-"))

	t.Logf("Creating repo: %s", path)
	resp, err := t.W.Repos.Create(ctx, workspace.CreateRepoRequest{
		Url:      url,
		Path:     path,
		Provider: "gitHub",
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		t.Logf("Removing repo: %s", path)
		err := t.W.Repos.Delete(context.WithoutCancel(ctx), workspace.DeleteRepoRequest{
			RepoId: resp.Id,
		})
		if err == nil || apierrs.IsMissing(err) {
			return
		}
		t.Logf("Unable to remove repo %s: %#v", path, err)
	})

	return path
}

// Create a new Unity Catalog volume in a catalog called "main" in the workspace.
func TemporaryVolume(t *WorkspaceT) string {
	ctx := t.ctx
	w := t.W

	// Create a schema
	schema, err := w.Schemas.Create(ctx, catalog.CreateSchema{
		CatalogName: "main",
		Name:        testutil.RandomName("test-schema-"),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		err := w.Schemas.Delete(context.WithoutCancel(ctx), catalog.DeleteSchemaRequest{
			FullName: schema.FullName,
		})
		require.NoError(t, err)
	})

	// Create a volume
	volume, err := w.Volumes.Create(ctx, catalog.CreateVolumeRequestContent{
		CatalogName: "main",
		SchemaName:  schema.Name,
		Name:        "my-volume",
		VolumeType:  catalog.VolumeTypeManaged,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		err := w.Volumes.Delete(context.WithoutCancel(ctx), catalog.DeleteVolumeRequest{
			Name: volume.FullName,
		})
		require.NoError(t, err)
	})

	return fmt.Sprintf("/Volumes/%s/%s/%s", "main", schema.Name, volume.Name)
}
