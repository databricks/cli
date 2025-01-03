package fs_test

import (
	"context"
	"strings"
	"testing"

	_ "github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/internal/testcli"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCompletionFile(t *testing.T, f filer.Filer) {
	err := f.Write(context.Background(), "dir1/file1.txt", strings.NewReader("abc"), filer.CreateParentDirectories)
	require.NoError(t, err)
}

func TestFsCompletion(t *testing.T) {
	ctx := context.Background()
	f, tmpDir := setupDbfsFiler(t)
	setupCompletionFile(t, f)

	stdout, _ := testcli.RequireSuccessfulRun(t, ctx, "__complete", "fs", "ls", tmpDir+"/")
	expectedOutput := tmpDir + "/dir1/\n:2\n"
	assert.Equal(t, expectedOutput, stdout.String())
}
