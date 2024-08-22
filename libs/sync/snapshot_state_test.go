package sync

import (
	"runtime"
	"testing"
	"time"

	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshotState(t *testing.T) {
	fileSet := fileset.New(vfs.MustNew("./testdata/sync-fileset"))
	files, err := fileSet.Files()
	require.NoError(t, err)

	// Assert initial contents of the fileset
	assert.Len(t, files, 4)
	assert.Equal(t, "invalid-nb.ipynb", files[0].Relative)
	assert.Equal(t, "my-nb.py", files[1].Relative)
	assert.Equal(t, "my-script.py", files[2].Relative)
	assert.Equal(t, "valid-nb.ipynb", files[3].Relative)

	// Assert snapshot state generated from the fileset. Note that the invalid notebook
	// has been ignored.
	s, err := NewSnapshotState(files)
	require.NoError(t, err)
	assertKeysOfMap(t, s.LastModifiedTimes, []string{"valid-nb.ipynb", "my-nb.py", "my-script.py"})
	assertKeysOfMap(t, s.LocalToRemoteNames, []string{"valid-nb.ipynb", "my-nb.py", "my-script.py"})
	assertKeysOfMap(t, s.RemoteToLocalNames, []string{"valid-nb", "my-nb", "my-script.py"})
	assert.NoError(t, s.validate())
}

func TestSnapshotStateValidationErrors(t *testing.T) {
	s := &SnapshotState{
		LastModifiedTimes: map[string]time.Time{
			"a": time.Now(),
		},
		LocalToRemoteNames: make(map[string]string),
		RemoteToLocalNames: make(map[string]string),
	}
	assert.EqualError(t, s.validate(), "invalid sync state representation. Local file a is missing the corresponding remote file")

	s = &SnapshotState{
		LastModifiedTimes:  map[string]time.Time{},
		LocalToRemoteNames: make(map[string]string),
		RemoteToLocalNames: map[string]string{
			"a": "b",
		},
	}
	assert.EqualError(t, s.validate(), "invalid sync state representation. local file b is missing the corresponding remote file")

	s = &SnapshotState{
		LastModifiedTimes: map[string]time.Time{
			"a": time.Now(),
		},
		LocalToRemoteNames: map[string]string{
			"a": "b",
		},
		RemoteToLocalNames: make(map[string]string),
	}
	assert.EqualError(t, s.validate(), "invalid sync state representation. Remote file b is missing the corresponding local file")

	s = &SnapshotState{
		LastModifiedTimes: make(map[string]time.Time),
		LocalToRemoteNames: map[string]string{
			"a": "b",
		},
		RemoteToLocalNames: map[string]string{
			"b": "a",
		},
	}
	assert.EqualError(t, s.validate(), "invalid sync state representation. Local file a is missing it's last modified time")

	s = &SnapshotState{
		LastModifiedTimes: map[string]time.Time{
			"a": time.Now(),
		},
		LocalToRemoteNames: map[string]string{
			"a": "b",
		},
		RemoteToLocalNames: map[string]string{
			"b": "b",
		},
	}
	assert.EqualError(t, s.validate(), "invalid sync state representation. Inconsistent values found. Local file a points to b. Remote file b points to b")

	s = &SnapshotState{
		LastModifiedTimes: map[string]time.Time{
			"a": time.Now(),
			"c": time.Now(),
		},
		LocalToRemoteNames: map[string]string{
			"a": "b",
			"c": "b",
		},
		RemoteToLocalNames: map[string]string{
			"b": "a",
		},
	}
	assert.EqualError(t, s.validate(), "invalid sync state representation. Inconsistent values found. Local file c points to b. Remote file b points to a")

	s = &SnapshotState{
		LastModifiedTimes: map[string]time.Time{
			"a": time.Now(),
		},
		LocalToRemoteNames: map[string]string{
			"a": "b",
		},
		RemoteToLocalNames: map[string]string{
			"b": "a",
			"c": "a",
		},
	}
	assert.EqualError(t, s.validate(), "invalid sync state representation. Inconsistent values found. Remote file c points to a. Local file a points to b")
}

func TestSnapshotStateWithBackslashes(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping test on non-Windows platform")
	}

	now := time.Now()
	s1 := &SnapshotState{
		LastModifiedTimes: map[string]time.Time{
			"foo\\bar.py": now,
		},
		LocalToRemoteNames: map[string]string{
			"foo\\bar.py": "foo/bar",
		},
		RemoteToLocalNames: map[string]string{
			"foo/bar": "foo\\bar.py",
		},
	}

	assert.NoError(t, s1.validate())

	s2 := s1.ToSlash()
	assert.NoError(t, s1.validate())
	assert.Equal(t, map[string]time.Time{"foo/bar.py": now}, s2.LastModifiedTimes)
	assert.Equal(t, map[string]string{"foo/bar.py": "foo/bar"}, s2.LocalToRemoteNames)
	assert.Equal(t, map[string]string{"foo/bar": "foo/bar.py"}, s2.RemoteToLocalNames)
}
