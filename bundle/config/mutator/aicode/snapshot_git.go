package aicode

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle/config"
)

// Local, no-network git packaging for a code_source block that pins a git ref.
// Ported from experimental/air/cmd (snapshot_git.go / snapshot_resolve.go /
// snapshot_package.go), trimmed to what a bundle deploy needs: resolve the ref to a
// local commit, then `git archive` that commit into an in-memory gzipped tarball the
// caller overlays on the sync root (same as the working-tree snapshot). The
// remote-fetch and provenance-sidecar paths are intentionally not ported — a ref must
// resolve to a commit already present locally.

// gitRepo runs git subcommands scoped to one repository via `git -C`. Arguments are
// passed as a slice, never a shell string, so a branch/commit value can't inject.
type gitRepo struct {
	path string
}

func newGitRepo(path string) gitRepo {
	return gitRepo{path: path}
}

// runBytes executes `git -C <path> <args...>` and returns raw stdout; a non-zero exit
// wraps stderr.
func (g gitRepo) runBytes(ctx context.Context, args ...string) ([]byte, error) {
	full := append([]string{"-C", g.path}, args...)
	cmd := exec.CommandContext(ctx, "git", full...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
		}
		return nil, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, msg)
	}
	return stdout.Bytes(), nil
}

func (g gitRepo) run(ctx context.Context, args ...string) (string, error) {
	out, err := g.runBytes(ctx, args...)
	return string(out), err
}

// isRepository reports whether the path is inside a git work tree. rev-parse (not a
// .git lookup) means a subdirectory of a repo counts — the common case when root_path
// is a subfolder of a monorepo.
func (g gitRepo) isRepository(ctx context.Context) bool {
	out, err := g.run(ctx, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return false
	}
	return strings.TrimSpace(out) == "true"
}

// hasUncommittedChanges reports whether there are staged or unstaged changes under the
// repo subtree. The `-- .` pathspec scopes the check so a subfolder snapshot considers
// only changes that could land in it.
func (g gitRepo) hasUncommittedChanges(ctx context.Context) (bool, error) {
	out, err := g.run(ctx, "status", "--porcelain", "--", ".")
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

// hasFilesAtCommit reports whether the commit has any files (blobs) under the given
// paths (whole tree when includePaths is empty). `-r` recurses so only blobs are
// listed, not tree entries — an empty result means there is nothing to package.
func (g gitRepo) hasFilesAtCommit(ctx context.Context, commitSHA string, includePaths []string) (bool, error) {
	args := append([]string{"ls-tree", "-r", "--name-only", commitSHA}, includePaths...)
	out, err := g.run(ctx, args...)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// validateIncludePathsExist checks that every include path exists at commitSHA. Without
// -d, `git ls-tree` reports both blobs and trees; empty output means the path is missing.
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
		return fmt.Errorf("code_source.include_paths do not exist at commit %s: %s", shortSHA(commitSHA), strings.Join(missing, ", "))
	}
	return nil
}

// shortSHA abbreviates a commit SHA to 8 chars for messages, tolerating a
// user-supplied abbreviation shorter than that.
func shortSHA(sha string) string {
	return sha[:min(len(sha), 8)]
}

// resolveGitCommit resolves a code_source.git ref to a commit SHA that exists locally,
// verifying include_paths exist at that commit. repoPath is the local code directory.
//
// git.commit pins a committed SHA; a dirty working tree is irrelevant (not archived).
// git.branch archives the branch's local HEAD, so a dirty tree is an error: the
// committed HEAD would not include the uncommitted changes.
func resolveGitCommit(ctx context.Context, repoPath string, git *config.CodeSourceGit, includePaths []string) (string, error) {
	repo := newGitRepo(repoPath)
	// Forward-slash the path in user-facing messages so output is identical across
	// operating systems (matches the repo's path-output convention).
	displayPath := filepath.ToSlash(repoPath)
	if !repo.isRepository(ctx) {
		return "", fmt.Errorf("code_source.git is set but %s is not a git repository", displayPath)
	}

	var commit string
	switch {
	case git.Commit != "":
		if !repo.commitExistsLocally(ctx, git.Commit) {
			return "", fmt.Errorf("commit %q does not exist locally; fetch it (e.g. `git fetch`) before deploying — the snapshot archives your local copy and does not fetch from a remote", git.Commit)
		}
		commit = git.Commit
	case git.Branch != "":
		dirty, err := repo.hasUncommittedChanges(ctx)
		if err != nil {
			return "", err
		}
		if dirty {
			return "", fmt.Errorf("uncommitted changes under %s would not be included: code_source.git.branch deploys the committed HEAD of %q. Commit your changes, or use git.commit to pin a specific revision", displayPath, git.Branch)
		}
		sha, err := repo.resolveLocalBranchSHA(ctx, git.Branch)
		if err != nil {
			return "", err
		}
		commit = sha
	default:
		// aicode.Validate guarantees exactly one of branch/commit is set.
		return "", errors.New("code_source.git requires either 'branch' or 'commit'")
	}

	if len(includePaths) > 0 {
		if err := repo.validateIncludePathsExist(ctx, commit, includePaths); err != nil {
			return "", err
		}
	}

	// Fail on an empty tree rather than package a codeless archive and deploy a job
	// with no code. Mirrors the working-tree path's len(files)==0 guard; the
	// include_paths existence check above does not catch an otherwise-empty root_path.
	hasFiles, err := repo.hasFilesAtCommit(ctx, commit, includePaths)
	if err != nil {
		return "", err
	}
	if !hasFiles {
		return "", fmt.Errorf("code_source.root_path has no files to package at commit %s", shortSHA(commit))
	}

	return commit, nil
}

// buildGitArchive produces a gzipped tarball of commitSHA via `git archive`, with every
// entry prefixed by prefix/ (so it expands to <prefix>/..., matching the runtime's
// /databricks/code_source/<dir> extraction contract). When includePaths is set only
// those paths are archived. It returns the archive bytes and their SHA-256 hex digest
// for content-addressed naming. The commit is deterministic, so an identical
// (commit, include_paths) yields an identical name and skips re-upload — but git's gzip
// output is not guaranteed byte-identical across git versions, so the name is stable
// only within a git version (a one-time re-upload after a git upgrade is acceptable).
func buildGitArchive(ctx context.Context, repoPath, commitSHA, prefix string, includePaths []string) ([]byte, string, error) {
	args := []string{"archive", "--format=tar.gz", "--prefix=" + prefix + "/", commitSHA}
	args = append(args, includePaths...)
	out, err := newGitRepo(repoPath).runBytes(ctx, args...)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create git archive: %w", err)
	}
	sum := sha256.Sum256(out)
	return out, hex.EncodeToString(sum[:]), nil
}
