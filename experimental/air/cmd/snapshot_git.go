package aircmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Local, no-network git introspection and the git-state provenance sidecar, ported
// from the Python CLI's cli/utils/git_state.py. The remote-fetch helpers
// (fetch_branch_sha, remote detection, partial clone) are deliberately not ported:
// the snapshot archives the local copy only, so a ref must resolve to a local commit.

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
	out, err := g.runBytes(ctx, args...)
	return string(out), err
}

// runBytes is run returning raw stdout bytes, for the dirty-diff capture which needs
// exact bytes and a size measurement.
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

// currentBranch returns the branch name, or "" for a detached HEAD or on error.
func (g gitRepo) currentBranch(ctx context.Context) string {
	out, err := g.run(ctx, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	branch := strings.TrimSpace(out)
	if branch == "HEAD" {
		return ""
	}
	return branch
}

// remoteURL returns the URL of the named remote, or "" if it has none.
func (g gitRepo) remoteURL(ctx context.Context, remoteName string) string {
	out, err := g.run(ctx, "remote", "get-url", remoteName)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// mergeBaseWithUpstream resolves the merge-base of HEAD and a likely upstream ref,
// trying <remote>/HEAD, /main, then /master. It reads only local remote-tracking
// refs (no fetch), returning "" if none resolve.
func (g gitRepo) mergeBaseWithUpstream(ctx context.Context, remoteName string) string {
	for _, ref := range []string{remoteName + "/HEAD", remoteName + "/main", remoteName + "/master"} {
		out, err := g.run(ctx, "merge-base", "HEAD", ref)
		if err != nil {
			continue
		}
		if base := strings.TrimSpace(out); base != "" {
			return base
		}
	}
	return ""
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

// --- git-state provenance sidecar (git_state.json + git_diff.patch) ---
//
// The backend reads git_state.json next to the tarball to tag the MLflow run with
// base/tip/dirty provenance, and logs git_diff.patch when the tree was dirty.
// Producing the sidecar is best-effort: callers warn and continue, never fail submit.

// snapshotStateSchemaVersion is the git_state.json schema version. Bump only in
// coordination with the backend reader.
const snapshotStateSchemaVersion = 1

// gitStateName and gitDiffName are the git provenance sidecar basenames, uploaded
// next to the code snapshot for human/agent inspection of what was submitted.
const (
	gitStateName = "git_state.json"
	gitDiffName  = "git_diff.patch"
)

// defaultRemoteName is the remote consulted for merge-base and repo URL (local refs
// only — the remote-fetch path is gone).
const defaultRemoteName = "origin"

// dirtyDiffSizeCapBytes caps the git_diff.patch sidecar; a larger diff records
// size_exceeded and is skipped to keep the upload small.
const dirtyDiffSizeCapBytes = 1024 * 1024

// dirtyDiffTimeout bounds `git diff HEAD` so provenance never delays submission.
const dirtyDiffTimeout = 5 * time.Second

// packaging_mode values: how the uploaded tarball was produced.
const (
	packagingModeGitArchive = "git_archive"
	packagingModePlainTar   = "plain_tar"
)

// diff_status values recorded in the sidecar.
const (
	diffStatusClean        = "clean"
	diffStatusCaptured     = "captured"
	diffStatusSizeExceeded = "size_exceeded"
	diffStatusTimeout      = "timeout"
)

// gitStateSidecar is the git_state.json record. Field names and the null-for-absent
// encoding match the Python source, so nullable fields are *string (absent → null).
type gitStateSidecar struct {
	SchemaVersion  int     `json:"schema_version"`
	PackagingMode  string  `json:"packaging_mode"`
	BaseCommit     *string `json:"base_commit"`
	TipCommit      *string `json:"tip_commit"`
	Branch         *string `json:"branch"`
	RepoURL        *string `json:"repo_url"`
	Dirty          bool    `json:"dirty"`
	DiffStatus     string  `json:"diff_status"`
	DiffPath       *string `json:"diff_path"`
	GeneratedAtUTC string  `json:"generated_at_utc"`
}

// nilIfEmpty maps "" to nil so an absent value serializes as JSON null.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// buildGitStateSidecar gathers git provenance. pinnedTip overrides the HEAD-derived
// tip for git_archive (the tarball reflects that commit, not HEAD); pass "" for
// plain_tar. Metadata is best-effort — unavailable fields become null.
func buildGitStateSidecar(ctx context.Context, git gitRepo, packagingMode, pinnedTip string, dirty bool, now time.Time) (gitStateSidecar, error) {
	tip := pinnedTip
	if tip == "" {
		head, err := git.headSHA(ctx)
		if err != nil {
			return gitStateSidecar{}, err
		}
		tip = head
	}

	return gitStateSidecar{
		SchemaVersion:  snapshotStateSchemaVersion,
		PackagingMode:  packagingMode,
		BaseCommit:     nilIfEmpty(git.mergeBaseWithUpstream(ctx, defaultRemoteName)),
		TipCommit:      nilIfEmpty(tip),
		Branch:         nilIfEmpty(git.currentBranch(ctx)),
		RepoURL:        nilIfEmpty(git.remoteURL(ctx, defaultRemoteName)),
		Dirty:          dirty,
		DiffStatus:     diffStatusClean,
		DiffPath:       nil,
		GeneratedAtUTC: now.UTC().Format("2006-01-02T15:04:05.000000") + "Z",
	}, nil
}

// marshal renders the sidecar as indented JSON (matching Python's json.dump indent=2).
func (s gitStateSidecar) marshal() ([]byte, error) {
	return json.MarshalIndent(s, "", "  ")
}

// captureDirtyDiff runs `git diff HEAD` over the repo subtree, returning a diff_status
// and the diff bytes (non-nil only when captured): clean (no changes or diff failed),
// captured (under the cap), size_exceeded, or timeout.
func captureDirtyDiff(ctx context.Context, git gitRepo, includePaths []string, sizeCapBytes int, timeout time.Duration) (string, []byte) {
	diffCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pathspecs := includePaths
	if len(pathspecs) == 0 {
		pathspecs = []string{"."}
	}
	args := append([]string{"diff", "HEAD", "--"}, pathspecs...)
	out, err := git.runBytes(diffCtx, args...)
	if err != nil {
		if errors.Is(diffCtx.Err(), context.DeadlineExceeded) {
			return diffStatusTimeout, nil
		}
		return diffStatusClean, nil
	}
	if len(out) == 0 {
		return diffStatusClean, nil
	}
	if len(out) > sizeCapBytes {
		return diffStatusSizeExceeded, nil
	}
	return diffStatusCaptured, out
}
