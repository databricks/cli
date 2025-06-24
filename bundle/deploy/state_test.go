package deploy

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/require"
)

func TestFromSlice(t *testing.T) {
	tmpDir := t.TempDir()
	fileset := fileset.New(vfs.MustNew(tmpDir))
	testutil.Touch(t, tmpDir, "test1.py")
	testutil.Touch(t, tmpDir, "test2.py")
	testutil.Touch(t, tmpDir, "test3.py")

	files, err := fileset.Files()
	require.NoError(t, err)

	f, err := fromSlice(files)
	require.NoError(t, err)
	require.Len(t, f, 3)

	for _, file := range f {
		require.Contains(t, []string{"test1.py", "test2.py", "test3.py"}, file.LocalPath)
	}
}

func TestToSlice(t *testing.T) {
	tmpDir := t.TempDir()
	root := vfs.MustNew(tmpDir)
	fileset := fileset.New(root)
	testutil.Touch(t, tmpDir, "test1.py")
	testutil.Touch(t, tmpDir, "test2.py")
	testutil.Touch(t, tmpDir, "test3.py")

	files, err := fileset.Files()
	require.NoError(t, err)

	f, err := fromSlice(files)
	require.NoError(t, err)
	require.Len(t, f, 3)

	s := f.toSlice(root)
	require.Len(t, s, 3)

	for _, file := range s {
		require.Contains(t, []string{"test1.py", "test2.py", "test3.py"}, file.Relative)

		// If the mtime is not zero we know we produced a valid fs.DirEntry.
		ts := file.Modified()
		require.NotZero(t, ts)
	}
}

func TestIsLocalStateStale(t *testing.T) {
	oldState, err := json.Marshal(DeploymentState{
		Seq: 1,
	})
	require.NoError(t, err)

	newState, err := json.Marshal(DeploymentState{
		Seq: 2,
	})
	require.NoError(t, err)

	require.True(t, isLocalStateStale(bytes.NewReader(oldState), bytes.NewReader(newState)))
	require.False(t, isLocalStateStale(bytes.NewReader(newState), bytes.NewReader(oldState)))
}
