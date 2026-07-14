package aicode

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Tarball builder ported from PR #5897 (in turn from cli/utils/snapshot.py). It
// shells out to `tar` to reuse tar's symlink, gitignore, and AppleDouble handling.
// The tarball's top-level dir name is load-bearing — the remote entry_script
// extracts to /databricks/code_source/<dir> — so the `-C parent dir` form preserves it.

// createPlainTarball writes a gzipped tar of repoPath's working tree to
// outputTarball via `tar`. The archive preserves repoPath's directory name as the
// top-level entry. .git and macOS AppleDouble files are always excluded; a
// .gitignore at repoPath is honored.
func createPlainTarball(ctx context.Context, repoPath, outputTarball string) error {
	dirName := filepath.Base(repoPath)
	parent := filepath.Dir(repoPath)

	args := []string{"-czf", outputTarball}

	// Exclude macOS AppleDouble files: they sort before the real top-level dir and
	// hijack a remote `head -1` parse. No-op on Linux.
	args = append(args, "--exclude=._*")

	// Never ship .git — it is large and unused on the remote.
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

	// Archive from the parent so the directory name is preserved as the top-level entry.
	args = append(args, "-C", parent, dirName)

	cmd := exec.CommandContext(ctx, "tar", args...)
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
