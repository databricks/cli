package sync

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"path"
	"testing"

	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubLister struct {
	out []filer.RemoteFileMetadata
	err error
}

func (s *stubLister) ListWithSHAs(ctx context.Context, dirPath string) ([]filer.RemoteFileMetadata, error) {
	return s.out, s.err
}

func sha256OfFile(t *testing.T, root vfs.Path, relative string) string {
	t.Helper()
	r, err := root.Open(relative)
	require.NoError(t, err)
	defer r.Close()
	h := sha256.New()
	_, err = h.Write(mustReadAll(t, r))
	require.NoError(t, err)
	return hex.EncodeToString(h.Sum(nil))
}

func mustReadAll(t *testing.T, r interface{ Read(p []byte) (int, error) }) []byte {
	t.Helper()
	buf := make([]byte, 0, 1024)
	tmp := make([]byte, 256)
	for {
		n, err := r.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
		}
		if err != nil {
			break
		}
	}
	return buf
}

const remoteRoot = "/Workspace/Users/foo@databricks.com/proj"

// loadTestFiles returns the test fileset and the mtime-based SnapshotState
// (which provides LocalToRemoteNames). All tests share the same on-disk
// fileset under testdata/sync-fileset.
func loadTestFiles(t *testing.T) ([]fileset.File, map[string]string) {
	t.Helper()
	fs := fileset.New(vfs.MustNew("./testdata/sync-fileset"))
	files, err := fs.Files()
	require.NoError(t, err)
	state, err := NewSnapshotState(files)
	require.NoError(t, err)
	return files, state.LocalToRemoteNames
}

func TestRemoteFilterReturnsUnchangedWhenNoPuts(t *testing.T) {
	rf := NewRemoteFilter(&stubLister{}, remoteRoot)
	d := diff{delete: []string{"old"}, rmdir: []string{"olddir"}}
	got := rf.Apply(t.Context(), d, nil, nil)
	assert.Equal(t, d.delete, got.delete)
	assert.Equal(t, d.rmdir, got.rmdir)
	assert.Empty(t, got.put)
}

func TestRemoteFilterReturnsUnchangedWhenListErrors(t *testing.T) {
	files, ltr := loadTestFiles(t)
	rf := NewRemoteFilter(&stubLister{err: errors.New("boom")}, remoteRoot)
	in := diff{put: []string{"my-script.py"}}
	got := rf.Apply(t.Context(), in, files, ltr)
	assert.Equal(t, []string{"my-script.py"}, got.put)
}

func TestRemoteFilterReturnsUnchangedWhenRemoteEmpty(t *testing.T) {
	files, ltr := loadTestFiles(t)
	rf := NewRemoteFilter(&stubLister{out: nil}, remoteRoot)
	in := diff{put: []string{"my-script.py"}}
	got := rf.Apply(t.Context(), in, files, ltr)
	assert.Equal(t, []string{"my-script.py"}, got.put)
}

func TestRemoteFilterDropsPutWhenSHAMatches(t *testing.T) {
	files, ltr := loadTestFiles(t)
	root := vfs.MustNew("./testdata/sync-fileset")
	wantSHA := sha256OfFile(t, root, "my-script.py")

	rf := NewRemoteFilter(&stubLister{out: []filer.RemoteFileMetadata{
		{Path: path.Join(remoteRoot, "my-script.py"), ContentSHA256Hex: wantSHA, ObjectType: "FILE"},
	}}, remoteRoot)

	in := diff{put: []string{"my-script.py"}}
	got := rf.Apply(t.Context(), in, files, ltr)
	assert.Empty(t, got.put)
}

func TestRemoteFilterKeepsPutWhenSHAMismatches(t *testing.T) {
	files, ltr := loadTestFiles(t)
	rf := NewRemoteFilter(&stubLister{out: []filer.RemoteFileMetadata{
		{Path: path.Join(remoteRoot, "my-script.py"), ContentSHA256Hex: "deadbeef", ObjectType: "FILE"},
	}}, remoteRoot)

	in := diff{put: []string{"my-script.py"}}
	got := rf.Apply(t.Context(), in, files, ltr)
	assert.Equal(t, []string{"my-script.py"}, got.put)
}

func TestRemoteFilterKeepsPutWhenRemoteMissing(t *testing.T) {
	files, ltr := loadTestFiles(t)
	rf := NewRemoteFilter(&stubLister{out: []filer.RemoteFileMetadata{
		{Path: path.Join(remoteRoot, "other-file"), ContentSHA256Hex: "x"},
	}}, remoteRoot)

	in := diff{put: []string{"my-script.py"}}
	got := rf.Apply(t.Context(), in, files, ltr)
	assert.Equal(t, []string{"my-script.py"}, got.put)
}

func TestRemoteFilterDropsPutForNotebookWhenSHAMatches(t *testing.T) {
	// Notebooks are stored without their .py extension on the workspace —
	// LocalToRemoteNames maps "my-nb.py" -> "my-nb". The filter must use the
	// remote name when looking up the SHA.
	files, ltr := loadTestFiles(t)
	root := vfs.MustNew("./testdata/sync-fileset")
	wantSHA := sha256OfFile(t, root, "my-nb.py")
	require.Equal(t, "my-nb", ltr["my-nb.py"])

	rf := NewRemoteFilter(&stubLister{out: []filer.RemoteFileMetadata{
		{Path: path.Join(remoteRoot, "my-nb"), ContentSHA256Hex: wantSHA, ObjectType: "NOTEBOOK"},
	}}, remoteRoot)

	in := diff{put: []string{"my-nb.py"}}
	got := rf.Apply(t.Context(), in, files, ltr)
	assert.Empty(t, got.put)
}

func TestRemoteFilterMixedDiff(t *testing.T) {
	files, ltr := loadTestFiles(t)
	root := vfs.MustNew("./testdata/sync-fileset")
	scriptSHA := sha256OfFile(t, root, "my-script.py")
	validNbSHA := sha256OfFile(t, root, "valid-nb.ipynb")

	rf := NewRemoteFilter(&stubLister{out: []filer.RemoteFileMetadata{
		{Path: path.Join(remoteRoot, "my-script.py"), ContentSHA256Hex: scriptSHA, ObjectType: "FILE"},
		{Path: path.Join(remoteRoot, "my-nb"), ContentSHA256Hex: "stale", ObjectType: "NOTEBOOK"},
		{Path: path.Join(remoteRoot, "valid-nb"), ContentSHA256Hex: validNbSHA, ObjectType: "NOTEBOOK"},
	}}, remoteRoot)

	in := diff{
		put:    []string{"my-script.py", "my-nb.py", "valid-nb.ipynb"},
		delete: []string{"gone"},
		mkdir:  []string{"new"},
	}
	got := rf.Apply(t.Context(), in, files, ltr)
	assert.ElementsMatch(t, []string{"my-nb.py"}, got.put, "only the SHA-mismatched notebook should be re-uploaded")
	assert.Equal(t, []string{"gone"}, got.delete)
	assert.Equal(t, []string{"new"}, got.mkdir)
}
