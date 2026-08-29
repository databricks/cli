package aircmd

import (
	"context"
	"errors"
	"fmt"
)

// This file ports the mode/ref resolution from the Python CLI's cli_entrypoint
// snapshot block (the if/elif at lines ~1541–1722), local-only. The remote-fetch
// branches are dropped: a git ref must resolve to a commit already present
// locally (git.remote is rejected at validation — see gitRef.validate).

// snapshotMode is how the snapshot tarball is produced.
type snapshotMode int

const (
	// modeGitArchive packages a pinned commit via `git archive`. The commit is
	// deterministic, so the tarball is cacheable by (commit, include_paths).
	modeGitArchive snapshotMode = iota
	// modePlainTar packages the working tree (including uncommitted changes) via
	// `tar`. Not cacheable — working-tree content isn't pinned to a SHA.
	modePlainTar
)

// snapshotPlan is the outcome of resolving how to package a snapshot: the mode,
// the commit SHA to archive (git_archive only; empty for plain_tar), and whether
// the working tree under the snapshot root has uncommitted changes.
type snapshotPlan struct {
	mode         snapshotMode
	commitSHA    string
	hasUncommit  bool
	isGitRepo    bool
	includePaths []string
}

// resolveSnapshotPlan decides how to package the snapshot (local-only):
//   - git.commit  → pin the SHA (must exist locally) → git_archive.
//   - git.branch  → the branch's local HEAD SHA → git_archive.
//   - no ref / non-git dir → the working tree → plain_tar (no caching).
//
// The dirty check runs at most once (git status is O(working tree)) and is threaded
// into the plan. Dirty + git.branch is an error: the committed HEAD wouldn't include
// the uncommitted changes.
func resolveSnapshotPlan(ctx context.Context, git gitRepo, ref *gitRef, includePaths []string) (snapshotPlan, error) {
	plan := snapshotPlan{includePaths: includePaths}
	plan.isGitRepo = git.isRepository(ctx)

	// Detect uncommitted changes once. When include_paths is set, only changes
	// under those paths can land in the snapshot, so scope the check to them —
	// both more correct and cheaper than scanning the whole repo.
	if plan.isGitRepo {
		var err error
		if len(includePaths) > 0 {
			plan.hasUncommit, err = git.hasUncommittedChangesInPaths(ctx, includePaths)
		} else {
			plan.hasUncommit, err = git.hasUncommittedChanges(ctx)
		}
		if err != nil {
			return snapshotPlan{}, err
		}
	}

	// Non-git directory: plain tar, no ref allowed. gitRef.validate already rejects
	// git.* on a non-git dir at load time, but guard here too since this function
	// is the single decision point.
	if !plan.isGitRepo {
		if ref != nil {
			return snapshotPlan{}, fmt.Errorf("git.* is set but %s is not a git repository", git.path)
		}
		plan.mode = modePlainTar
		return plan, nil
	}

	// git repo, no ref: package the working tree as plain tar (uncommitted changes
	// included). Provenance is captured separately via the git_state sidecar.
	if ref == nil {
		plan.mode = modePlainTar
		return plan, nil
	}

	switch {
	case ref.Commit != nil:
		// git.commit pins a committed SHA; local uncommitted changes are irrelevant
		// and won't be included. The commit must exist locally — no remote fetch.
		commit := *ref.Commit
		if !git.commitExistsLocally(ctx, commit) {
			return snapshotPlan{}, fmt.Errorf("commit %q does not exist locally; fetch it (e.g. `git fetch`) before submitting — the snapshot archives your local copy and does not fetch from a remote", commit)
		}
		plan.mode = modeGitArchive
		plan.commitSHA = commit

	case ref.Branch != nil:
		// git.branch deploys the branch's local HEAD. A dirty tree here is an error:
		// the committed HEAD wouldn't include the uncommitted changes.
		if plan.hasUncommit {
			return snapshotPlan{}, fmt.Errorf("uncommitted changes under %s would not be included: git.branch deploys the committed HEAD of %q. Commit your changes, or use git.commit to pin a specific revision", git.path, *ref.Branch)
		}
		sha, err := git.resolveLocalBranchSHA(ctx, *ref.Branch)
		if err != nil {
			return snapshotPlan{}, err
		}
		plan.mode = modeGitArchive
		plan.commitSHA = sha

	default:
		// gitRef.validate guarantees exactly one of branch/commit is set.
		return snapshotPlan{}, errors.New("git: must specify either 'branch' or 'commit'")
	}

	// For git_archive with include_paths, verify each path exists at the resolved
	// commit so a typo fails fast rather than producing an empty subtree.
	if len(includePaths) > 0 {
		if err := git.validateIncludePathsExist(ctx, plan.commitSHA, includePaths); err != nil {
			return snapshotPlan{}, err
		}
	}

	return plan, nil
}
