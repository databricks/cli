package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetChangedFiles returns the files that headRef changes relative to baseRef.
//
// It uses git's three-dot form (baseRef...headRef), which diffs headRef against
// the merge base of the two refs rather than against the current tip of baseRef.
// This matters in CI: baseRef is the live tip of the target branch (e.g. the PR
// event's base.sha, which tracks origin/main). A two-dot "git diff baseRef
// headRef" would then also report every file that advanced on main after the
// branch was cut, none of which the PR actually touches — inflating the changed
// set and triggering test jobs (and integration tests) that the change doesn't
// require. Diffing against the merge base yields only the PR's own changes.
//
// "git diff baseRef...headRef" is equivalent to
// "git diff $(git merge-base baseRef headRef) headRef".
// See the <commit>...<commit> form at https://git-scm.com/docs/git-diff.
func GetChangedFiles(headRef, baseRef string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", baseRef+"..."+headRef)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get diff between %s and %s: %w", baseRef, headRef, err)
	}

	lines := strings.Split(string(output), "\n")

	// Drop the last line (always empty)
	if len(lines) > 0 {
		lines = lines[:len(lines)-1]
	}

	return lines, nil
}
