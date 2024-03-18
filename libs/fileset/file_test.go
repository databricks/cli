package fileset

import (
	"path/filepath"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestNotebookFileIsNotebook(t *testing.T) {
	f := NewNotebookFile(nil, "", "")
	isNotebook, err := f.IsNotebook()
	require.NoError(t, err)
	require.True(t, isNotebook)
}

func TestSourceFileIsNotNotebook(t *testing.T) {
	f := NewSourceFile(nil, "", "")
	isNotebook, err := f.IsNotebook()
	require.NoError(t, err)
	require.False(t, isNotebook)
}

func TestUnknownFileDetectsNotebook(t *testing.T) {
	tmpDir := t.TempDir()
	testutil.Touch(t, tmpDir, "test.py")
	testutil.TouchNotebook(t, tmpDir, "notebook.py")

	f := NewFile(nil, filepath.Join(tmpDir, "test.py"), "test.py")
	isNotebook, err := f.IsNotebook()
	require.NoError(t, err)
	require.False(t, isNotebook)

	f = NewFile(nil, filepath.Join(tmpDir, "notebook.py"), "notebook.py")
	isNotebook, err = f.IsNotebook()
	require.NoError(t, err)
	require.True(t, isNotebook)
}
