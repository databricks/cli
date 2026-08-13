package acceptance_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var changedTestDirs = map[string]bool{
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

func TestClassifyChangedTestsStatuses(t *testing.T) {
	diff := diffLines(
		"A\tacceptance/bundle/added/script",
		"M\tacceptance/bundle/modified/databricks.yml",
		"R100\tacceptance/bundle/old/script\tacceptance/bundle/moved/script",
		"M\tlibs/dyn/value.go",
	)
	changed, dropped := classifyChangedTests(diff, changedTestDirs, 10)
	assert.Zero(t, dropped)
	assert.Equal(t, map[string][]string{
		"bundle/added":    nil,
		"bundle/modified": nil,
		"bundle/moved":    nil,
	}, changed)
}

func TestClassifyChangedTestsPriority(t *testing.T) {
	diff := diffLines(
		"R090\tacceptance/bundle/old/script\tacceptance/bundle/moved/script",
		"M\tacceptance/bundle/regenerated/output.txt",
		"M\tacceptance/bundle/modified/script",
		"A\tacceptance/bundle/added/script",
	)
	// The cap keeps added first, then a changed fixture, then a regenerated output,
	// then a moved dir.
	ranked := []string{"bundle/added", "bundle/modified", "bundle/regenerated", "bundle/moved"}
	for limit := 1; limit <= len(ranked); limit++ {
		changed, dropped := classifyChangedTests(diff, changedTestDirs, limit)
		assert.Len(t, changed, limit)
		assert.Equal(t, len(ranked)-limit, dropped, "limit=%d", limit)
		for _, dir := range ranked[:limit] {
			assert.Contains(t, changed, dir, "limit=%d", limit)
		}
	}
}

func TestClassifyChangedTestsFixtureBeatsOutputInSameDir(t *testing.T) {
	// A dir with both a fixture and an output change ranks as a fixture change.
	diff := diffLines(
		"M\tacceptance/bundle/modified/output.txt",
		"M\tacceptance/bundle/modified/databricks.yml",
		"M\tacceptance/bundle/regenerated/out.requests.txt",
	)
	changed, dropped := classifyChangedTests(diff, changedTestDirs, 1)
	assert.Equal(t, map[string][]string{"bundle/modified": nil}, changed)
	assert.Equal(t, 1, dropped)
}

func TestClassifyChangedTestsNestedFixture(t *testing.T) {
	// "out" is matched against the path relative to the test dir, so a file in a
	// subdirectory is a fixture even when its own name starts with "out".
	diff := diffLines(
		"M\tacceptance/bundle/modified/subdir/outer.py",
		"M\tacceptance/bundle/regenerated/output.txt",
	)
	changed, dropped := classifyChangedTests(diff, changedTestDirs, 1)
	assert.Equal(t, map[string][]string{"bundle/modified": nil}, changed)
	assert.Equal(t, 1, dropped)
}

func TestClassifyChangedTestsInvariantConfigRanksAsFixture(t *testing.T) {
	// The invariant config is the fixture its dirs are generated from, so it outranks
	// a dir whose output was regenerated.
	diff := diffLines(
		"M\tacceptance/bundle/regenerated/output.txt",
		"M\tacceptance/bundle/invariant/configs/job.yml.tmpl",
	)
	changed, dropped := classifyChangedTests(diff, changedTestDirs, 2)
	assert.NotContains(t, changed, "bundle/regenerated")
	assert.Equal(t, 1, dropped)
}

func TestClassifyChangedTestsNestedDir(t *testing.T) {
	// A file maps to the innermost test dir that owns it.
	diff := diffLines("M\tacceptance/cmd/sync/nested/deeper/script")
	changed, _ := classifyChangedTests(diff, changedTestDirs, 10)
	assert.Equal(t, map[string][]string{"cmd/sync/nested/deeper": nil}, changed)
}

func TestClassifyChangedTestsInvariantConfig(t *testing.T) {
	// A changed invariant config re-enables every invariant dir, restricted to that config.
	diff := diffLines("M\tacceptance/bundle/invariant/configs/job.yml.tmpl")
	changed, _ := classifyChangedTests(diff, changedTestDirs, 10)
	assert.Equal(t, map[string][]string{
		"bundle/invariant/jobs": {"INPUT_CONFIG=job.yml.tmpl"},
		"bundle/invariant/apps": {"INPUT_CONFIG=job.yml.tmpl"},
	}, changed)
}

func TestClassifyChangedTestsInvariantConfigAndDir(t *testing.T) {
	// A non-config change to an invariant dir unlocks all of its variants; the
	// regenerated test.toml files are ignored.
	diff := diffLines(
		"M\tacceptance/bundle/invariant/configs/job.yml.tmpl",
		"M\tacceptance/bundle/invariant/jobs/script",
		"M\tacceptance/bundle/invariant/apps/out.test.toml",
	)
	changed, _ := classifyChangedTests(diff, changedTestDirs, 10)
	assert.Equal(t, map[string][]string{
		"bundle/invariant/jobs": nil,
		"bundle/invariant/apps": {"INPUT_CONFIG=job.yml.tmpl"},
	}, changed)
}

func TestClassifyChangedTestsEmptyDiff(t *testing.T) {
	changed, dropped := classifyChangedTests("", changedTestDirs, 10)
	assert.Empty(t, changed)
	assert.Zero(t, dropped)
}
