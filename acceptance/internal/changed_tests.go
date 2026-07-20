package internal

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	invariantConfigsPrefix = "acceptance/bundle/invariant/configs/"
	invariantDirPrefix     = "bundle/invariant/"

	acceptancePrefix = "acceptance/"
	// binPrefix holds helper scripts placed on PATH for every test (print_requests.py,
	// contains.py, ...), so a change to any of them can affect any test.
	binPrefix = "acceptance/bin/"
)

// ChangedTest describes how one acceptance test dir was affected by a branch diff.
type ChangedTest struct {
	// Added is true when the dir is brand new (its "script" file was added on the branch).
	Added bool

	// VariantFilters restricts which env-matrix variants of the dir are relevant.
	// nil means all variants; a non-nil slice (e.g. "INPUT_CONFIG=job.yml.tmpl")
	// restricts to matching variants. Only invariant-config changes set this.
	VariantFilters []string
}

// ChangedTests returns the acceptance test dirs affected by the diff against the
// merge base with baseRef, mapped to how each was affected. testDirs is the set
// of known test dirs (relative to acceptance/).
//
// When headRef is empty, the diff is against the working tree: this covers
// committed, staged, and unstaged changes alike, which enables the local dev
// workflow of "touch a config, run the test, see it re-enabled" without
// committing (the same reason lintdiff.py uses --merge-base). Untracked files
// (not yet git-added) are not visible to git diff and will not be re-enabled
// until staged or committed.
//
// When headRef is non-empty, the diff is between the two refs (baseRef and
// headRef), independent of what is checked out. This is what CI and tests that
// replay a historical PR want: only committed changes on headRef, no dependence
// on the working tree. The three-dot form baseRef...headRef would only cover
// committed changes too, but --merge-base keeps the semantics identical to the
// working-tree case.
func ChangedTests(baseRef, headRef string, testDirs map[string]bool) (map[string]ChangedTest, error) {
	args := []string{"diff", "--name-status", "--merge-base", "-M", baseRef}
	if headRef != "" {
		args = append(args, headRef)
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("git diff against %s failed: %w", baseRef, err)
	}
	return ParseChangedTests(string(out), testDirs), nil
}

// ParseChangedTests turns the output of `git diff --name-status -M` into the set
// of affected acceptance test dirs. It is split from ChangedTests so the mapping
// logic can be unit-tested with literal diff strings, without a git repo.
//
// A rename record "R<score>\t<old>\t<new>" affects tests at both paths: the old
// path's removal and the new path's modification can each touch a different test
// or shared input. Each record is therefore expanded into one classification per
// affected path (a rename into a deletion of the old path plus a modification of
// the new) so a single classifier covers deletes, renames, and edits uniformly.
func ParseChangedTests(diffOutput string, testDirs map[string]bool) map[string]ChangedTest {
	result := map[string]ChangedTest{}

	for line := range strings.SplitSeq(strings.TrimSpace(diffOutput), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		status := fields[0]
		if strings.HasPrefix(status, "R") {
			// A rename removes the old path and modifies content at the new one.
			// The new path is "M", not "A": moving a script into a dir does not
			// make that dir a brand-new test.
			classifyPath("D", fields[1], testDirs, result)
			classifyPath("M", fields[2], testDirs, result)
			continue
		}
		classifyPath(status, fields[len(fields)-1], testDirs, result)
	}

	return result
}

