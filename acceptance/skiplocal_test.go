package acceptance_test

import (
	"errors"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// DATABRICKS_TEST_SKIPLOCAL skips acceptance tests on cloud runs that already have
// testserver coverage. Every acceptance test runs locally, so this skips them all
// unless withchanged re-enables tests touched on the branch.
const (
	SkipLocalEnvVar = "DATABRICKS_TEST_SKIPLOCAL"

	// SkipLocalAll skips every acceptance test.
	SkipLocalAll = "true"
	// SkipLocalWithChanged skips acceptance tests except those added or changed on
	// this branch (relative to the merge base with origin/main), so a cloud run
	// still exercises the tests this branch touches.
	SkipLocalWithChanged = "withchanged"

	// maxChangedLocalTests caps how many changed tests SkipLocalWithChanged re-enables,
	// keeping the cloud run bounded. Added tests are preferred over modified ones.
	maxChangedLocalTests = 50

	invariantConfigsPrefix = "acceptance/bundle/invariant/configs/"
	invariantDirPrefix     = "bundle/invariant/"
)

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

// selectChangedLocalTests returns a map of test dir → extra env filters for
// re-enabling under SkipLocalWithChanged. A nil filter slice means all variants
// of that dir run; a non-nil slice restricts to variants matching those filters
// (applied by the caller via checkEnvFilters in the variant loop).
// Added dirs come before modified ones; the total is capped at maxChangedLocalTests.
//
// A changed invariant config (acceptance/bundle/invariant/configs/*.yml.tmpl)
// maps to all invariant subdirs with an INPUT_CONFIG= filter, so touching
// job.yml.tmpl re-enables all subdirs but only for their job.yml.tmpl variants.
//
// --merge-base diffs the working tree against the merge base of HEAD and
// origin/main. This covers committed, staged, and unstaged changes alike —
// the working tree reflects all three. Untracked files (not yet git-added)
// are not visible to git diff and will not be re-enabled until staged or
// committed. The three-dot form origin/main...HEAD only covers committed
// changes and misses unstaged edits, which breaks the "touch a config, run
// the test" local dev workflow (same reason lintdiff.py uses --merge-base).
func selectChangedLocalTests(t *testing.T, testDirs map[string]bool) map[string][]string {
	out, err := exec.Command("git", "diff", "--name-status", "--merge-base", "-M", "origin/main").Output()
	if err != nil {
		// A failed diff (most commonly a missing origin/main in a shallow CI
		// checkout) must not be silently treated as "nothing changed": that
		// disables change detection and lets newly added tests skip. Fail loudly.
		// Every caller (push.yml PR cells, integration runs) now fetches origin/main.
		stderr := ""
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			stderr = strings.TrimSpace(string(exitErr.Stderr))
		}
		t.Fatalf("git diff --merge-base origin/main failed: %v\n%s", err, stderr)
	}
	diff := strings.TrimSpace(string(out))

	// result accumulates dirs with their filters; added tracks brand-new dirs.
	// nil filter slice = all variants run; non-nil = restricted to those filters.
	result := map[string][]string{}
	added := map[string]bool{}

	for line := range strings.SplitSeq(diff, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		status := fields[0]
		path := fields[len(fields)-1]

		// A changed invariant config re-enables all invariant subdirs with an
		// INPUT_CONFIG filter, unless a subdir was already unlocked by a non-config change.
		if strings.HasPrefix(path, invariantConfigsPrefix) {
			configName := path[len(invariantConfigsPrefix):]
			// Strip -init.sh / -cleanup.sh suffixes to get the base config name.
			if i := strings.Index(configName, "-"); i > 0 && strings.HasSuffix(configName, ".sh") {
				configName = configName[:i]
			}
			if strings.HasSuffix(configName, ".yml.tmpl") {
				for dir := range testDirs {
					if strings.HasPrefix(dir, invariantDirPrefix) {
						if existing, ok := result[dir]; !ok || existing != nil {
							result[dir] = append(result[dir], "INPUT_CONFIG="+configName)
						}
					}
				}
			}
			continue
		}

		dir := testDirForFile(path, testDirs)
		if dir == "" {
			continue
		}
		result[dir] = nil // nil = all variants; overrides any prior config-scoped filter
		// A script file with status A means the test dir is brand new.
		// Renames (R) land here as the destination path but are not "added".
		if status == "A" && strings.HasSuffix(path, "/script") {
			added[dir] = true
		}
	}

	var addedDirs, modifiedDirs []string
	for dir := range result {
		if added[dir] {
			addedDirs = append(addedDirs, dir)
		} else {
			modifiedDirs = append(modifiedDirs, dir)
		}
	}
	slices.Sort(addedDirs)
	slices.Sort(modifiedDirs)

	selected := append(addedDirs, modifiedDirs...)
	if len(selected) > maxChangedLocalTests {
		selected = selected[:maxChangedLocalTests]
	}

	out2 := make(map[string][]string, len(selected))
	for _, dir := range selected {
		out2[dir] = result[dir]
	}
	return out2
}
