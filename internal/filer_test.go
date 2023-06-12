package internal

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/service/files"
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

func runFilerReadWriteTest(t *testing.T, ctx context.Context, f filer.Filer) {
	var err error

	// Write should fail because the root path doesn't yet exist.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`hello world`))
	assert.True(t, errors.As(err, &filer.NoSuchDirectoryError{}))
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Read should fail because the root path doesn't yet exist.
	_, err = f.Read(ctx, "/foo/bar")
	assert.True(t, errors.As(err, &filer.FileDoesNotExistError{}))
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Read should fail because the path points to a directory
	err = f.Mkdir(ctx, "/dir")
	require.NoError(t, err)
	_, err = f.Read(ctx, "/dir")
	assert.ErrorIs(t, err, fs.ErrInvalid)

	// Write with CreateParentDirectories flag should succeed.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`hello world`), filer.CreateParentDirectories)
	assert.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "/foo/bar", `hello world`)

	// Write should fail because there is an existing file at the specified path.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`hello universe`))
	assert.True(t, errors.As(err, &filer.FileAlreadyExistsError{}))
	assert.True(t, errors.Is(err, fs.ErrExist))

	// Write with OverwriteIfExists should succeed.
	err = f.Write(ctx, "/foo/bar", strings.NewReader(`hello universe`), filer.OverwriteIfExists)
	assert.NoError(t, err)
	filerTest{t, f}.assertContents(ctx, "/foo/bar", `hello universe`)

	// Write should succeed if there is no existing file at the specified path.
	err = f.Write(ctx, "/foo/qux", strings.NewReader(`hello universe`))
	assert.NoError(t, err)

	// Stat on a directory should succeed.
	// Note: size and modification time behave differently between WSFS and DBFS.
	info, err := f.Stat(ctx, "/foo")
	require.NoError(t, err)
	assert.Equal(t, "foo", info.Name())
	assert.True(t, info.Mode().IsDir())
	assert.Equal(t, true, info.IsDir())

	// Stat on a file should succeed.
	// Note: size and modification time behave differently between WSFS and DBFS.
	info, err = f.Stat(ctx, "/foo/bar")
	require.NoError(t, err)
	assert.Equal(t, "bar", info.Name())
	assert.True(t, info.Mode().IsRegular())
	assert.Equal(t, false, info.IsDir())

	// Delete should fail if the file doesn't exist.
	err = f.Delete(ctx, "/doesnt_exist")
	assert.True(t, errors.As(err, &filer.FileDoesNotExistError{}))
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Stat should fail if the file doesn't exist.
	_, err = f.Stat(ctx, "/doesnt_exist")
	assert.True(t, errors.As(err, &filer.FileDoesNotExistError{}))
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Delete should succeed for file that does exist.
	err = f.Delete(ctx, "/foo/bar")
	assert.NoError(t, err)

	// Delete should fail for a non-empty directory.
	err = f.Delete(ctx, "/foo")
	assert.True(t, errors.As(err, &filer.DirectoryNotEmptyError{}))
	assert.True(t, errors.Is(err, fs.ErrInvalid))

	// Delete should succeed for a non-empty directory if the DeleteRecursively flag is set.
	err = f.Delete(ctx, "/foo", filer.DeleteRecursively)
	assert.NoError(t, err)

	// Delete of the filer root should ALWAYS fail, otherwise subsequent writes would fail.
	// It is not in the filer's purview to delete its root directory.
	err = f.Delete(ctx, "/")
	assert.True(t, errors.As(err, &filer.CannotDeleteRootError{}))
	assert.True(t, errors.Is(err, fs.ErrInvalid))
}