// classifyPath records the test dirs affected by a single changed path with the
// given git status (A/M/D). It handles, in precedence order: invariant configs,
// bin/ helpers, shared inherited files, an owning test dir, and finally the
// subtree of any other shared input in a non-test dir.
func classifyPath(status, path string, testDirs map[string]bool, result map[string]ChangedTest) {
	// A changed invariant config re-enables all invariant subdirs with an
	// INPUT_CONFIG filter, unless a subdir was already unlocked by a non-config change.
	if strings.HasPrefix(path, invariantConfigsPrefix) {
		configName := path[len(invariantConfigsPrefix):]
		// Strip -init.sh / -cleanup.sh suffixes to get the base config name.
		if i := strings.Index(configName, "-"); i > 0 && strings.HasSuffix(configName, ".sh") {
			configName = configName[:i]
		}
		if !strings.HasSuffix(configName, ".yml.tmpl") {
			return
		}
		// A deleted config template has no variant left to run, so ignore it;
		// otherwise it would add an INPUT_CONFIG= filter matching nothing. A
		// deleted -init.sh/-cleanup.sh helper is not the template (configName was
		// stripped above), so its variant still exists and is re-enabled.
		if status == "D" && strings.HasSuffix(path, ".yml.tmpl") {
			return
		}
		for dir := range testDirs {
			if strings.HasPrefix(dir, invariantDirPrefix) {
				existing, ok := result[dir]
				if !ok || existing.VariantFilters != nil {
					existing.VariantFilters = append(existing.VariantFilters, "INPUT_CONFIG="+configName)
					result[dir] = existing
				}
			}
		}
		return
	}

	// bin/ holds helper scripts placed on PATH for every test, so a change to
	// any of them can affect any test: re-enable the whole suite.
	if strings.HasPrefix(path, binPrefix) {
		markSubtree("", testDirs, result)
		return
	}

	// A shared file (script.prepare/script.cleanup/test.toml) is concatenated or
	// inherited into every test at or below its dir, so it re-enables that whole
	// subtree. This runs before the single-dir mapping because such a file can
	// live inside a test dir that also has nested test dirs, which the single-dir
	// mapping would miss.
	if sharedTestFile(path) {
		markSubtree(containingDir(path), testDirs, result)
		return
	}

	dir := testDirForFile(path, testDirs)
	if dir == "" {
		// The file is not owned by any test dir. It is a shared input in a
		// non-test dir (a sourced _script, a fixture read via $TESTDIR/..), so
		// conservatively re-enable every test dir in its subtree. A stray file at
		// the acceptance root (no subtree dir) affects nothing identifiable and is
		// ignored.
		if base := containingDir(path); base != "" {
			markSubtree(base, testDirs, result)
		}
		return
	}
	// A direct change re-enables all variants, so clear any config-scoped filter
	// (nil VariantFilters = all variants). Added is sticky: once any path marks
	// the dir new it stays new, since paths for the same dir arrive in arbitrary
	// order. A script file with status A means the test dir is brand new.
	ct := result[dir]
	ct.VariantFilters = nil
	if status == "A" && strings.HasSuffix(path, "/script") {
		ct.Added = true
	}
	result[dir] = ct
}

// sharedTestFilenames are files that, when they live in an ancestor dir, affect
// every test below them: script.prepare / script.cleanup are concatenated into
// each descendant's script (see readMergedScriptContents in acceptance_test.go)
// and test.toml config is inherited by descendants.
var sharedTestFilenames = map[string]bool{
	"script.prepare": true,
	"script.cleanup": true,
	"test.toml":      true,
}

// sharedTestFile reports whether path is a shared file whose change propagates to
// every test in its subtree.
func sharedTestFile(path string) bool {
	return strings.HasPrefix(path, acceptancePrefix) && sharedTestFilenames[filepath.Base(path)]
}

// containingDir returns the dir of an acceptance/ path relative to acceptance/
// ("" for a file directly under acceptance/), or "" if path is outside acceptance/.
func containingDir(path string) string {
	if !strings.HasPrefix(path, acceptancePrefix) {
		return ""
	}
	dir := filepath.ToSlash(filepath.Dir(path[len(acceptancePrefix):]))
	if dir == "." {
		return ""
	}
	return dir
}

// markSubtree re-enables (with all variants) every test dir equal to or nested
// under base. An empty base matches every test dir. It preserves any existing
// Added flag and only clears VariantFilters, since a subtree-wide change affects
// all variants.
func markSubtree(base string, testDirs map[string]bool, result map[string]ChangedTest) {
	for dir := range testDirs {
		if base == "" || dir == base || strings.HasPrefix(dir, base+"/") {
			ct := result[dir]
			ct.VariantFilters = nil
			result[dir] = ct
		}
	}
}

// testDirForFile maps a repo-relative changed file (e.g. acceptance/bundle/foo/script)
// to its owning test dir relative to acceptance/ (e.g. bundle/foo), or "" if the file
// is outside acceptance/ or not under any known test dir.
func testDirForFile(repoRelPath string, testDirs map[string]bool) string {
	parts := strings.Split(filepath.ToSlash(repoRelPath), "/")
	if len(parts) < 2 || parts[0] != "acceptance" {
		return ""
	}
	// Longest ancestor first so nested tests map to the innermost test dir.
	for depth := len(parts); depth > 1; depth-- {
		candidate := strings.Join(parts[1:depth], "/")
		if testDirs[candidate] {
			return candidate
		}
	}
	return ""
}
