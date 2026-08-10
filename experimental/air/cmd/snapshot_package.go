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

	args := []string{"-czf", outName}

	// Exclude macOS AppleDouble files: they sort before the real top-level dir and
	// hijack a remote `head -1` parse. No-op on Linux.
	args = append(args, "--exclude=._*")

	// Never ship .git — provenance flows via the git_state.json sidecar.
	args = append(args, "--exclude=.git")

	// Honor .gitignore if present.
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	if patterns, err := parseGitignore(gitignorePath); err == nil {
		for _, p := range patterns {
			if strings.Contains(p, "/") {
				// Anchor path-relative patterns to the archive root so they don't
				// match identically-named paths in subdirectories.
				args = append(args, "--exclude="+dirName+"/"+strings.TrimPrefix(p, "/"))
			} else {
				args = append(args, "--exclude="+p)
			}
		}
	}

	// Archive from the parent so the directory name is preserved; with include_paths,
	// prefix each so entries nest under it (matching git archive --prefix). -C only
	// affects the file operands that follow it, not the -f archive path (which
	// resolves against tar's working dir, set to outDirAbs below).
	args = append(args, "-C", parent)
	if len(includePaths) > 0 {
		for _, p := range includePaths {
			args = append(args, dirName+"/"+p)
		}
	} else {
		args = append(args, dirName)
	}

	cmd := exec.CommandContext(ctx, "tar", args...)
	// Run tar in the output directory so the bare -f basename lands there.
	cmd.Dir = outDirAbs
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

// parseGitignore reads a .gitignore and returns tar --exclude patterns. It mirrors
// the Python CLI's lossy normalization so plain-tar snapshots exclude the same set:
//
//   - comments (#…) and blank lines are skipped;
//   - negation patterns (!…) are unsupported by tar --exclude and skipped;
//   - a trailing "/" (directory marker) is stripped;
//   - "**" is not a path-separator-agnostic wildcard in tar, so "**/foo" → "foo"
//     and "foo/**" → "foo"; a mid-path "**" has no tar equivalent and is skipped.
//
// A missing file returns (nil, error); callers treat any error as "no patterns".
func parseGitignore(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var patterns []string
	for raw := range strings.SplitSeq(string(data), "\n") {
		line := strings.TrimRight(raw, " \t\r")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "!") {
			continue
		}
		line = strings.TrimRight(line, "/")
		if strings.Contains(line, "**") {
			switch {
			case strings.HasPrefix(line, "**/"):
				line = line[len("**/"):]
			case strings.HasSuffix(line, "/**"):
				line = line[:len(line)-len("/**")]
			default:
				continue
			}
		}
		patterns = append(patterns, line)
	}
	return patterns, nil
}
