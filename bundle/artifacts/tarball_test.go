package artifacts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

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
		if errors.Is(err, io.EOF) {
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

// TestTarballFromGitArchivesRef verifies git mode archives the committed tree with
// entries named relative to the artifact's path (here the repo root, so no prefix).
func TestTarballFromGitArchivesRef(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "config", "user.email", "t@example.com")
	runGit(t, repo, "config", "user.name", "t")
	// Keep line endings verbatim so the content check doesn't depend on the runner's
	// core.autocrlf (true by default on the Windows CI image).
	runGit(t, repo, "config", "core.autocrlf", "false")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "train.py"), []byte("print('x')\n"), 0o644))
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-qm", "init")

	b := &bundle.Bundle{SyncRootPath: repo}
	a := &config.Artifact{Path: repo, Git: &config.ArtifactGit{Commit: "HEAD"}}

	var buf bytes.Buffer
	require.NoError(t, tarballFromGit(t.Context(), b, a, &buf))

	// path == repo root, so the entry is relative to it (no injected prefix).
	entries := tarEntries(t, buf.Bytes())
	require.Contains(t, entries, "train.py")
	assert.Equal(t, "print('x')", strings.TrimSpace(entries["train.py"]))
}

func TestTarballFromGitRequiresRef(t *testing.T) {
	b := &bundle.Bundle{SyncRootPath: t.TempDir()}
	a := &config.Artifact{Path: b.SyncRootPath, Git: &config.ArtifactGit{}}
	err := tarballFromGit(t.Context(), b, a, io.Discard)
	require.ErrorContains(t, err, "git.commit or git.branch")
}

// The tarball is content-addressed on upload, so the same tree must always compress to
// the same bytes. We now gzip with parallel pgzip; this guards that its output stays
// deterministic (pgzip only parallelizes across fixed-size blocks, so core count doesn't
// leak into the bytes) and that it decompresses cleanly.
func TestTarballFromGitIsReproducible(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "config", "user.email", "t@example.com")
	runGit(t, repo, "config", "user.name", "t")
	runGit(t, repo, "config", "core.autocrlf", "false")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "train.py"), []byte("print('x')\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "big.txt"), bytes.Repeat([]byte("compress me\n"), 200000), 0o644))
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-qm", "init")

	b := &bundle.Bundle{SyncRootPath: repo}
	a := &config.Artifact{Path: repo, Git: &config.ArtifactGit{Commit: "HEAD"}}

	var buf1, buf2 bytes.Buffer
	require.NoError(t, tarballFromGit(t.Context(), b, a, &buf1))
	require.NoError(t, tarballFromGit(t.Context(), b, a, &buf2))

	assert.Equal(t, buf1.Bytes(), buf2.Bytes(), "identical tree must compress to identical bytes")
	// Round-trips through gzip and carries the packed file.
	assert.Equal(t, "print('x')", strings.TrimSpace(tarEntries(t, buf1.Bytes())["train.py"]))
}

// TestTarballFromGitSubdirIsReproducible covers a `path` below the repository root:
// entries are relative to `path` and carry the commit time, not the current time,
// so rebuilding the same commit gives the same bytes.
func TestTarballFromGitSubdirIsReproducible(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "config", "user.email", "t@example.com")
	runGit(t, repo, "config", "user.name", "t")
	runGit(t, repo, "config", "core.autocrlf", "false")
	src := filepath.Join(repo, "proj", "src")
	require.NoError(t, os.MkdirAll(src, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "main.py"), []byte("print('x')\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(repo, "unrelated.py"), []byte("print('y')\n"), 0o644))
	runGit(t, repo, "add", "-A")
	commitTime := time.Unix(1700000000, 0)
	cmd := exec.Command("git", "commit", "-qm", "init")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "GIT_COMMITTER_DATE=@1700000000 +0000", "GIT_AUTHOR_DATE=@1700000000 +0000")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "git commit: %s", out)

	b := &bundle.Bundle{SyncRootPath: filepath.Join(repo, "proj")}
	a := &config.Artifact{Path: src, Git: &config.ArtifactGit{Commit: "HEAD"}}

	var buf bytes.Buffer
	require.NoError(t, tarballFromGit(t.Context(), b, a, &buf))

	gzr, err := gzip.NewReader(bytes.NewReader(buf.Bytes()))
	require.NoError(t, err)
	tr := tar.NewReader(gzr)
	var names []string
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		if hdr.Typeflag == tar.TypeXGlobalHeader {
			continue
		}
		names = append(names, hdr.Name)
		assert.True(t, commitTime.Equal(hdr.ModTime), "%s: mtime %v, want commit time %v", hdr.Name, hdr.ModTime, commitTime)
	}
	assert.Equal(t, []string{"main.py"}, names)
}

// TestTarballFromGitRejectsPathMissingFromRef: git archive run from a directory the
// ref doesn't contain succeeds with an empty archive, which must not be uploaded.
func TestTarballFromGitRejectsPathMissingFromRef(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "config", "user.email", "t@example.com")
	runGit(t, repo, "config", "user.name", "t")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "README"), []byte("x\n"), 0o644))
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-qm", "init")
	src := filepath.Join(repo, "src")
	require.NoError(t, os.MkdirAll(src, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "main.py"), []byte("print('x')\n"), 0o644))

	b := &bundle.Bundle{SyncRootPath: repo}
	a := &config.Artifact{Path: src, Git: &config.ArtifactGit{Commit: "HEAD"}}

	var buf bytes.Buffer
	err := tarballFromGit(t.Context(), b, a, &buf)
	assert.ErrorContains(t, err, "is not in the tree of HEAD")
}

// TestTarballFromGitReportsUnknownRef: a ref that doesn't resolve keeps git's own
// error rather than the "not in the tree" one meant for a missing path.
func TestTarballFromGitReportsUnknownRef(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	runGit(t, repo, "config", "user.email", "t@example.com")
	runGit(t, repo, "config", "user.name", "t")
	require.NoError(t, os.WriteFile(filepath.Join(repo, "README"), []byte("x\n"), 0o644))
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-qm", "init")

	b := &bundle.Bundle{SyncRootPath: repo}
	a := &config.Artifact{Path: repo, Git: &config.ArtifactGit{Branch: "no-such-branch"}}

	var buf bytes.Buffer
	err := tarballFromGit(t.Context(), b, a, &buf)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "git archive no-such-branch")
	assert.NotContains(t, err.Error(), "is not in the tree")
}
