// Command fingerprint computes a content hash over a set of tracked files at a
// git commit. Two commits with the same fingerprint for a given input set have
// byte-identical inputs, so a test target that passed at one can be safely
// skipped at the other.
//
// This is the primitive behind result-reuse across PR iterations: unlike
// testmask, which diffs the cumulative PR range and so re-runs a target whenever
// the PR *ever* touched it, a content fingerprint is identical across PR
// iterations that leave a target's inputs unchanged — and it is force-push /
// rebase immune because it addresses content, not history. A target that passed
// at a fingerprint can be skipped at any later commit with the same fingerprint.
package main

import (
	"cmp"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"
	"strings"
)

// TreeEntry is one tracked file: its path and git blob object id (content hash).
type TreeEntry struct {
	Path string
	Blob string
}

// Fingerprint returns a deterministic hash over the given tree entries.
//
// It is order-independent (entries are sorted by path first) so it does not
// depend on git's listing order, and it binds each path to its blob id so both
// a content change (blob changes) and a rename (path changes) alter the result.
// The path is length-prefixed to make the encoding unambiguous, so no two
// distinct entry sets can collide by concatenation.
func Fingerprint(entries []TreeEntry) string {
	sorted := slices.Clone(entries)
	slices.SortFunc(sorted, func(a, b TreeEntry) int {
		return cmp.Compare(a.Path, b.Path)
	})

	h := sha256.New()
	for _, e := range sorted {
		// Length-prefix the path so "ab"+"c" and "a"+"bc" cannot hash alike.
		fmt.Fprintf(h, "%d\x00%s\x00%s\x00", len(e.Path), e.Path, e.Blob)
	}
	return hex.EncodeToString(h.Sum(nil))
}

// ParseLsTree parses the output of
//
//	git ls-tree -r --format='%(objectname) %(path)' <commit> [-- <paths>...]
//
// into tree entries. It is split from the git invocation so the parsing is
// unit-testable with literal strings, without a git repo (the same split
// testmask uses for GetChangedFiles vs GetTargets).
//
// Each line is "<blob-sha> <path>". A blank trailing line is ignored. Paths with
// spaces are preserved because only the first field is the blob id.
func ParseLsTree(out string) []TreeEntry {
	var entries []TreeEntry
	for line := range strings.SplitSeq(out, "\n") {
		if line == "" {
			continue
		}
		blob, path, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		entries = append(entries, TreeEntry{Path: path, Blob: blob})
	}
	return entries
}
