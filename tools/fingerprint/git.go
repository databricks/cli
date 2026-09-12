package main

import (
	"fmt"
	"os/exec"
)

// TreeEntriesAt lists the tracked files matching pathspecs at the given commit,
// with their blob ids, via `git ls-tree`.
//
// It uses ls-tree (a snapshot of one commit) rather than a diff so the
// fingerprint is absolute: it depends only on the content at `commit`, not on
// any base ref. Blob ids come straight from git's index, so no file is read.
func TreeEntriesAt(commit string, pathspecs []string) ([]TreeEntry, error) {
	args := []string{"ls-tree", "-r", "--format=%(objectname) %(path)", commit}
	if len(pathspecs) > 0 {
		args = append(args, "--")
		args = append(args, pathspecs...)
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-tree at %s failed: %w", commit, err)
	}
	return ParseLsTree(string(out)), nil
}
