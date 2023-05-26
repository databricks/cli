package helpers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/databricks/cli/internal"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/require"
)

type workspaceTestdata struct {
	root   string
	t      *testing.T
	client *databricks.WorkspaceClient
}

func NewWorkspaceTestdata(t *testing.T) *workspaceTestdata {
	ctx := context.Background()
	w := databricks.Must(databricks.NewWorkspaceClient())

	me, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)
	path := fmt.Sprintf("/Users/%s/%s", me.UserName, internal.RandomName("wsfs-files-"))

	// Ensure directory exists, but doesn't exist YET!
	// Otherwise we could inadvertently remove a directory that already exists on cleanup.
	t.Logf("mkdir %s", path)
	err = w.Workspace.MkdirsByPath(ctx, path)
	require.NoError(t, err)

	// Remove test directory on test completion.
	t.Cleanup(func() {
		t.Logf("rm -rf %s", path)
		err := w.Workspace.Delete(ctx, workspace.Delete{
			Path:      path,
			Recursive: true,
		})
		if err == nil || apierr.IsMissing(err) {
			return
		}
		t.Logf("unable to remove temporary workspace path %s: %#v", path, err)
	})

	return &workspaceTestdata{
		root:   path,
		t:      t,
		client: w,
	}
}

func (w *workspaceTestdata) RootPath() string {
	return w.root
}

func (w *workspaceTestdata) AddFile(name string, content string) {
	path := path.Join(w.root, name)
	ctx := context.Background()

	// url path for uploading file API
	urlPath := fmt.Sprintf(
		"/api/2.0/workspace-files/import-file/%s?overwrite=true",
		url.PathEscape(strings.TrimLeft(path, "/")),
	)

	// initialize API client
	apiClient, err := client.New(w.client.Config)
	require.NoError(w.t, err)

	// Make API request
	err = apiClient.Do(ctx, http.MethodPost, urlPath, content, nil)
	require.NoError(w.t, err)
}

func (w *workspaceTestdata) Mkdir(name string) {
	ctx := context.Background()
	path := path.Join(w.root, name)
	err := w.client.Workspace.MkdirsByPath(ctx, path)
	require.NoError(w.t, err)
}
