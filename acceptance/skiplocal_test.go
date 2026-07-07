package acceptance_test

import (
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
)

// DATABRICKS_TEST_SKIPLOCAL controls skipping of Local acceptance tests.
const (
	SkipLocalEnvVar = "DATABRICKS_TEST_SKIPLOCAL"

	// SkipLocalAll skips every test with Local = true.
	SkipLocalAll = "true"
	// SkipLocalWithChanged skips Local tests except those added or changed on this
	// branch (relative to the merge base with origin/main), so a cloud run still
	// exercises the tests this branch touches.
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
// of that dir run; a non-nil slice restricts to variants matching those filters.
// Added dirs come before modified ones; the total is capped at maxChangedLocalTests.
//
// Invariant configs (acceptance/bundle/invariant/configs/*.yml.tmpl) each feed
// every invariant subdir. A changed config re-enables all invariant subdirs but
// only for variants where INPUT_CONFIG matches the changed file — so touching
// job.yml.tmpl runs only the job.yml.tmpl variants, not all ~40 configs.
//
// --merge-base diffs the working tree against the merge base of HEAD and
// origin/main, so uncommitted edits are included. The three-dot form
// origin/main...HEAD would only cover committed changes and would miss a file
// touched but not yet committed, which breaks the "touch a config, run the
// test" local dev workflow (same reason lintdiff.py uses --merge-base).
func selectChangedLocalTests(testDirs map[string]bool) map[string][]string {
	out, _ := exec.Command("git", "diff", "--name-status", "--merge-base", "-M", "origin/main").Output()
	diff := strings.TrimSpace(string(out))

	// result accumulates dirs with their filters; added tracks brand-new dirs.
	// nil filter slice = all variants run; non-nil = restricted to those filters.
	result := map[string][]string{}
	added := map[string]bool{}

	addDir := func(dir string, filter string) {
		if filter == "" {
			result[dir] = nil // non-config change → run all variants
			return
		}
		// Config-specific change: restrict to this INPUT_CONFIG, unless the dir
		// was already unlocked for all variants by a non-config change.
		if existing, ok := result[dir]; !ok || existing != nil {
			result[dir] = append(result[dir], "INPUT_CONFIG="+filter)
		}
	}

	for _, line := range strings.Split(diff, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		status := fields[0]
		path := fields[len(fields)-1]

		// A changed invariant config re-enables all invariant subdirs, restricted
		// to only the INPUT_CONFIG variant matching the changed config file.
		if strings.HasPrefix(path, invariantConfigsPrefix) {
			configName := path[len(invariantConfigsPrefix):]
			// Strip -init.sh / -cleanup.sh suffixes to get the base config name.
			if i := strings.Index(configName, "-"); i > 0 && strings.HasSuffix(configName, ".sh") {
				configName = configName[:i]
			}
			if strings.HasSuffix(configName, ".yml.tmpl") {
				for dir := range testDirs {
					if strings.HasPrefix(dir, invariantDirPrefix) {
						addDir(dir, configName)
					}
				}
			}
			continue
		}

		dir := testDirForFile(path, testDirs)
		if dir == "" {
			continue
		}
		addDir(dir, "")
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
