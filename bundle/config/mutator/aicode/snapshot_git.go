package aicode

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Local, no-network git introspection for the code_source snapshot, ported from
// the Python CLI's cli/utils/git_state.py and PR #5897's snapshot_git.go. Only the
// subset the DABs resolve path needs is kept: is-repo, HEAD SHA, and dirty check.
// The remote-fetch helpers and the git_state.json/git_diff.patch provenance
// sidecars are intentionally not ported — the SDK ai_runtime_task has nowhere to
// attach sidecar paths, and the snapshot archives the local copy only.

// gitRepo runs git subcommands scoped to one repository via `git -C`. Arguments
// are passed as a slice, never a shell string, so path values can't inject.
type gitRepo struct {
	path string
}

func newGitRepo(path string) gitRepo {
	return gitRepo{path: path}
}

// run executes `git -C <path> <args...>` and returns trimmed stdout; a non-zero
// exit wraps stderr.
func (g gitRepo) run(ctx context.Context, args ...string) (string, error) {
	full := append([]string{"-C", g.path}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// isRepository reports whether the path is inside a git work tree. Using
// `rev-parse --is-inside-work-tree` (not a .git lookup) means a subdirectory of a
// repo counts — the common case when code_source_path is a subfolder of a monorepo.
func (g gitRepo) isRepository(ctx context.Context) bool {
	out, err := g.run(ctx, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false
	}
	return out == "true"
}

// headSHA returns the current HEAD commit SHA.
func (g gitRepo) headSHA(ctx context.Context) (string, error) {
	return g.run(ctx, "rev-parse", "HEAD")
}

// hasUncommittedChanges reports whether there are staged or unstaged changes under
// the repo subtree. The `-- .` pathspec scopes the check so a subfolder snapshot
// considers only changes that could land in it.
func (g gitRepo) hasUncommittedChanges(ctx context.Context) (bool, error) {
	out, err := g.run(ctx, "status", "--porcelain", "--", ".")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

// shortSHA abbreviates a commit SHA to 8 chars for log/error messages.
func shortSHA(sha string) string {
	return sha[:min(len(sha), 8)]
}
