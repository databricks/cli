package internal

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"testing"

	"github.com/databricks/bricks/utilities"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type filerTest struct {
	*testing.T
	utilities.Filer
}

func (f filerTest) assertContents(ctx context.Context, name string, contents string) {
	reader, err := f.Read(ctx, name)
	if !assert.NoError(f, err) {
		return
	}

	body, err := io.ReadAll(reader)
	if !assert.NoError(f, err) {
		return
	}

	assert.Equal(f, contents, string(body))
}

func TestAccFilerWorkspaceFiles(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w := databricks.Must(databricks.NewWorkspaceClient())
	me, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	tmpdir := fmt.Sprintf("/Users/%s/%s", me.UserName, RandomName("wsfs-files-"))
	f, err := utilities.NewWorkspaceFilesClient(w, tmpdir)
	require.NoError(t, err)

	// Check if we can use this API here, skip test if we cannot.
	_, err = f.Read(ctx, "/foo")
	if apierr, ok := err.(apierr.APIError); ok && apierr.StatusCode == http.StatusBadRequest {
		t.Skip(apierr.Message)
	}

	// Remove test directory on test completion.
	t.Cleanup(func() {
		err := w.Workspace.Delete(ctx, workspace.Delete{
			Path: tmpdir,
		})
		if err != nil {
			t.Logf("unable to remove %s: %s", tmpdir, err)
		}
	})

	// Write should fail because the root path doesn't yet exist.
	err = f.Write(ctx, "/foo", strings.NewReader(`"hello world"`))
	assert.True(t, errors.As(err, &utilities.NoSuchDirectoryError{}))

	// Read should fail because the root path doesn't yet exist.
	_, err = f.Read(ctx, "/foo")
	assert.True(t, apierr.IsMissing(err))

	// Write with CreateParentDirectories flag should succeed.
	err = f.Write(ctx, "/foo", strings.NewReader(`"hello world"`), utilities.CreateParentDirectories)
	assert.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "/foo", `"hello world"`)

	// Write should fail because there is an existing file at the specified path.
	err = f.Write(ctx, "/foo", strings.NewReader(`"hello universe"`))
	assert.True(t, errors.As(err, &utilities.FileAlreadyExistsError{}))

	// Write with OverwriteIfExists should succeed.
	err = f.Write(ctx, "/foo", strings.NewReader(`"hello universe"`), utilities.OverwriteIfExists)
	assert.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "/foo", `"hello universe"`)

	// Remove file.
	err = w.Workspace.Delete(ctx, workspace.Delete{
		Path: path.Join(tmpdir, "foo"),
	})
	assert.NoError(t, err)
}
