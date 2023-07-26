package internal

import (
	"context"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccFsRmForFile(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := temporaryDbfsDir(t, w)

	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	// create file to delete
	err = f.Write(ctx, "hello.txt", strings.NewReader("abc"))
	require.NoError(t, err)

	// check file was created
	info, err := f.Stat(ctx, "hello.txt")
	require.NoError(t, err)
	require.Equal(t, "hello.txt", info.Name())
	require.Equal(t, info.IsDir(), false)

	// Run rm command
	stdout, stderr := RequireSuccessfulRun(t, "fs", "rm", "dbfs:"+path.Join(tmpDir, "hello.txt"))
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "", stdout.String())

	// assert file was deleted
	_, err = f.Stat(ctx, "hello.txt")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAccFsRmForEmptyDirectory(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := temporaryDbfsDir(t, w)

	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	// create directory to delete
	err = f.Mkdir(ctx, "avacado")
	require.NoError(t, err)

	// check directory was created
	info, err := f.Stat(ctx, "avacado")
	require.NoError(t, err)
	require.Equal(t, "avacado", info.Name())
	require.Equal(t, info.IsDir(), true)

	// Run rm command
	stdout, stderr := RequireSuccessfulRun(t, "fs", "rm", "dbfs:"+path.Join(tmpDir, "avacado"))
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "", stdout.String())

	// assert directory was deleted
	_, err = f.Stat(ctx, "avacado")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAccFsRmForNonEmptyDirectory(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := temporaryDbfsDir(t, w)

	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	// create file in dir
	err = f.Write(ctx, "avacado/guacamole", strings.NewReader("abc"), filer.CreateParentDirectories)
	require.NoError(t, err)

	// check file was created
	info, err := f.Stat(ctx, "avacado/guacamole")
	require.NoError(t, err)
	require.Equal(t, "guacamole", info.Name())
	require.Equal(t, info.IsDir(), false)

	// Run rm command
	_, _, err = RequireErrorRun(t, "fs", "rm", "dbfs:"+path.Join(tmpDir, "avacado"))
	assert.ErrorIs(t, err, fs.ErrInvalid)
	assert.ErrorContains(t, err, "directory not empty")
}

func TestAccFsRmForNonExistentFile(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	// Expect error if file does not exist
	_, _, err := RequireErrorRun(t, "fs", "rm", "dbfs:/does-not-exist")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAccFsRmForNonEmptyDirectoryWithRecursiveFlag(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := temporaryDbfsDir(t, w)

	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	// create file in dir
	err = f.Write(ctx, "avacado/guacamole", strings.NewReader("abc"), filer.CreateParentDirectories)
	require.NoError(t, err)

	// check file was created
	info, err := f.Stat(ctx, "avacado/guacamole")
	require.NoError(t, err)
	require.Equal(t, "guacamole", info.Name())
	require.Equal(t, info.IsDir(), false)

	// Run rm command
	stdout, stderr := RequireSuccessfulRun(t, "fs", "rm", "dbfs:"+path.Join(tmpDir, "avacado"), "--recursive")
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "", stdout.String())

	// assert directory was deleted
	_, err = f.Stat(ctx, "avacado")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}
