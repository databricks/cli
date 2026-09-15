package aircmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/libs/tarpack"
	"github.com/klauspost/pgzip"
)

// Tar builders ported from cli/utils/snapshot.py. The git-ref path snapshots a commit
// with `git archive`; the plain path enumerates the working tree with `git ls-files`
// (honoring .gitignore) and writes the tar in Go. The tarball's top-level dir name is
// load-bearing — the remote entry_script extracts to /databricks/code_source/<dir> — so
// both forms prefix every entry with it.

// createGitArchiveSnapshot writes a gzipped tar of commitSHA to outputTarball via
// `git archive`, with every entry prefixed by directoryName/. When subtreePrefix is
// set, only that repository subtree is archived and includePaths are relative to it.
func createGitArchiveSnapshot(ctx context.Context, git gitRepo, commitSHA, outputTarball, directoryName string, includePaths []string, subtreePrefix string) error {
	treeish := commitSHA
	archiveGit := git
	if subtreePrefix != "" {
		repoRoot, err := git.repositoryRoot(ctx)
		if err != nil {
			return err
		}
		archiveGit = newGitRepo(repoRoot)
		treeish = commitSHA + ":" + subtreePrefix
	}

	args := []string{
		"archive",
		"--format=tar.gz",
		"--prefix=" + directoryName + "/",
		"-o", outputTarball,
		treeish,
	}
	args = append(args, includePaths...)
	if _, err := archiveGit.run(ctx, args...); err != nil {
		return fmt.Errorf("failed to create git archive: %w", err)
	}
	return nil
}

// createPlainTarball writes a gzipped tar of repoPath's working tree to
// outputTarball. The archive preserves repoPath's directory name as the top-level
// entry. When includePaths is set, only those paths (nested under the directory
// name) are archived. .git and macOS AppleDouble files are always excluded; a
// .gitignore at repoPath is honored.
func createPlainTarball(ctx context.Context, repoPath, outputTarball string, includePaths []string, isGitRepo bool) error {
	dirName := filepath.Base(repoPath)

	files, err := snapshotFiles(ctx, repoPath, includePaths, isGitRepo)
	if err != nil {
		return err
	}
	entries := make([]tarpack.Entry, len(files))
	for i, f := range files {
		entries[i] = tarpack.Entry{
			Name: filepath.ToSlash(filepath.Join(dirName, f.rel)),
			Path: filepath.Join(repoPath, f.rel),
		}
	}

	out, err := os.Create(outputTarball)
	if err != nil {
		return fmt.Errorf("failed to create tarball: %w", err)
	}

	// Gzip in parallel with klauspost/pgzip rather than a single-threaded writer:
	// compression dominates packaging time on a large tree, and pgzip spreads it
	// across cores. tarpack writes the tar in Go, so there is no `tar` subprocess
	// (and no Windows colon-in-path issue a `tar -f` argument would hit).
	gz, err := pgzip.NewWriterLevel(out, pgzip.DefaultCompression)
	if err != nil {
		out.Close()
		return err
	}
	if err := tarpack.Write(gz, entries); err != nil {
		gz.Close()
		out.Close()
		return fmt.Errorf("failed to create plain tarball: %w", err)
	}
	if err := gz.Close(); err != nil {
		out.Close()
		return fmt.Errorf("failed to finalize gzip: %w", err)
	}
	return out.Close()
}

// snapshotFile is a file selected for the snapshot: its repo-relative path (native
// separators) with the size and mtime the warm cache uses to detect changes.
type snapshotFile struct {
	rel     string
	size    int64
	modTime int64 // Unix nanoseconds
}

func snapshotFiles(ctx context.Context, repoPath string, includePaths []string, isGitRepo bool) ([]snapshotFile, error) {
	args := []string{"-C", repoPath, "ls-files", "-z", "--cached", "--others", "--exclude-standard"}
	if !isGitRepo {
		gitDir, err := os.MkdirTemp("", "air-snapshot-git-")
		if err != nil {
			return nil, fmt.Errorf("failed to create temporary git metadata: %w", err)
		}
		defer os.RemoveAll(gitDir)
		cmd := exec.CommandContext(ctx, "git", "--git-dir", gitDir, "--work-tree", repoPath, "init", "--quiet")
		if output, err := cmd.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("failed to initialize temporary git metadata: %w: %s", err, strings.TrimSpace(string(output)))
		}
		args = []string{"--git-dir", gitDir, "--work-tree", repoPath, "ls-files", "-z", "--cached", "--others", "--exclude-standard"}
	}

	if len(includePaths) > 0 {
		args = append(args, "--")
		args = append(args, includePaths...)
	}
	output, err := exec.CommandContext(ctx, "git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate git ignore rules: %w", err)
	}

	var files []snapshotFile
	for raw := range bytes.SplitSeq(output, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		name := filepath.ToSlash(string(raw))
		base := filepath.Base(name)
		if name == ".git" || strings.HasPrefix(name, ".git/") || strings.HasPrefix(base, "._") {
			continue
		}
		info, err := os.Lstat(filepath.Join(repoPath, filepath.FromSlash(name)))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to inspect snapshot path %q: %w", name, err)
		}
		files = append(files, snapshotFile{rel: filepath.FromSlash(name), size: info.Size(), modTime: info.ModTime().UnixNano()})
	}
	return files, nil
}
