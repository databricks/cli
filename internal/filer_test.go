package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/dbfs"
	"github.com/databricks/databricks-sdk-go/service/workspace"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type filerTest struct {
	*testing.T
	filer.Filer
}

func (f filerTest) assertContents(ctx context.Context, name string, contents string) {
	reader, err := f.Read(ctx, name)
	if !assert.NoError(f, err) {
		return
	}

	var body bytes.Buffer
	_, err = io.Copy(&body, reader)
	if !assert.NoError(f, err) {
		return
	}

	assert.Equal(f, contents, body.String())
}

func temporaryWorkspaceDir(t *testing.T, w *databricks.WorkspaceClient) string {
	ctx := context.Background()
	me, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	path := fmt.Sprintf("/Users/%s/%s", me.UserName, RandomName("integration-test-filer-wsfs-"))

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
		t.Logf("unable to remove temporary workspace directory %s: %#v", path, err)
	})

	return path
}

func TestAccFilerWorkspaceFiles(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w := databricks.Must(databricks.NewWorkspaceClient())
	tmpdir := temporaryWorkspaceDir(t, w)
	f, err := filer.NewWorkspaceFilesClient(w, tmpdir)
	require.NoError(t, err)

	// Check if we can use this API here, skip test if we cannot.
	_, err = f.Read(ctx, "we_use_this_call_to_test_if_this_api_is_enabled")
	var aerr *apierr.APIError
	if errors.As(err, &aerr) && aerr.StatusCode == http.StatusBadRequest {
		t.Skip(aerr.Message)
	}

	// Write should fail because the root path doesn't yet exist.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`hello world`))
	assert.True(t, errors.As(err, &filer.NoSuchDirectoryError{}))

	// Read should fail because the root path doesn't yet exist.
	_, err = f.Read(ctx, "/foo/bar")
	assert.True(t, errors.As(err, &filer.FileDoesNotExistError{}))

	// Write with CreateParentDirectories flag should succeed.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`hello world`), filer.CreateParentDirectories)
	assert.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "/foo/bar", `hello world`)

	// Write should fail because there is an existing file at the specified path.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`hello universe`))
	assert.True(t, errors.As(err, &filer.FileAlreadyExistsError{}))

	// Write with OverwriteIfExists should succeed.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`hello universe`), filer.OverwriteIfExists)
	assert.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "/foo/bar", `hello universe`)

	// Delete should fail if the file doesn't exist.
	err = f.Delete(ctx, "/doesnt_exist")
	assert.True(t, errors.As(err, &filer.FileDoesNotExistError{}))

	// Delete should succeed for file that does exist.
	err = f.Delete(ctx, "/foo/bar")
	assert.NoError(t, err)
}

func temporaryDbfsDir(t *testing.T, w *databricks.WorkspaceClient) string {
	ctx := context.Background()
	path := fmt.Sprintf("/tmp/%s", RandomName("integration-test-filer-dbfs-"))

	// This call fails if the path already exists.
	t.Logf("mkdir dbfs:%s", path)
	err := w.Dbfs.MkdirsByPath(ctx, path)
	require.NoError(t, err)

	// Remove test directory on test completion.
	t.Cleanup(func() {
		t.Logf("rm -rf dbfs:%s", path)
		err := w.Dbfs.Delete(ctx, dbfs.Delete{
			Path:      path,
			Recursive: true,
		})
		if err == nil || apierr.IsMissing(err) {
			return
		}
		t.Logf("unable to remove temporary dbfs directory %s: %#v", path, err)
	})

	return path
}

func TestAccFilerDbfs(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w := databricks.Must(databricks.NewWorkspaceClient())
	tmpdir := temporaryDbfsDir(t, w)
	f, err := filer.NewDbfsClient(w, tmpdir)
	require.NoError(t, err)

	// Read should fail because the root path doesn't yet exist.
	_, err = f.Read(ctx, "/foo/bar")
	assert.True(t, errors.As(err, &filer.FileDoesNotExistError{}))

	// Write with CreateParentDirectories flag should succeed.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`"hello world"`), filer.CreateParentDirectories)
	assert.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "/foo/bar", `"hello world"`)

	// Write should fail because there is an existing file at the specified path.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`"hello universe"`))
	assert.True(t, errors.As(err, &filer.FileAlreadyExistsError{}))

	// Write with OverwriteIfExists should succeed.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`"hello universe"`), filer.OverwriteIfExists)
	assert.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "/foo/bar", `"hello universe"`)

	// Delete should fail if the file doesn't exist.
	err = f.Delete(ctx, "/doesnt_exist")
	assert.True(t, errors.As(err, &filer.FileDoesNotExistError{}))

	// Delete should succeed for file that does exist.
	err = f.Delete(ctx, "/foo/bar")
	assert.NoError(t, err)
}
