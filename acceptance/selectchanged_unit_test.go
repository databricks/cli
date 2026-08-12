package acceptance_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var changedTestDirs = map[string]bool{
	"bundle/added":           true,
	"bundle/modified":        true,
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
		"M\tacceptance/bundle/modified/script",
		"A\tacceptance/bundle/added/script",
	)
	// The cap keeps added first, then modified, then moved.
	for limit, expected := range map[int][]string{
		1: {"bundle/added"},
		2: {"bundle/added", "bundle/modified"},
		3: {"bundle/added", "bundle/modified", "bundle/moved"},
	} {
		changed, dropped := classifyChangedTests(diff, changedTestDirs, limit)
		assert.Len(t, changed, limit)
		assert.Equal(t, 3-limit, dropped, "limit=%d", limit)
		for _, dir := range expected {
			assert.Contains(t, changed, dir, "limit=%d", limit)
		}
	}
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
