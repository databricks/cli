package internal

import (
	"context"
	"fmt"
	"strings"
	"testing"

	_ "github.com/databricks/cli/cmd/fs"
	"github.com/databricks/cli/libs/filer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCompletionFile(t *testing.T, f filer.Filer) {
	err := f.Write(context.Background(), "dir1/file1.txt", strings.NewReader("abc"), filer.CreateParentDirectories)
	require.NoError(t, err)
}

func TestAccFsCompletion(t *testing.T) {
	f, tmpDir := setupDbfsFiler(t)
	setupCompletionFile(t, f)

	stdout, _ := RequireSuccessfulRun(t, "__complete", "fs", "ls", tmpDir+"/")
	expectedOutput := fmt.Sprintf("%s/dir1/\n:2\n", tmpDir)
	assert.Equal(t, expectedOutput, stdout.String())
}
