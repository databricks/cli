package tarpack

import (
	"archive/tar"
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type readEntry struct {
	typeflag byte
	mode     int64
	linkname string
	content  string
}

func readTar(t *testing.T, b []byte) map[string]readEntry {
	t.Helper()
	tr := tar.NewReader(bytes.NewReader(b))
	out := map[string]readEntry{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		data, err := io.ReadAll(tr)
		require.NoError(t, err)
		out[hdr.Name] = readEntry{
			typeflag: hdr.Typeflag,
			mode:     hdr.Mode,
			linkname: hdr.Linkname,
			content:  string(data),
		}
	}
	return out
}

func TestWriteRegularFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("alpha"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("beta"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "run.sh"), []byte("#!/bin/sh\n"), 0o644))
	// Chmod explicitly (not via the WriteFile mode arg) so the umask can't strip
	// the bits we assert on below.
	require.NoError(t, os.Chmod(filepath.Join(dir, "a.txt"), 0o644))
	require.NoError(t, os.Chmod(filepath.Join(dir, "run.sh"), 0o755))

	var buf bytes.Buffer
	err := Write(&buf, []Entry{
		{Name: "pkg/a.txt", Path: filepath.Join(dir, "a.txt")},
		{Name: "pkg/sub/b.txt", Path: filepath.Join(dir, "sub", "b.txt")},
		{Name: "pkg/run.sh", Path: filepath.Join(dir, "run.sh")},
	})
	require.NoError(t, err)

	got := readTar(t, buf.Bytes())
	require.Len(t, got, 3)
	assert.Equal(t, "alpha", got["pkg/a.txt"].content)
	assert.Equal(t, byte(tar.TypeReg), got["pkg/a.txt"].typeflag)
	assert.Equal(t, "beta", got["pkg/sub/b.txt"].content)
	// The owner execute bit is preserved (Windows has no execute bit).
	if runtime.GOOS != "windows" {
		assert.Equal(t, int64(0o755), got["pkg/run.sh"].mode&0o777)
		assert.Equal(t, int64(0o644), got["pkg/a.txt"].mode&0o777)
	}
}

func TestWriteSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation needs privilege on Windows")
	}
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("alpha"), 0o644))
	require.NoError(t, os.Symlink("a.txt", filepath.Join(dir, "link")))

	var buf bytes.Buffer
	err := Write(&buf, []Entry{
		{Name: "pkg/link", Path: filepath.Join(dir, "link")},
	})
	require.NoError(t, err)

	got := readTar(t, buf.Bytes())
	require.Len(t, got, 1)
	// Stored as a symlink entry, not the dereferenced content.
	assert.Equal(t, byte(tar.TypeSymlink), got["pkg/link"].typeflag)
	assert.Equal(t, "a.txt", got["pkg/link"].linkname)
	assert.Empty(t, got["pkg/link"].content)
}

func TestWriteSkipsIrregularAndMissingIsError(t *testing.T) {
	dir := t.TempDir()

	// A directory entry is skipped (directories are implied by entry names).
	var buf bytes.Buffer
	require.NoError(t, Write(&buf, []Entry{{Name: "pkg/sub", Path: dir}}))
	assert.Empty(t, readTar(t, buf.Bytes()))

	// A missing path is a hard error, not silently dropped.
	err := Write(io.Discard, []Entry{{Name: "pkg/gone", Path: filepath.Join(dir, "gone")}})
	assert.Error(t, err)
}
