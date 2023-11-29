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

func TestAccFsCatForDbfs(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := TemporaryDbfsDir(t, w)

	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	err = f.Write(ctx, "a/hello.txt", strings.NewReader("abc"), -1, filer.CreateParentDirectories)
	require.NoError(t, err)

	stdout, stderr := RequireSuccessfulRun(t, "fs", "cat", "dbfs:"+path.Join(tmpDir, "a", "hello.txt"))
	assert.Equal(t, "", stderr.String())
	assert.Equal(t, "abc", stdout.String())
}

func TestAccFsCatForDbfsOnNonExistentFile(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	_, _, err := RequireErrorRun(t, "fs", "cat", "dbfs:/non-existent-file")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAccFsCatForDbfsInvalidScheme(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	_, _, err := RequireErrorRun(t, "fs", "cat", "dab:/non-existent-file")
	assert.ErrorContains(t, err, "invalid scheme: dab")
}

func TestAccFsCatDoesNotSupportOutputModeJson(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := TemporaryDbfsDir(t, w)

	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	err = f.Write(ctx, "hello.txt", strings.NewReader("abc"), -1)
	require.NoError(t, err)

	_, _, err = RequireErrorRun(t, "fs", "cat", "dbfs:"+path.Join(tmpDir, "hello.txt"), "--output=json")
	assert.ErrorContains(t, err, "json output not supported")
}

// TODO: Add test asserting an error when cat is called on an directory. Need this to be
// fixed in the SDK first (https://github.com/databricks/databricks-sdk-go/issues/414)
