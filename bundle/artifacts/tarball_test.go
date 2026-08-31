package artifacts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// tarEntries reads a gzipped tarball and returns entry name -> content.
func tarEntries(t *testing.T, b []byte) map[string]string {
	t.Helper()
	gzr, err := gzip.NewReader(bytes.NewReader(b))
	require.NoError(t, err)
	tr := tar.NewReader(gzr)
	out := map[string]string{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		body, err := io.ReadAll(tr)
		require.NoError(t, err)
		out[hdr.Name] = string(body)
	}
	return out
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git %v: %s", args, out)
}

// TestTarballFromGitPrefixesEntries verifies git mode nests every entry under the
// code-source dir name (the load-bearing top-level the runtime extracts to
// /databricks/code_source/<dir>), matching aicode and the air CLI.
func TestTarballFromGitPrefixesEntries(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "config", "user.email", "t@example.com")
	runGit(t, repo, "config", "user.name", "t")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "train.py"), []byte("print('x')\n"), 0o644))
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-qm", "init")

	b := &bundle.Bundle{SyncRootPath: repo}
	a := &config.Artifact{Path: repo, Git: &config.ArtifactGit{Commit: "HEAD"}}

	var buf bytes.Buffer
	require.NoError(t, tarballFromGit(t.Context(), b, a, &buf))

	dir := filepath.Base(repo)
	entries := tarEntries(t, buf.Bytes())
	assert.Equal(t, "print('x')\n", entries[dir+"/train.py"])
}

func TestTarballFromGitRequiresRef(t *testing.T) {
	b := &bundle.Bundle{SyncRootPath: t.TempDir()}
	a := &config.Artifact{Path: b.SyncRootPath, Git: &config.ArtifactGit{}}
	err := tarballFromGit(t.Context(), b, a, io.Discard)
	require.ErrorContains(t, err, "git.commit or git.branch")
}
