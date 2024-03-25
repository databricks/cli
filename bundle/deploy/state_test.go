package deploy

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/fileset"
	"github.com/stretchr/testify/require"
)

func TestFromSlice(t *testing.T) {
	tmpDir := t.TempDir()
	fileset := fileset.New(tmpDir)
	testutil.Touch(t, tmpDir, "test1.py")
	testutil.Touch(t, tmpDir, "test2.py")
	testutil.Touch(t, tmpDir, "test3.py")

	files, err := fileset.All()
	require.NoError(t, err)

	f, err := FromSlice(files)
	require.NoError(t, err)
	require.Len(t, f, 3)

	for _, file := range f {
		require.Contains(t, []string{"test1.py", "test2.py", "test3.py"}, file.LocalPath)
	}
}

func TestToSlice(t *testing.T) {
	tmpDir := t.TempDir()
	fileset := fileset.New(tmpDir)
	testutil.Touch(t, tmpDir, "test1.py")
	testutil.Touch(t, tmpDir, "test2.py")
	testutil.Touch(t, tmpDir, "test3.py")

	files, err := fileset.All()
	require.NoError(t, err)

	f, err := FromSlice(files)
	require.NoError(t, err)
	require.Len(t, f, 3)

	s := f.ToSlice(tmpDir)
	require.Len(t, s, 3)

	for _, file := range s {
		require.Contains(t, []string{"test1.py", "test2.py", "test3.py"}, file.Name())
		require.Contains(t, []string{
			filepath.Join(tmpDir, "test1.py"),
			filepath.Join(tmpDir, "test2.py"),
			filepath.Join(tmpDir, "test3.py"),
		}, file.Absolute)
		require.False(t, file.IsDir())
		require.NotZero(t, file.Type())
		info, err := file.Info()
		require.NoError(t, err)
		require.NotNil(t, info)
		require.Equal(t, file.Name(), info.Name())
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
