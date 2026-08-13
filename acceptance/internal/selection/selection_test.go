package selection_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/databricks/cli/acceptance/internal/selection"
)

var testDirs = map[string]bool{
	"bundle/added":           true,
	"bundle/modified":        true,
	"bundle/regenerated":     true,
	"bundle/moved":           true,
	"bundle/untouched":       true,
	"bundle/invariant/jobs":  true,
	"bundle/invariant/apps":  true,
	"cmd/sync/nested":        true,
	"cmd/sync/nested/deeper": true,
}

// diffLines joins name-status records the way `git diff --name-status` prints them.
func diffLines(lines ...string) string {
	return strings.Join(lines, "\n")
}

func TestFromDiffStatuses(t *testing.T) {
	diff := diffLines(
		"A\tacceptance/bundle/added/script",
		"M\tacceptance/bundle/modified/databricks.yml",
		"R100\tacceptance/bundle/old/script\tacceptance/bundle/moved/script",
		"M\tlibs/dyn/value.go",
	)
	result := selection.FromDiff(diff, testDirs, 10)
	assert.Zero(t, result.Dropped)
	assert.Equal(t, map[string][]string{
		"bundle/added":    nil,
		"bundle/modified": nil,
		"bundle/moved":    nil,
	}, result.Tests())
}

func TestFromDiffPriority(t *testing.T) {
	diff := diffLines(
		"R090\tacceptance/bundle/old/script\tacceptance/bundle/moved/script",
		"M\tacceptance/bundle/regenerated/output.txt",
		"M\tacceptance/bundle/modified/script",
		"A\tacceptance/bundle/added/script",
	)
	// The cap keeps added first, then a changed fixture, then a moved dir, and last a dir
	// where only the golden output was regenerated.
	scored := []string{"bundle/added", "bundle/modified", "bundle/moved", "bundle/regenerated"}
	for limit := 1; limit <= len(scored); limit++ {
		result := selection.FromDiff(diff, testDirs, limit)
		assert.Len(t, result.Selected, limit)
		assert.Equal(t, len(scored)-limit, result.Dropped, "limit=%d", limit)
		for _, dir := range scored[:limit] {
			assert.Contains(t, result.Tests(), dir, "limit=%d", limit)
		}
	}
}

func TestFromDiffScores(t *testing.T) {
	diff := diffLines(
		"A\tacceptance/bundle/added/script",
		"M\tacceptance/bundle/modified/script",
		"R090\tacceptance/bundle/old/script\tacceptance/bundle/moved/script",
		"M\tacceptance/bundle/regenerated/output.txt",
	)
	result := selection.FromDiff(diff, testDirs, 10)
	scores := map[string]int{}
	for _, test := range result.Selected {
		scores[test.Dir] = test.Score
	}
	assert.Equal(t, map[string]int{
		"bundle/added":       10,
		"bundle/modified":    5,
		"bundle/moved":       2,
		"bundle/regenerated": -1,
	}, scores)
}

func TestFromDiffFixtureBeatsOutputInSameDir(t *testing.T) {
	// A dir with both a fixture and an output change ranks as a fixture change.
	diff := diffLines(
		"M\tacceptance/bundle/modified/output.txt",
		"M\tacceptance/bundle/modified/databricks.yml",
		"M\tacceptance/bundle/regenerated/out.requests.txt",
	)
	result := selection.FromDiff(diff, testDirs, 1)
	assert.Equal(t, map[string][]string{"bundle/modified": nil}, result.Tests())
	assert.Equal(t, 1, result.Dropped)
}

func TestFromDiffNestedFixture(t *testing.T) {
	// "out" is matched against the path relative to the test dir, so a file in a
	// subdirectory is a fixture even when its own name starts with "out".
	diff := diffLines(
		"M\tacceptance/bundle/modified/subdir/outer.py",
		"M\tacceptance/bundle/regenerated/output.txt",
	)
	result := selection.FromDiff(diff, testDirs, 1)
	assert.Equal(t, map[string][]string{"bundle/modified": nil}, result.Tests())
	assert.Equal(t, 1, result.Dropped)
}

func TestFromDiffInvariantConfigRanksAsFixture(t *testing.T) {
	// The invariant config is the fixture its dirs are generated from, so it outranks
	// a dir whose output was regenerated.
	diff := diffLines(
		"M\tacceptance/bundle/regenerated/output.txt",
		"M\tacceptance/bundle/invariant/configs/job.yml.tmpl",
	)
	result := selection.FromDiff(diff, testDirs, 2)
	assert.NotContains(t, result.Tests(), "bundle/regenerated")
	assert.Equal(t, 1, result.Dropped)
}

func TestFromDiffNestedDir(t *testing.T) {
	// A file maps to the innermost test dir that owns it.
	diff := diffLines("M\tacceptance/cmd/sync/nested/deeper/script")
	result := selection.FromDiff(diff, testDirs, 10)
	assert.Equal(t, map[string][]string{"cmd/sync/nested/deeper": nil}, result.Tests())
}

func TestFromDiffInvariantConfig(t *testing.T) {
	// A changed invariant config re-enables every invariant dir, restricted to that config.
	diff := diffLines("M\tacceptance/bundle/invariant/configs/job.yml.tmpl")
	result := selection.FromDiff(diff, testDirs, 10)
	assert.Equal(t, map[string][]string{
		"bundle/invariant/jobs": {"INPUT_CONFIG=job.yml.tmpl"},
		"bundle/invariant/apps": {"INPUT_CONFIG=job.yml.tmpl"},
	}, result.Tests())
}

func TestFromDiffInvariantConfigAndDir(t *testing.T) {
	// A non-config change to an invariant dir unlocks all of its variants; the
	// regenerated test.toml files are ignored.
	diff := diffLines(
		"M\tacceptance/bundle/invariant/configs/job.yml.tmpl",
		"M\tacceptance/bundle/invariant/jobs/script",
		"M\tacceptance/bundle/invariant/apps/out.test.toml",
	)
	result := selection.FromDiff(diff, testDirs, 10)
	assert.Equal(t, map[string][]string{
		"bundle/invariant/jobs": nil,
		"bundle/invariant/apps": {"INPUT_CONFIG=job.yml.tmpl"},
	}, result.Tests())
}

func TestFromDiffEmptyDiff(t *testing.T) {
	result := selection.FromDiff("", testDirs, 10)
	assert.Empty(t, result.Tests())
	assert.Zero(t, result.Dropped)
}