func runFilerReadDirTest(t *testing.T, ctx context.Context, f filer.Filer) {
	var err error
	var info fs.FileInfo

	// We start with an empty directory.
	entries, err := f.ReadDir(ctx, ".")
	require.NoError(t, err)
	assert.Len(t, entries, 0)

	// Write a file.
	err = f.Write(ctx, "/hello.txt", strings.NewReader(`hello world`))
	require.NoError(t, err)

	// Create a directory.
	err = f.Mkdir(ctx, "/dir")
	require.NoError(t, err)

	// Write a file.
	err = f.Write(ctx, "/dir/world.txt", strings.NewReader(`hello world`))
	require.NoError(t, err)

	// Create a nested directory (check that it creates intermediate directories).
	err = f.Mkdir(ctx, "/dir/a/b/c")
	require.NoError(t, err)

	// Expect an error if the path doesn't exist.
	_, err = f.ReadDir(ctx, "/dir/a/b/c/d/e")
	assert.True(t, errors.As(err, &filer.NoSuchDirectoryError{}), err)
	assert.True(t, errors.Is(err, fs.ErrNotExist))

	// Expect two entries in the root.
	entries, err = f.ReadDir(ctx, ".")
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "dir", entries[0].Name())
	assert.True(t, entries[0].IsDir())
	assert.Equal(t, "hello.txt", entries[1].Name())
	assert.False(t, entries[1].IsDir())
	info, err = entries[1].Info()
	require.NoError(t, err)
	assert.Greater(t, info.ModTime().Unix(), int64(0))

	// Expect two entries in the directory.
	entries, err = f.ReadDir(ctx, "/dir")
	require.NoError(t, err)
	assert.Len(t, entries, 2)
	assert.Equal(t, "a", entries[0].Name())
	assert.True(t, entries[0].IsDir())
	assert.Equal(t, "world.txt", entries[1].Name())
	assert.False(t, entries[1].IsDir())
	info, err = entries[1].Info()
	require.NoError(t, err)
	assert.Greater(t, info.ModTime().Unix(), int64(0))

	// Expect a single entry in the nested path.
	entries, err = f.ReadDir(ctx, "/dir/a/b")
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "c", entries[0].Name())
	assert.True(t, entries[0].IsDir())

	// Expect an error trying to call ReadDir on a file
	_, err = f.ReadDir(ctx, "/hello.txt")
	assert.ErrorIs(t, err, fs.ErrInvalid)

	// Expect 0 entries for an empty directory
	err = f.Mkdir(ctx, "empty-dir")
	require.NoError(t, err)
	entries, err = f.ReadDir(ctx, "empty-dir")
	assert.NoError(t, err)
	assert.Len(t, entries, 0)

	// Expect one entry for a directory with a file in it
	err = f.Write(ctx, "dir-with-one-file/my-file.txt", strings.NewReader("abc"), filer.CreateParentDirectories)
	require.NoError(t, err)
	entries, err = f.ReadDir(ctx, "dir-with-one-file")
	assert.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, entries[0].Name(), "my-file.txt")
	assert.False(t, entries[0].IsDir())
}

func temporaryWorkspaceDir(t *testing.T, w *databricks.WorkspaceClient) string {
	ctx := context.Background()
	me, err := w.CurrentUser.Me(ctx)
	require.NoError(t, err)

	path := fmt.Sprintf("/Users/%s/%s", me.UserName, RandomName("integration-test-wsfs-"))

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

func setupWorkspaceFilesTest(t *testing.T) (context.Context, filer.Filer) {
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

	return ctx, f
}

func TestAccFilerWorkspaceFilesReadWrite(t *testing.T) {
	ctx, f := setupWorkspaceFilesTest(t)
	runFilerReadWriteTest(t, ctx, f)
}

func TestAccFilerWorkspaceFilesReadDir(t *testing.T) {
	ctx, f := setupWorkspaceFilesTest(t)
	runFilerReadDirTest(t, ctx, f)
}

func temporaryDbfsDir(t *testing.T, w *databricks.WorkspaceClient) string {
	ctx := context.Background()
	path := fmt.Sprintf("/tmp/%s", RandomName("integration-test-dbfs-"))

	// This call fails if the path already exists.
	t.Logf("mkdir dbfs:%s", path)
	err := w.Dbfs.MkdirsByPath(ctx, path)
	require.NoError(t, err)

	// Remove test directory on test completion.
	t.Cleanup(func() {
		t.Logf("rm -rf dbfs:%s", path)
		err := w.Dbfs.Delete(ctx, files.Delete{
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

func setupFilerDbfsTest(t *testing.T) (context.Context, filer.Filer) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w := databricks.Must(databricks.NewWorkspaceClient())
	tmpdir := temporaryDbfsDir(t, w)
	f, err := filer.NewDbfsClient(w, tmpdir)
	require.NoError(t, err)
	return ctx, f
}

func TestAccFilerDbfsReadWrite(t *testing.T) {
	ctx, f := setupFilerDbfsTest(t)
	runFilerReadWriteTest(t, ctx, f)
}

func TestAccFilerDbfsReadDir(t *testing.T) {
	ctx, f := setupFilerDbfsTest(t)
	runFilerReadDirTest(t, ctx, f)
}

func setupFilerLocalTest(t *testing.T) (context.Context, filer.Filer) {
	ctx := context.Background()
	f, err := filer.NewLocalClient(t.TempDir())
	require.NoError(t, err)
	return ctx, f
}

func TestAccFilerLocalReadWrite(t *testing.T) {
	ctx, f := setupFilerLocalTest(t)
	runFilerReadWriteTest(t, ctx, f)
}

func TestAccFilerLocalReadDir(t *testing.T) {
	ctx, f := setupFilerLocalTest(t)
	runFilerReadDirTest(t, ctx, f)
}
