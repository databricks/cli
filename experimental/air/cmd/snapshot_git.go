package aircmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// Local, no-network git introspection ported from the Python CLI's
// cli/utils/git_state.py. The remote-fetch helpers (fetch_branch_sha, remote
// detection, partial clone) are deliberately not ported: the snapshot archives the
// local copy only, so a ref must resolve to a local commit.

// gitRepo runs git subcommands scoped to one repository via `git -C`. Arguments are
// passed as a slice, never a shell string, so branch/commit values can't inject.
type gitRepo struct {
	path string
}

func newGitRepo(path string) gitRepo {
	return gitRepo{path: path}
}

// run executes `git <args...>` and returns stdout; a non-zero exit wraps stderr.
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
	return stdout.String(), nil
}

// isRepository reports whether the path is inside a git work tree. Using
// `rev-parse --is-inside-work-tree` (not a .git lookup) means a subdirectory of a
// repo counts — the common case when root_path is a subfolder of a monorepo.
func (g gitRepo) isRepository(ctx context.Context) bool {
	out, err := g.run(ctx, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "true"
}

// headSHA returns the current HEAD commit SHA.
func (g gitRepo) headSHA(ctx context.Context) (string, error) {
	out, err := g.run(ctx, "rev-parse", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// hasUncommittedChanges reports whether there are staged or unstaged changes under
// the repo subtree. The `-- .` pathspec scopes the check so a subfolder snapshot
// considers only changes that could land in it.
func (g gitRepo) hasUncommittedChanges(ctx context.Context) (bool, error) {
	out, err := g.run(ctx, "status", "--porcelain", "--", ".")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// hasUncommittedChangesInPaths reports whether there are uncommitted changes within
// the include paths (empty includePaths yields false).
//
// The pathspecs limit `git status` to those subtrees (it is O(working tree), slow on
// a large monorepo) and already scope the output to what could land in the snapshot.
// Unlike the Python source we don't re-parse the entries to filter by name: git
// reports a rename as `R <new>\x00<old>`, so a name-based re-filter keys off the old
// path and could miss a rename into an include path. The only caller needs the bool.
func (g gitRepo) hasUncommittedChangesInPaths(ctx context.Context, includePaths []string) (bool, error) {
	var pathspecs []string
	for _, p := range includePaths {
		if s := strings.TrimRight(p, "/"); s != "" {
			pathspecs = append(pathspecs, s)
		}
	}
	if len(pathspecs) == 0 {
		return false, nil
	}

	args := append([]string{"status", "--porcelain", "--"}, pathspecs...)
	out, err := g.run(ctx, args...)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// resolveLocalBranchSHA resolves a branch to its local-HEAD commit. No remote is
// contacted; the branch must exist locally.
func (g gitRepo) resolveLocalBranchSHA(ctx context.Context, branch string) (string, error) {
	out, err := g.run(ctx, "rev-parse", "refs/heads/"+branch)
	if err != nil {
		return "", fmt.Errorf("failed to resolve local branch %q; ensure the branch exists locally and root_path is correct: %w", branch, err)
	}
	return strings.TrimSpace(out), nil
}

// commitExistsLocally reports whether commitSHA is in the local object store, without
// triggering a promisor/lazy fetch.
func (g gitRepo) commitExistsLocally(ctx context.Context, commitSHA string) bool {
	_, err := g.run(ctx, "cat-file", "-e", commitSHA)
	return err == nil
}

// validateIncludePathsExist checks that every include path exists at commitSHA.
// `git ls-tree` (without -d, so both blobs and trees count) reports an entry when the
// path exists; empty output means missing.
func (g gitRepo) validateIncludePathsExist(ctx context.Context, commitSHA string, includePaths []string) error {
	var missing []string
	for _, p := range includePaths {
		out, err := g.run(ctx, "ls-tree", commitSHA, p)
		if err != nil {
			return err
		}
		if strings.TrimSpace(out) == "" {
			missing = append(missing, p)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("include_paths do not exist at commit %s: %s", shortSHA(commitSHA), strings.Join(missing, ", "))
	}
	return nil
}

// shortSHA abbreviates a commit SHA to 8 chars for log/error messages, tolerating
// user-supplied abbreviations shorter than that.
func shortSHA(sha string) string {
	return sha[:min(len(sha), 8)]
}
