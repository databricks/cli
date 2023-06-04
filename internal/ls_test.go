package internal

import (
	"context"
	"encoding/json"
	"io/fs"
	"strings"
	"testing"

	_ "github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFsLsForDbfs(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := temporaryDbfsDir(t, w)

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
	assert.Equal(t, "a", parsedStdout[0]["name"])
	assert.Equal(t, true, parsedStdout[0]["is_directory"])
	assert.Equal(t, float64(0), parsedStdout[0]["size"])
	assert.Equal(t, "bye.txt", parsedStdout[1]["name"])
	assert.Equal(t, false, parsedStdout[1]["is_directory"])
	assert.Equal(t, float64(3), parsedStdout[1]["size"])
}

func TestFsLsForDbfsOnEmptyDir(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := temporaryDbfsDir(t, w)

	stdout, stderr := RequireSuccessfulRun(t, "fs", "ls", "dbfs:"+tmpDir, "--output=json")
	assert.Equal(t, "", stderr.String())
	var parsedStdout []map[string]any
	err = json.Unmarshal(stdout.Bytes(), &parsedStdout)
	require.NoError(t, err)

	// assert on ls output
	assert.Equal(t, 0, len(parsedStdout))
}

func TestFsLsForDbfsForNonexistingDir(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	_, _, err := RequireErrorRun(t, "fs", "ls", "dbfs:/john-cena", "--output=json")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestFsLsWithoutScheme(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	_, _, err := RequireErrorRun(t, "fs", "ls", "/ray-mysterio", "--output=json")
	assert.ErrorContains(t, err, "expected dbfs path (with the dbfs:/ prefix): /ray-mysterio")
}
