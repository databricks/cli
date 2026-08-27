package aircmd

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
func createPlainTarball(ctx context.Context, repoPath, outputTarball string, includePaths []string) error {
	dirName := filepath.Base(repoPath)
	// Absolute so it resolves correctly regardless of tar's working dir (set below).
	parent, err := filepath.Abs(filepath.Dir(repoPath))
	if err != nil {
		return err
	}

	// Pass the archive path relative to its own directory (run tar there), never a
	// full path: on Windows an absolute path like `C:\out\x.tar.gz` makes tar read
	// the `C:` as a remote host ("Cannot connect to C:"), since tar treats a colon
	// in the -f arg as host:path. A bare basename with -C avoids that on GNU tar and
	// bsdtar alike.
	outDirAbs, err := filepath.Abs(filepath.Dir(outputTarball))
	if err != nil {
		return err
	}
	outName := filepath.Base(outputTarball)

	files, err := snapshotFiles(ctx, repoPath, includePaths)
	if err != nil {
		return err
	}
	args := []string{"-czf", outName, "-C", parent, "--null", "--no-recursion", "-T", "-"}

	cmd := exec.CommandContext(ctx, "tar", args...)
	// Run tar in the output directory so the bare -f basename lands there.
	cmd.Dir = outDirAbs
	var stdin bytes.Buffer
	for _, file := range files {
		stdin.WriteString(filepath.ToSlash(filepath.Join(dirName, file)))
		stdin.WriteByte(0)
	}
	cmd.Stdin = &stdin
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return fmt.Errorf("failed to create plain tarball: %w: %s", err, msg)
		}
		return fmt.Errorf("failed to create plain tarball: %w", err)
	}
	return nil
}

func snapshotFiles(ctx context.Context, repoPath string, includePaths []string) ([]string, error) {
	args := []string{"-C", repoPath, "ls-files", "-z", "--cached", "--others", "--exclude-standard"}
	if !newGitRepo(repoPath).isRepository(ctx) {
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
		files = append(files, filepath.FromSlash(name))
	}
	return files, nil
}
