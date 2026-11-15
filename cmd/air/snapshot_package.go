package aircmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
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

// createGitArchiveSnapshot writes a gzipped tar of commitSHA to outputTarball, with every
// entry prefixed by directoryName/. The tar is streamed from `git archive --format=tar`
// into a parallel pgzip writer: git's own tar.gz format compresses on one thread, which
// dominates packaging time on a large repo. When subtreePrefix is set, only that
// repository subtree is archived and includePaths are relative to it.
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
		"--format=tar",
		"--prefix=" + directoryName + "/",
		treeish,
	}
	args = append(args, includePaths...)

	out, err := os.Create(outputTarball)
	if err != nil {
		return fmt.Errorf("failed to create tarball: %w", err)
	}
	gz, err := pgzip.NewWriterLevel(out, pgzip.DefaultCompression)
	if err != nil {
		out.Close()
		return err
	}
	if err := archiveGit.runTo(ctx, ctxWriter{ctx, gz}, args...); err != nil {
		gz.Close()
		out.Close()
		return fmt.Errorf("failed to create git archive: %w", err)
	}
	if err := gz.Close(); err != nil {
		out.Close()
		return fmt.Errorf("failed to finalize gzip: %w", err)
	}
	return out.Close()
}

// createPlainTarball writes a gzipped tar of the given working-tree files to
// outputTarball. The archive preserves repoPath's directory name as the top-level
// entry. files come pre-resolved from snapshotFiles (.gitignore honored, .git and
// macOS AppleDouble excluded), so the caller can reuse the same listing to
// content-address the upload.
func createPlainTarball(ctx context.Context, repoPath, outputTarball string, files []snapshotFile) error {
	dirName := filepath.Base(repoPath)

	entries := make([]tarpack.Entry, len(files))
	for i, file := range files {
		entries[i] = tarpack.Entry{
			Name: filepath.ToSlash(filepath.Join(dirName, file.rel)),
			Path: filepath.Join(repoPath, file.rel),
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
	if err := tarpack.Write(ctxWriter{ctx, gz}, entries); err != nil {
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

// ctxWriter fails writes once ctx is done, so a cancelled submission (for
// example, one whose config validation failed) stops packaging promptly instead
// of archiving the whole tree.
type ctxWriter struct {
	ctx context.Context
	w   io.Writer
}

func (w ctxWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	return w.w.Write(p)
}

// snapshotFile is a file selected for the snapshot: its repo-relative path (native
// separators) plus the metadata used to content-address the plain_tar upload.
type snapshotFile struct {
	rel        string
	size       int64
	modTime    int64 // Unix nanoseconds
	mode       uint32
	linkTarget string
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
		linkTarget := ""
		if info.Mode()&os.ModeSymlink != 0 {
			linkTarget, err = os.Readlink(filepath.Join(repoPath, filepath.FromSlash(name)))
			if err != nil {
				return nil, fmt.Errorf("failed to inspect snapshot symlink %q: %w", name, err)
			}
		}
		files = append(files, snapshotFile{
			rel:        filepath.FromSlash(name),
			size:       info.Size(),
			modTime:    info.ModTime().UnixNano(),
			mode:       uint32(info.Mode()),
			linkTarget: linkTarget,
		})
	}
	if len(files) == 0 {
		return nil, emptySnapshotError(ctx, repoPath, includePaths, isGitRepo)
	}
	return files, nil
}

// emptySnapshotError explains why no file was selected for the snapshot. An empty
// tarball is accepted by the upload but fails on the node, after compute was
// provisioned, so it is rejected here with the most likely cause.
func emptySnapshotError(ctx context.Context, repoPath string, includePaths []string, isGitRepo bool) error {
	root := filepath.ToSlash(repoPath)
	prefix := fmt.Sprintf("code_source snapshot of %s contains no files; refusing to upload an empty snapshot", root)
	if isGitRepo {
		// check-ignore -v prints "<source>:<line>:<pattern>\t<path>" and exits non-zero
		// when the path is not ignored.
		if out, err := newGitRepo(repoPath).run(ctx, "check-ignore", "-v", "."); err == nil {
			rule, _, _ := strings.Cut(strings.TrimSpace(out), "\t")
			return fmt.Errorf("%s: root_path is ignored by git (%s), and the snapshot only includes files git does not ignore. Move the project out of the ignored directory, un-ignore it, or change code_source.root_path. Inspect the rule with `git -C %s check-ignore -v .`", prefix, rule, root)
		}
	}
	if len(includePaths) > 0 {
		return fmt.Errorf("%s: no file under include_paths (%s) is selected. Check that the paths exist under root_path and are not ignored by git (`git check-ignore -v <path>`)", prefix, strings.Join(includePaths, ", "))
	}
	return fmt.Errorf("%s: the directory is empty or all of its files are excluded by .gitignore rules (`git check-ignore -v <path>` shows the rule for a file). Check code_source.root_path", prefix)
}
