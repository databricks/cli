package acceptance_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
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
)

// validateSkipLocal fails the test immediately on an unsupported
// DATABRICKS_TEST_SKIPLOCAL value. Empty or absent means disabled.
func validateSkipLocal(t *testing.T) {
	t.Helper()
	switch os.Getenv(SkipLocalEnvVar) {
	case "", SkipLocalAll, SkipLocalWithChanged:
	default:
		t.Fatalf("Unsupported %s=%q, expected %q or %q", SkipLocalEnvVar, os.Getenv(SkipLocalEnvVar), SkipLocalAll, SkipLocalWithChanged)
	}
}

// git runs a git command and returns trimmed stdout.
// A non-zero exit yields an empty string.
func git(args ...string) string {
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
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

// selectChangedLocalTests returns the set of test dirs to re-enable under
// SkipLocalWithChanged: those added or changed on this branch vs the merge base,
// added-first and capped at maxChangedLocalTests.
//
// A test dir is "added" when its script file has status A in the diff (didn't
// exist at the merge base). Renames (R) are treated as modified, not added.
// The three-dot form origin/main...HEAD diffs HEAD against the merge base of
// the two refs in one call (see tools/testmask/git.go for the rationale).
func selectChangedLocalTests(testDirs map[string]bool) map[string]bool {
	diff := git("diff", "--name-status", "-M", "origin/main...HEAD")

	added := map[string]bool{}
	changed := map[string]bool{}
	for _, line := range strings.Split(diff, "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		status := fields[0]
		path := fields[len(fields)-1]
		dir := testDirForFile(path, testDirs)
		if dir == "" {
			continue
		}
		changed[dir] = true
		// A script file with status A means the test dir is brand new.
		// Renames (R) land here as the destination path but are not "added".
		if status == "A" && strings.HasSuffix(path, "/script") {
			added[dir] = true
		}
	}

	var addedDirs, modifiedDirs []string
	for dir := range changed {
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

	result := make(map[string]bool, len(selected))
	for _, dir := range selected {
		result[dir] = true
	}
	return result
}
