package sync

import (
	"testing"

	"github.com/databricks/cli/libs/fileset"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotState(t *testing.T) {
	fileSet := fileset.New("./testdata/sync-fileset")
	files, err := fileSet.All()
	require.NoError(t, err)

	// Assert initial contents of the fileset
	assert.Len(t, files, 4)
	assert.Equal(t, "invalid-nb.ipynb", files[0].Name())
	assert.Equal(t, "my-nb.py", files[1].Name())
	assert.Equal(t, "my-script.py", files[2].Name())
	assert.Equal(t, "valid-nb.ipynb", files[3].Name())

	// Assert snapshot state generated from the fileset. Note that the invalid notebook
	// has been ignored.
	s, err := NewSnapshotState(files)
	require.NoError(t, err)
	assertKeysOfMap(t, s.LastModifiedTimes, []string{"valid-nb.ipynb", "my-nb.py", "my-script.py"})
	assertKeysOfMap(t, s.LocalToRemoteNames, []string{"valid-nb.ipynb", "my-nb.py", "my-script.py"})
	assertKeysOfMap(t, s.RemoteToLocalNames, []string{"valid-nb", "my-nb", "my-script.py"})
}
