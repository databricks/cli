package artifacts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/databricks/cli/libs/fileset"
	"github.com/databricks/cli/libs/vfs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// readTar returns entry name -> content, mode bits, and modtime.
func readTar(t *testing.T, b []byte) (map[string]string, map[string]int64, map[string]time.Time) {
	t.Helper()
	gzr, err := gzip.NewReader(bytes.NewReader(b))
	require.NoError(t, err)
	tr := tar.NewReader(gzr)
	content := map[string]string{}
	modes := map[string]int64{}
	mtimes := map[string]time.Time{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		body, err := io.ReadAll(tr)
		require.NoError(t, err)
		content[hdr.Name] = string(body)
		modes[hdr.Name] = hdr.Mode
		mtimes[hdr.Name] = hdr.ModTime
	}
	return content, modes, mtimes
}

func TestAddFileToTarball(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "train.py"), []byte("print('x')\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "run.sh"), []byte("echo hi\n"), 0o755))

	root := vfs.MustNew(dir)
	list, err := fileset.New(root).Files()
	require.NoError(t, err)

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	for _, f := range list {
		require.NoError(t, addFileToTarball(tw, root, f))
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	content, modes, mtimes := readTar(t, buf.Bytes())

	// Entries keep their sync-root-relative slash paths.
	assert.Equal(t, "print('x')\n", content["src/train.py"])
	assert.Equal(t, "echo hi\n", content["run.sh"])

	// Reproducible: every entry is stamped with the fixed epoch.
	assert.True(t, mtimes["src/train.py"].Equal(tarballEpoch))

	// Owner execute bit preserved; other files normalized to 0644. Windows has no
	// execute bit so skip the mode assertions there.
	if runtime.GOOS != "windows" {
		assert.Equal(t, int64(0o755), modes["run.sh"])
		assert.Equal(t, int64(0o644), modes["src/train.py"])
	}
}

// TestTarballComposesFilesAcrossDirs is the packing-layer proof of the `include`
// semantics: a tgz composed from a code directory and a sibling file outside it
// (e.g. a local env file not under code_source) keeps both, with bundle-root-relative
// entry names. This is what distinguishes DABs `include` from the air CLI's
// code-source-root-relative include. Full include-selection behavior (which paths the
// sync walker yields) is covered by acceptance tests, since it needs a WorkspaceClient.
func TestTarballComposesFilesAcrossDirs(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "src"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "config"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "src", "train.py"), []byte("print('x')\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "config", "train.env"), []byte("KEY=1\n"), 0o644))

	root := vfs.MustNew(dir)
	list, err := fileset.New(root).Files()
	require.NoError(t, err)

	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	for _, f := range list {
		require.NoError(t, addFileToTarball(tw, root, f))
	}
	require.NoError(t, tw.Close())
	require.NoError(t, gzw.Close())

	content, _, _ := readTar(t, buf.Bytes())
	// The code file and the sibling config file both land, each under its own
	// bundle-root-relative path — not collapsed to a single code-source root.
	assert.Equal(t, "print('x')\n", content["src/train.py"])
	assert.Equal(t, "KEY=1\n", content["config/train.env"])
}
