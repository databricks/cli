package aicode

import "context"

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

// snapshotPlan is the outcome of resolving how to package a snapshot: the mode and
// the commit SHA to archive (git_archive only; empty for plain_tar). includePaths is
// carried through to packaging and the cache key.
//
// The plan mirrors PR #5897's richer type so this package can later grow git-ref and
// include_paths selection once the ai_runtime_task schema can carry them; for now the
// DABs code_source_path is a plain directory, so the decision reduces to
// "committed git repo → archive HEAD, otherwise tar the working tree".
type snapshotPlan struct {
	mode         snapshotMode
	commitSHA    string
	includePaths []string
}

// resolveSnapshotPlan decides how to package a local code_source directory:
//   - a git work tree with no uncommitted changes → git_archive of HEAD (cacheable);
//   - otherwise (non-git dir, or a dirty tree) → plain_tar of the working tree.
//
// Archiving HEAD when clean lets unchanged code hit the upload cache across deploys;
// falling back to plain_tar when dirty ensures local edits are actually deployed
// rather than silently dropped.
func resolveSnapshotPlan(ctx context.Context, git gitRepo) (snapshotPlan, error) {
	if !git.isRepository(ctx) {
		return snapshotPlan{mode: modePlainTar}, nil
	}

	dirty, err := git.hasUncommittedChanges(ctx)
	if err != nil {
		return snapshotPlan{}, err
	}
	if dirty {
		return snapshotPlan{mode: modePlainTar}, nil
	}

	sha, err := git.headSHA(ctx)
	if err != nil {
		return snapshotPlan{}, err
	}
	return snapshotPlan{mode: modeGitArchive, commitSHA: sha}, nil
}
