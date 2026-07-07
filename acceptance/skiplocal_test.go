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

// resolveBaseRef returns the ref to diff against: origin/main when present, else main.
func resolveBaseRef() string {
	if git("rev-parse", "--verify", "origin/main") != "" {
		return "origin/main"
	}
	return "main"
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
// A renamed test dir is treated as modified, not added (git -M detects renames
// as a single R entry rather than add+delete). --merge-base folds the merge-base
// computation into the diff call itself, matching tools/lintdiff.py.
func selectChangedLocalTests(testDirs map[string]bool) map[string]bool {
	base := resolveBaseRef()

	diff := git("diff", "--name-status", "--merge-base", "-M", base)

	renamedInto := map[string]bool{}
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
		if strings.HasPrefix(status, "R") || strings.HasPrefix(status, "C") {
			renamedInto[dir] = true
		}
	}

	// Resolve the merge-base SHA for the batched ls-tree call below.
	mergeBase := git("merge-base", "HEAD", base)

	// Batch all script-existence checks into one ls-tree call.
	var scriptPaths []string
	for dir := range changed {
		scriptPaths = append(scriptPaths, "acceptance/"+dir+"/script")
	}
	slices.Sort(scriptPaths)
	existsAtBase := map[string]bool{}
	if mergeBase != "" && len(scriptPaths) > 0 {
		args := append([]string{"ls-tree", "--name-only", mergeBase}, scriptPaths...)
		for _, line := range strings.Split(git(args...), "\n") {
			if line != "" {
				existsAtBase[line] = true
			}
		}
	}

	var added, modified []string
	for dir := range changed {
		isNew := !renamedInto[dir] && !existsAtBase["acceptance/"+dir+"/script"]
		if isNew {
			added = append(added, dir)
		} else {
			modified = append(modified, dir)
		}
	}
	slices.Sort(added)
	slices.Sort(modified)

	selected := append(added, modified...)
	if len(selected) > maxChangedLocalTests {
		selected = selected[:maxChangedLocalTests]
	}

	result := make(map[string]bool, len(selected))
	for _, dir := range selected {
		result[dir] = true
	}
	return result
}
