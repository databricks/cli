package internal

import (
	"context"
	"path"
	"regexp"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccFsMkdirCreatesDirectory(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := temporaryDbfsDir(t, w)

	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	// create directory "a"
	stdout, stderr := RequireSuccessfulRun(t, "fs", "mkdir", "dbfs:"+path.Join(tmpDir, "a"))
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "", stdout.String())

	// assert directory "a" is created
	info, err := f.Stat(ctx, "a")
	require.NoError(t, err)
	assert.Equal(t, "a", info.Name())
	assert.Equal(t, true, info.IsDir())
}

func TestAccFsMkdirCreatesMultipleDirectories(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := temporaryDbfsDir(t, w)

	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	// create directory /a/b/c
	stdout, stderr := RequireSuccessfulRun(t, "fs", "mkdir", "dbfs:"+path.Join(tmpDir, "a", "b", "c"))
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "", stdout.String())

	// assert directory "a" is created
	infoA, err := f.Stat(ctx, "a")
	require.NoError(t, err)
	assert.Equal(t, "a", infoA.Name())
	assert.Equal(t, true, infoA.IsDir())

	// assert directory "b" is created
	infoB, err := f.Stat(ctx, "a/b")
	require.NoError(t, err)
	assert.Equal(t, "b", infoB.Name())
	assert.Equal(t, true, infoB.IsDir())

	// assert directory "c" is created
	infoC, err := f.Stat(ctx, "a/b/c")
	require.NoError(t, err)
	assert.Equal(t, "c", infoC.Name())
	assert.Equal(t, true, infoC.IsDir())
}

func TestAccFsMkdirWhenDirectoryAlreadyExists(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := temporaryDbfsDir(t, w)

	// create directory "a"
	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)
	err = f.Mkdir(ctx, "a")
	require.NoError(t, err)

	// assert run is successful without any errors
	stdout, stderr := RequireSuccessfulRun(t, "fs", "mkdir", "dbfs:"+path.Join(tmpDir, "a"))
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "", stdout.String())
}

func TestAccFsMkdirWhenFileExistsAtPath(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := temporaryDbfsDir(t, w)

	// create file hello
	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)
	err = f.Write(ctx, "hello", strings.NewReader("abc"))
	require.NoError(t, err)

	// assert run fails
	_, _, err = RequireErrorRun(t, "fs", "mkdir", "dbfs:"+path.Join(tmpDir, "hello"))
	// Different backends return different errors (for example: file in s3 vs dbfs)
	regex := regexp.MustCompile(`^Path is a file: .*$|^Cannot create directory .* because .* is an existing file`)
	assert.Regexp(t, regex, err.Error())
}
