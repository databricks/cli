package acceptance_test

import (
	"slices"

	"github.com/databricks/cli/acceptance/internal"
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

// selectChangedLocalTests returns a map of test dir → extra env filters for
// re-enabling under SkipLocalWithChanged. A nil filter slice means all variants
// of that dir run; a non-nil slice restricts to variants matching those filters
// (applied by the caller via checkEnvFilters in the variant loop).
// Added dirs come before modified ones; the total is capped at maxChangedLocalTests.
//
// Detection (which dirs the branch's diff against origin/main affects, and how)
// lives in internal.ChangedTests; this wrapper applies the skiplocal-specific
// ordering and cap. On error (e.g. origin/main unreachable) it returns nil, so a
// local run without origin/main degrades to skipping all Local tests.
func selectChangedLocalTests(testDirs map[string]bool) map[string][]string {
	// Empty headRef: diff against the working tree so uncommitted local edits
	// re-enable their tests without needing a commit.
	changed, err := internal.ChangedTests("origin/main", "", testDirs)
	if err != nil {
		return nil
	}

	var addedDirs, modifiedDirs []string
	for dir, ct := range changed {
		if ct.Added {
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

	out := make(map[string][]string, len(selected))
	for _, dir := range selected {
		out[dir] = changed[dir].VariantFilters
	}
	return out
}
