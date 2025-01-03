package acc

import (
	"fmt"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/catalog"
	"github.com/databricks/databricks-sdk-go/service/files"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/require"
)

func TemporaryWorkspaceDir(t *WorkspaceT, name ...string) string {
	ctx := t.ctx
	me, err := t.W.CurrentUser.Me(ctx)
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
		err := t.W.Workspace.Delete(ctx, workspace.Delete{
			Path:      basePath,
			Recursive: true,
		})
		if err == nil || apierr.IsMissing(err) {
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
		err := t.W.Dbfs.Delete(ctx, files.Delete{
			Path:      path,
			Recursive: true,
		})
		if err == nil || apierr.IsMissing(err) {
			return
		}
		t.Logf("Unable to remove temporary DBFS directory %s: %#v", path, err)
	})

	return path
}

func TemporaryRepo(t *WorkspaceT, url string) string {
	ctx := t.ctx
	me, err := t.W.CurrentUser.Me(ctx)
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
		err := t.W.Repos.Delete(ctx, workspace.DeleteRepoRequest{
			RepoId: resp.Id,
		})
		if err == nil || apierr.IsMissing(err) {
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
		err := w.Schemas.Delete(ctx, catalog.DeleteSchemaRequest{
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
		err := w.Volumes.Delete(ctx, catalog.DeleteVolumeRequest{
			Name: volume.FullName,
		})
		require.NoError(t, err)
	})

	return fmt.Sprintf("/Volumes/%s/%s/%s", "main", schema.Name, volume.Name)
}
