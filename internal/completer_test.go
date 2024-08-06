package internal

import (
	"context"
	"strings"
	"testing"

	_ "github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCompletionFiles(t *testing.T, f filer.Filer) {
	err := f.Write(context.Background(), "a/hello.txt", strings.NewReader("abc"), filer.CreateParentDirectories)
	require.NoError(t, err)
	err = f.Write(context.Background(), "bye.txt", strings.NewReader("def"))
	require.NoError(t, err)
}

func TestAccFsCompletion(t *testing.T) {
	t.Parallel()

	f, tmpDir := setupDbfsFiler(t)
	setupCompletionFiles(t, f)

	stdout, stderr := RequireSuccessfulRun(t, "__complete", "fs", "cat", tmpDir, "--output=json")
	assert.Equal(t, "", stderr.String())

	// var parsedStdout []map[string]any
	// err := json.Unmarshal(stdout.Bytes(), &parsedStdout)
	require.NoError(t, err)

	// assert.Len(t, parsedStdout, 2)
	assert.Equal(t, "a", stdout)
}
