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

	"github.com/klauspost/pgzip"
)

// Tar builders ported from cli/utils/snapshot.py. Both shell out (git archive / tar)
// for parity and to reuse git's/tar's symlink, gitignore, and AppleDouble handling.
// The tarball's top-level dir name is load-bearing — the remote entry_script extracts
// to /databricks/code_source/<dir> — so the --prefix / `-C parent dir` forms preserve it.

// createGitArchiveSnapshot writes a gzipped tar of commitSHA to outputTarball via
// `git archive`, with every entry prefixed by directoryName/. When includePaths is
// set, only those paths are archived.
func createGitArchiveSnapshot(ctx context.Context, git gitRepo, commitSHA, outputTarball, directoryName string, includePaths []string) error {
	args := []string{
		"archive",
		"--format=tar.gz",
		"--prefix=" + directoryName + "/",
		"-o", outputTarball,
		commitSHA,
	}
	args = append(args, includePaths...)
	if _, err := git.run(ctx, args...); err != nil {
		return fmt.Errorf("failed to create git archive: %w", err)
	}
	return nil
}

// createPlainTarball writes a gzipped tar of repoPath's working tree to
// outputTarball via `tar`. The archive preserves repoPath's directory name as the
// top-level entry. When includePaths is set, only those paths (nested under the
// directory name) are archived. .git and macOS AppleDouble files are always
// excluded; a .gitignore at repoPath is honored.
func createPlainTarball(ctx context.Context, repoPath, outputTarball string, includePaths []string, isGitRepo bool) error {
	dirName := filepath.Base(repoPath)
	// Absolute so it resolves correctly regardless of tar's working dir.
	parent, err := filepath.Abs(filepath.Dir(repoPath))
	if err != nil {
		return err
	}

	files, err := snapshotFiles(ctx, repoPath, includePaths, isGitRepo)
	if err != nil {
		return err
	}

	out, err := os.Create(outputTarball)
	if err != nil {
		return fmt.Errorf("failed to create tarball: %w", err)
	}

	// tar writes the uncompressed archive to stdout (`-cf -`) and we gzip it here with
	// klauspost/pgzip instead of tar's built-in -z: tar's gzip is single-threaded and
	// dominates packaging time on a large tree, whereas pgzip spreads it across cores
	// (measured ~18x faster on a ~470 MiB archive) while still emitting an ordinary gzip
	// stream. BestSpeed because the uploaded size does not matter for this workflow, only
	// latency. Compressing outside tar also passes no archive path to tar, sidestepping
	// the Windows colon-in-path issue a `-f <path>` argument otherwise hits (tar reads the
	// `C:` in `C:\out\x.tar.gz` as a remote host).
	gz, err := pgzip.NewWriterLevel(out, pgzip.BestSpeed)
	if err != nil {
		out.Close()
		return err
	}

	args := []string{"-cf", "-", "-C", parent, "--null", "--no-recursion", "-T", "-"}
	cmd := exec.CommandContext(ctx, "tar", args...)
	var stdin bytes.Buffer
	for _, file := range files {
		stdin.WriteString(filepath.ToSlash(filepath.Join(dirName, file)))
		stdin.WriteByte(0)
	}
	cmd.Stdin = &stdin
	cmd.Stdout = gz
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		gz.Close()
		out.Close()
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("failed to create plain tarball: %w: %s", err, msg)
		}
		return fmt.Errorf("failed to create plain tarball: %w", err)
	}
	if err := gz.Close(); err != nil {
		out.Close()
		return fmt.Errorf("failed to finalize gzip: %w", err)
	}
	return out.Close()
}

func snapshotFiles(ctx context.Context, repoPath string, includePaths []string, isGitRepo bool) ([]string, error) {
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

	var files []string
	for raw := range bytes.SplitSeq(output, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		name := filepath.ToSlash(string(raw))
		base := filepath.Base(name)
		if name == ".git" || strings.HasPrefix(name, ".git/") || strings.HasPrefix(base, "._") {
			continue
		}
		_, err := os.Lstat(filepath.Join(repoPath, filepath.FromSlash(name)))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to inspect snapshot path %q: %w", name, err)
		}
		files = append(files, filepath.FromSlash(name))
	}
	return files, nil
}
