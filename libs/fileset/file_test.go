package fileset

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/require"
)

func TestNotebookFileIsNotebook(t *testing.T) {
	f := NewNotebookFile(nil, nil, "")
	isNotebook, err := f.IsNotebook()
	require.NoError(t, err)
	require.True(t, isNotebook)
}

func TestSourceFileIsNotNotebook(t *testing.T) {
	f := NewSourceFile(nil, nil, "")
	isNotebook, err := f.IsNotebook()
	require.NoError(t, err)
	require.False(t, isNotebook)
}

func TestFileSize(t *testing.T) {
	tmpDir := t.TempDir()
	contents := []byte("hello world")
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.txt"), contents, 0o644))

	files, err := New(vfs.MustNew(tmpDir)).Files()
	require.NoError(t, err)
	require.Len(t, files, 1)
	size, ok := files[0].Size()
	require.True(t, ok)
	require.Equal(t, int64(len(contents)), size)
}

func TestUnknownFileDetectsNotebook(t *testing.T) {
	tmpDir := t.TempDir()
	root := vfs.MustNew(tmpDir)

	t.Run("file", func(t *testing.T) {
		testutil.Touch(t, tmpDir, "test.py")
		f := NewFile(root, nil, "test.py")
		isNotebook, err := f.IsNotebook()
		require.NoError(t, err)
		require.False(t, isNotebook)
	})

	t.Run("notebook", func(t *testing.T) {
		testutil.TouchNotebook(t, tmpDir, "notebook.py")
		f := NewFile(root, nil, "notebook.py")
		isNotebook, err := f.IsNotebook()
		require.NoError(t, err)
		require.True(t, isNotebook)
	})
}
