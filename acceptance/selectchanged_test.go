package acceptance_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
)

// Cloud PR runs set DATABRICKS_TEST_SELECT_CHANGED=N to run only the acceptance
// tests this branch touches, at most N of them, instead of the full suite.
const (
	SelectChangedEnvVar = "DATABRICKS_TEST_SELECT_CHANGED"

	// Cap for runs that need change detection without setting the env var: the subset
	// selector keeps changed tests on top of its hash-selected fraction, so a PR that
	// edits hundreds of test dirs must not turn the subset cells back into a full run.
	subsetChangedLimit = 50

	invariantConfigsPrefix = "acceptance/bundle/invariant/configs/"
	invariantDirPrefix     = "bundle/invariant/"
)

// getSelectChangedLimit returns the number of changed tests to select, or 0 when
// DATABRICKS_TEST_SELECT_CHANGED is unset (feature off).
func getSelectChangedLimit(t *testing.T) int {
	raw := os.Getenv(SelectChangedEnvVar)
	if raw == "" {
		return 0
	}

	limit, err := strconv.Atoi(raw)
	if err != nil || limit <= 0 {
		t.Fatalf("Invalid %s=%q, expected a positive integer", SelectChangedEnvVar, raw)
	}

	return limit
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

// selectChangedTests returns a map of test dir → extra env filters for the tests this
// branch changed. A nil filter slice means all variants of that dir run; a non-nil
// slice restricts to variants matching those filters (applied by the caller via
// checkEnvFilters in the variant loop).
//
// --merge-base diffs the working tree against the merge base of HEAD and
// origin/main. This covers committed, staged, and unstaged changes alike —
// the working tree reflects all three. Untracked files (not yet git-added)
// are not visible to git diff and will not be re-enabled until staged or
// committed. The three-dot form origin/main...HEAD only covers committed
// changes and misses unstaged edits, which breaks the "touch a config, run
// the test" local dev workflow (same reason lintdiff.py uses --merge-base).
func selectChangedTests(t *testing.T, testDirs map[string]bool, limit int) map[string][]string {
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

	changed, dropped := classifyChangedTests(strings.TrimSpace(string(out)), testDirs, limit)

	// Log the outcome up front: which tests the diff picked, and how many the limit
	// cut, so a CI run shows what it is about to cover without reading every skip line.
	names := make([]string, 0, len(changed))
	for dir, filters := range changed {
		if filters != nil {
			dir += "[" + strings.Join(filters, ",") + "]"
		}
		names = append(names, dir)
	}
	slices.Sort(names)
	t.Logf("Selected %d changed tests (limit=%d, %d not selected): %s", len(names), limit, dropped, strings.Join(names, " "))

	return changed
}

// classifyChangedTests maps `git diff --name-status` output to test dirs, keeping at
// most limit of them, in this order: added dirs, dirs with a changed fixture (script,
// test.toml, databricks.yml, ...), dirs where only generated files changed (out*), and
// finally moved dirs. It also returns how many changed dirs the limit dropped.
//
// Generated files rank below fixtures because a regenerated golden usually comes from a
// change elsewhere in the tree and lands on hundreds of dirs at once, which would
// otherwise fill the whole quota and crowd out the tests this branch actually edits.
//
// A changed invariant config (acceptance/bundle/invariant/configs/*.yml.tmpl)
// maps to all invariant subdirs with an INPUT_CONFIG= filter, so touching
// job.yml.tmpl re-enables all subdirs but only for their job.yml.tmpl variants.
func classifyChangedTests(diff string, testDirs map[string]bool, limit int) (map[string][]string, int) {
	// result accumulates dirs with their filters; added, fixture and moved record how
	// each dir changed, so the cap can rank them.
	// nil filter slice = all variants run; non-nil = restricted to those filters.
	result := map[string][]string{}
	added := map[string]bool{}
	fixture := map[string]bool{}
	moved := map[string]bool{}

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
						// The config is the fixture these dirs are generated from.
						fixture[dir] = true
					}
				}
			}
			continue
		}

		// test.toml and out.test.toml under the invariant tree regenerate
		// automatically when INPUT_CONFIG changes; ignore them so they don't
		// unlock all variants of every invariant subdir.
		if strings.HasPrefix(path, "acceptance/"+invariantDirPrefix) {
			if name := filepath.Base(path); name == "test.toml" || name == "out.test.toml" {
				continue
			}
		}

		dir := testDirForFile(path, testDirs)
		if dir == "" {
			continue
		}
		result[dir] = nil // nil = all variants; overrides any prior config-scoped filter
		// Everything starting with "out" is generated (output.txt, out.requests.txt,
		// out.test.toml); the rest is a fixture the test is defined by.
		if !strings.HasPrefix(filepath.Base(path), "out") {
			fixture[dir] = true
		}
		// The status of a dir's script file says how the dir itself changed:
		// A means brand new, R (Rnnn) means moved here from another path.
		if strings.HasSuffix(path, "/script") {
			switch {
			case status == "A":
				added[dir] = true
			case strings.HasPrefix(status, "R"):
				moved[dir] = true
			}
		}
	}

	var addedDirs, fixtureDirs, generatedDirs, movedDirs []string
	for dir := range result {
		switch {
		case added[dir]:
			addedDirs = append(addedDirs, dir)
		case moved[dir]:
			movedDirs = append(movedDirs, dir)
		case fixture[dir]:
			fixtureDirs = append(fixtureDirs, dir)
		default:
			generatedDirs = append(generatedDirs, dir)
		}
	}
	slices.Sort(addedDirs)
	slices.Sort(fixtureDirs)
	slices.Sort(generatedDirs)
	slices.Sort(movedDirs)

	selected := slices.Concat(addedDirs, fixtureDirs, generatedDirs, movedDirs)
	dropped := 0
	if len(selected) > limit {
		dropped = len(selected) - limit
		selected = selected[:limit]
	}

	out := make(map[string][]string, len(selected))
	for _, dir := range selected {
		out[dir] = result[dir]
	}
	return out, dropped
}
