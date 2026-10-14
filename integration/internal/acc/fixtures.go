package acc

import (
	"context"
	"fmt"
	"net/http"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/apiclient"
	"github.com/databricks/cli/libs/apierrs"
	"github.com/databricks/cli/libs/utils"
	"github.com/databricks/cli/libs/workspaceapi"
	"github.com/databricks/sdk-go/repos/v1"
	sdkscim "github.com/databricks/sdk-go/scim/v1"
	"github.com/databricks/sdk-go/uc/schemas/v1"
	"github.com/databricks/sdk-go/uc/volumes/v1"
	"github.com/stretchr/testify/require"
)

func TemporaryWorkspaceDir(t *WorkspaceT, name ...string) string {
	ctx := t.ctx
	me, err := t.W.Scim.Me(ctx, sdkscim.MeRequest{})
	require.NoError(t, err)

	// Prefix the name with "integration-test-" to make it easier to identify.
	name = append([]string{"integration-test-"}, name...)
	basePath := fmt.Sprintf("/Users/%s/%s", utils.Deref(me.UserName), testutil.RandomName(name...))

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
	api, err := apiclient.New(t.W.Config)
	require.NoError(t, err)
	err = api.Do(ctx, http.MethodPost, "/api/2.0/dbfs/mkdirs", nil, nil, map[string]any{"path": path}, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		t.Logf("Removing DBFS directory %s", path)
		err := api.Do(context.WithoutCancel(ctx), http.MethodPost, "/api/2.0/dbfs/delete", nil, nil, map[string]any{"path": path, "recursive": true}, nil)
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
	path := fmt.Sprintf("/Repos/%s/%s", utils.Deref(me.UserName), testutil.RandomName("integration-test-"))

	t.Logf("Creating repo: %s", path)
	resp, err := t.W.Repos.CreateRepo(ctx, repos.CreateRepoRequest{
		Url:      new(url),
		Path:     new(path),
		Provider: new("gitHub"),
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		t.Logf("Removing repo: %s", path)
		_, err := t.W.Repos.DeleteRepo(context.WithoutCancel(ctx), repos.DeleteRepoRequest{
			Id: resp.Id,
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
	schema, err := w.Schemas.CreateSchema(ctx, schemas.CreateSchemaRequest{
		CatalogName: new("main"),
		Name:        new(testutil.RandomName("test-schema-")),
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := w.Schemas.DeleteSchema(context.WithoutCancel(ctx), schemas.DeleteSchemaRequest{
			FullNameArg: schema.FullName,
		})
		require.NoError(t, err)
	})

	// Create a volume
	volume, err := w.Volumes.CreateVolume(ctx, volumes.CreateVolumeRequest{
		CatalogName: new("main"),
		SchemaName:  schema.Name,
		Name:        new("my-volume"),
		VolumeType:  volumes.VolumeType_Managed,
	})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := w.Volumes.DeleteVolume(context.WithoutCancel(ctx), volumes.DeleteVolumeRequest{
			FullNameArg: volume.FullName,
		})
		require.NoError(t, err)
	})

	return fmt.Sprintf("/Volumes/%s/%s/%s", "main", utils.Deref(schema.Name), utils.Deref(volume.Name))
}
