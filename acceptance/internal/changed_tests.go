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
// Each diff line is "<status>\t<path>" (or "<status>\t<old>\t<new>" for renames);
// the last field is the current path. A changed invariant config
// (acceptance/bundle/invariant/configs/*.yml.tmpl) maps to all invariant subdirs
// with an INPUT_CONFIG= filter, so touching job.yml.tmpl re-enables all subdirs
// but only for their job.yml.tmpl variants. A direct change to a subdir takes
// precedence and re-enables all of its variants.
func ParseChangedTests(diffOutput string, testDirs map[string]bool) map[string]ChangedTest {
	result := map[string]ChangedTest{}

	for line := range strings.SplitSeq(strings.TrimSpace(diffOutput), "\n") {
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
						existing, ok := result[dir]
						if !ok || existing.VariantFilters != nil {
							existing.VariantFilters = append(existing.VariantFilters, "INPUT_CONFIG="+configName)
							result[dir] = existing
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
		// nil VariantFilters = all variants; overrides any prior config-scoped filter.
		// A script file with status A means the test dir is brand new. Renames (R)
		// land here as the destination path but are not "added".
		result[dir] = ChangedTest{
			Added: status == "A" && strings.HasSuffix(path, "/script"),
		}
	}

	return result
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
