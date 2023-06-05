package internal

import (
	"context"
	"encoding/json"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFsCatForDbfs(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	ctx := context.Background()
	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	tmpDir := temporaryDbfsDir(t, w)

	f, err := filer.NewDbfsClient(w, tmpDir)
	require.NoError(t, err)

	err = f.Write(ctx, "a/hello.txt", strings.NewReader("abc"), filer.CreateParentDirectories)
	require.NoError(t, err)

	stdout, stderr := RequireSuccessfulRun(t, "fs", "cat", "dbfs:"+path.Join(tmpDir, "a", "hello.txt"), "--output=json")
	assert.Equal(t, "", stderr.String())
	var parsedStdout map[string]any
	err = json.Unmarshal(stdout.Bytes(), &parsedStdout)
	require.NoError(t, err)

	// assert on cat output
	assert.Equal(t, map[string]any{
		"content": "abc",
	}, parsedStdout)
}

func TestFsCatForDbfsOnNonExistantFile(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	_, _, err := RequireErrorRun(t, "fs", "cat", "dbfs:/non-existant-file", "--output=json")
	assert.ErrorIs(t, err, fs.ErrNotExist)
}

func TestFsCatForDbfsInvalidScheme(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	_, _, err := RequireErrorRun(t, "fs", "cat", "dab:/non-existant-file", "--output=json")
	assert.ErrorContains(t, err, "expected dbfs path (with the dbfs:/ prefix): dab:/non-existant-file")
}

// TODO: Add test asserting an error when cat is called on an directory. Need this to be
// fixed in the SDK first (https://github.com/databricks/databricks-sdk-go/issues/414)
