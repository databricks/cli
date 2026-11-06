package testserver_test

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"testing"

	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/service/sandbox"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const sandboxParent = "sandboxes/" + testserver.TestDefaultSandboxId

func newSandboxAPI(t *testing.T) sandbox.SandboxInterface {
	t.Helper()
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	w, err := databricks.NewWorkspaceClient(&databricks.Config{
		Host:               server.URL,
		Token:              "testtoken",
		RateLimitPerSecond: 10000,
	})
	require.NoError(t, err)
	return w.Sandbox
}

func upload(t *testing.T, api sandbox.SandboxInterface, path, contents string) error {
	t.Helper()
	return api.UploadFile(t.Context(), sandbox.UploadFileRequest{
		Parent:   sandboxParent,
		Path:     path,
		Contents: io.NopCloser(strings.NewReader(contents)),
	})
}

func TestSandboxFilesListPagination(t *testing.T) {
	api := newSandboxAPI(t)
	for _, name := range []string{"c", "a", "e", "b", "d"} {
		require.NoError(t, upload(t, api, "/tmp/dir/"+name, ""))
	}

	resp, err := api.ListDirectoryContents(t.Context(), sandbox.ListDirectoryContentsRequest{
		Parent:   sandboxParent,
		Path:     "/tmp/dir",
		PageSize: 2,
	}).Next(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "a", resp.Name)

	all, err := api.ListDirectoryContentsAll(t.Context(), sandbox.ListDirectoryContentsRequest{
		Parent:   sandboxParent,
		Path:     "/tmp/dir",
		PageSize: 2,
	})
	require.NoError(t, err)
	var names []string
	for _, info := range all {
		names = append(names, info.Name)
	}
	assert.Equal(t, []string{"a", "b", "c", "d", "e"}, names)
}

func TestSandboxFilesUploadRejectsUnknownParameters(t *testing.T) {
	// The real server rejects "overwrite", like any parameter besides "path".
	server := testserver.New(t)
	testserver.AddDefaultHandlers(server)
	c, err := client.New(&config.Config{Host: server.URL, Token: "testtoken"})
	require.NoError(t, err)

	err = c.Do(t.Context(), http.MethodPut, "/api/2.0/sandbox-files/"+sandboxParent+"/content",
		map[string]string{"Content-Type": "application/octet-stream"},
		map[string]any{"path": "/tmp/file", "overwrite": false},
		io.NopCloser(strings.NewReader("")), nil)
	assert.ErrorIs(t, err, apierr.ErrInvalidParameterValue)
}

func TestSandboxFilesUploadAfterDeletingRoot(t *testing.T) {
	api := newSandboxAPI(t)
	_, err := api.DeletePath(t.Context(), sandbox.DeletePathRequest{Parent: sandboxParent, Path: "/", Recursive: true})
	require.NoError(t, err)
	require.NoError(t, upload(t, api, "/tmp/file", ""))
}

func TestSandboxFilesRenameIsExclusive(t *testing.T) {
	api := newSandboxAPI(t)
	const n = 8
	for i := range n {
		require.NoError(t, upload(t, api, fmt.Sprintf("/tmp/src%d", i), fmt.Sprintf("writer-%d", i)))
	}

	var wg sync.WaitGroup
	errs := make([]error, n)
	for i := range n {
		wg.Go(func() {
			_, errs[i] = api.RenamePath(t.Context(), sandbox.RenamePathRequest{
				Parent:      sandboxParent,
				Source:      fmt.Sprintf("/tmp/src%d", i),
				Destination: "/tmp/dst",
			})
		})
	}
	wg.Wait()

	wins := 0
	for _, err := range errs {
		if err == nil {
			wins++
			continue
		}
		assert.ErrorIs(t, err, apierr.ErrAlreadyExists)
	}
	assert.Equal(t, 1, wins)
}
