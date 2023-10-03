package internal

import (
	"context"
	"encoding/json"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"testing"

	_ "github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccFsLsForDbfs(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := TemporaryDbfsDir(t, w)

	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	err = f.Mkdir(ctx, "a")
	require.NoError(t, err)
	err = f.Write(ctx, "a/hello.txt", strings.NewReader("abc"), filer.CreateParentDirectories)
	require.NoError(t, err)
	err = f.Write(ctx, "bye.txt", strings.NewReader("def"))
	require.NoError(t, err)

	stdout, stderr := RequireSuccessfulRun(t, "fs", "ls", "dbfs:"+tmpDir, "--output=json")
	assert.Equal(t, "", stderr.String())
	var parsedStdout []map[string]any
	err = json.Unmarshal(stdout.Bytes(), &parsedStdout)
	require.NoError(t, err)

	// assert on ls output
	assert.Len(t, parsedStdout, 2)
	assert.Equal(t, "a", parsedStdout[0]["name"])
	assert.Equal(t, true, parsedStdout[0]["is_directory"])
	assert.Equal(t, float64(0), parsedStdout[0]["size"])
	assert.Equal(t, "bye.txt", parsedStdout[1]["name"])
	assert.Equal(t, false, parsedStdout[1]["is_directory"])
	assert.Equal(t, float64(3), parsedStdout[1]["size"])
}

func TestAccFsLsForDbfsWithAbsolutePaths(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := TemporaryDbfsDir(t, w)

	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	err = f.Mkdir(ctx, "a")
	require.NoError(t, err)
	err = f.Write(ctx, "a/hello.txt", strings.NewReader("abc"), filer.CreateParentDirectories)
	require.NoError(t, err)
	err = f.Write(ctx, "bye.txt", strings.NewReader("def"))
	require.NoError(t, err)

	stdout, stderr := RequireSuccessfulRun(t, "fs", "ls", "dbfs:"+tmpDir, "--output=json", "--absolute")
	assert.Equal(t, "", stderr.String())
	var parsedStdout []map[string]any
	err = json.Unmarshal(stdout.Bytes(), &parsedStdout)
	require.NoError(t, err)

	// assert on ls output
	assert.Len(t, parsedStdout, 2)
	assert.Equal(t, path.Join("dbfs:", tmpDir, "a"), parsedStdout[0]["name"])
	assert.Equal(t, true, parsedStdout[0]["is_directory"])
	assert.Equal(t, float64(0), parsedStdout[0]["size"])

	assert.Equal(t, path.Join("dbfs:", tmpDir, "bye.txt"), parsedStdout[1]["name"])
	assert.Equal(t, false, parsedStdout[1]["is_directory"])
	assert.Equal(t, float64(3), parsedStdout[1]["size"])
}

func TestAccFsLsForDbfsOnFile(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := TemporaryDbfsDir(t, w)

	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	err = f.Mkdir(ctx, "a")
	require.NoError(t, err)
	err = f.Write(ctx, "a/hello.txt", strings.NewReader("abc"), filer.CreateParentDirectories)
	require.NoError(t, err)

	_, _, err = RequireErrorRun(t, "fs", "ls", "dbfs:"+path.Join(tmpDir, "a", "hello.txt"), "--output=json")
	assert.Regexp(t, regexp.MustCompile("not a directory: .*/a/hello.txt"), err.Error())
}

func TestAccFsLsForDbfsOnEmptyDir(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := TemporaryDbfsDir(t, w)

	stdout, stderr := RequireSuccessfulRun(t, "fs", "ls", "dbfs:"+tmpDir, "--output=json")
	assert.Equal(t, "", stderr.String())
	var parsedStdout []map[string]any
	err = json.Unmarshal(stdout.Bytes(), &parsedStdout)
	require.NoError(t, err)

	// assert on ls output
	assert.Equal(t, 0, len(parsedStdout))
}

func TestAccFsLsForDbfsForNonexistingDir(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	_, _, err := RequireErrorRun(t, "fs", "ls", "dbfs:/john-cena", "--output=json")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestAccFsLsWithoutScheme(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	_, _, err := RequireErrorRun(t, "fs", "ls", "/ray-mysterio", "--output=json")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}
