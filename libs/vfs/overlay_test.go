package vfs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverlayServesVirtualFileAndRealFiles(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "real.txt"), []byte("real"), 0o644))

	base := MustNew(dir)
	ov, err := Overlay(base, map[string][]byte{
		".air_snapshots/src_abc.tar.gz": []byte("SNAPSHOT-BYTES"),
	})
	require.NoError(t, err)

	// Real file still readable through the overlay.
	got, err := ov.ReadFile("real.txt")
	require.NoError(t, err)
	assert.Equal(t, "real", string(got))

	// Virtual file readable via Open (the path sync's applyPut uses).
	f, err := ov.Open(".air_snapshots/src_abc.tar.gz")
	require.NoError(t, err)
	b, err := io.ReadAll(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	assert.Equal(t, "SNAPSHOT-BYTES", string(b))

	// Virtual file is NOT on disk — the working tree stays clean.
	assert.NoFileExists(t, filepath.Join(dir, ".air_snapshots", "src_abc.tar.gz"))
}

func TestOverlayWalkDirSurfacesVirtualFile(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "real.txt"), []byte("real"), 0o644))
	ov, err := Overlay(MustNew(dir), map[string][]byte{
		".air_snapshots/src_abc.tar.gz": []byte("x"),
	})
	require.NoError(t, err)

	var walked []string
	require.NoError(t, fs.WalkDir(ov, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			walked = append(walked, name)
		}
		return nil
	}))

	slices.Sort(walked)
	// Both the real file and the virtual snapshot (in its virtual dir) are walked,
	// so bundle file sync uploads the snapshot without it existing on disk.
	assert.Equal(t, []string{".air_snapshots/src_abc.tar.gz", "real.txt"}, walked)
}

// Names outside the root are rejected rather than silently mis-registered. An
// absolute name additionally used to hang the ancestor walk (path.Dir("/") == "/").
func TestOverlayRejectsNamesOutsideRoot(t *testing.T) {
	for _, name := range []string{
		"/foo/bar.txt",
		"../foo.txt",
		"a/../../b",
		"",
	} {
		t.Run(name, func(t *testing.T) {
			_, err := Overlay(MustNew(t.TempDir()), map[string][]byte{name: []byte("x")})
			require.ErrorContains(t, err, "invalid file name")
		})
	}
}

// A relative name that stays inside the root is accepted, including one that only
// normalizes to an in-root path after cleaning.
func TestOverlayAcceptsInRootNames(t *testing.T) {
	for _, name := range []string{"a.txt", "./b/c.txt", "d/../e.txt"} {
		t.Run(name, func(t *testing.T) {
			ov, err := Overlay(MustNew(t.TempDir()), map[string][]byte{name: []byte("x")})
			require.NoError(t, err)
			got, err := ov.ReadFile(name)
			require.NoError(t, err)
			assert.Equal(t, "x", string(got))
		})
	}
}
