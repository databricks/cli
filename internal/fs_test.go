package internal

import (
	"context"
	"path"
	"testing"

	"github.com/databricks/cli/cmd/fs"
	"github.com/databricks/databricks-sdk-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilerForPathForDbfsPaths(t *testing.T) {
	t.Log(GetEnvOrSkipTest(t, "CLOUD_ENV"))

	w, err := databricks.NewWorkspaceClient()
	require.NoError(t, err)

	ctx := context.Background()
	tmpDir := temporaryDbfsDir(t, w)

	f, path, err := fs.FilerForPath(ctx, path.Join("dbfs:", tmpDir))
	assert.NoError(t, err)

	// assert dbfs scheme is trimmed from input path
	assert.Equal(t, tmpDir, path)

	// assert filer works
	stat, err := f.Stat(ctx, tmpDir)
	require.NoError(t, err)
	assert.True(t, stat.IsDir())
}
