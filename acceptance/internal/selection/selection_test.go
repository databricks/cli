package selection_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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

// fromDiff selects against a root with no materialized configs, so no test dir contradicts a
// variant filter. TestFromDiffSkipsConfigNotInMatrix covers the case where one does.
func fromDiff(t *testing.T, diff string, limit int) selection.Result {
	return selection.FromDiff(t.TempDir(), diff, testDirs, limit)
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
	result := fromDiff(t, diff, 10)
	assert.Zero(t, result.Dropped)
	assert.Equal(t, map[string][]string{
		"bundle/added":    nil,
		"bundle/modified": nil,
		"bundle/moved":    nil,
	}, result.Tests())
}

func TestFromDiffPriority(t *testing.T) {
	diff := diffLines(
		"R100\tacceptance/bundle/old/script\tacceptance/bundle/moved/script",
		"M\tacceptance/bundle/regenerated/output.txt",
		"M\tacceptance/bundle/modified/script",
		"A\tacceptance/bundle/added/script",
	)
	// The cap keeps added first, then a changed fixture, then a moved dir, and last a dir
	// where only the golden output was regenerated.
	scored := []string{"bundle/added", "bundle/modified", "bundle/moved", "bundle/regenerated"}
	for limit := 1; limit <= len(scored); limit++ {
		result := fromDiff(t, diff, limit)
		assert.Len(t, result.Selected, limit)
		assert.Equal(t, len(scored)-limit, result.Dropped, "limit=%d", limit)
		for _, dir := range scored[:limit] {
			assert.Contains(t, result.Tests(), dir, "limit=%d", limit)
		}
	}
}

func TestFromDiffScores(t *testing.T) {
	// Scores add up: a new dir counts as new (5) plus its fixtures (5) plus its goldens
	// (1); a dir whose script and golden both changed counts 5+1. A dir that only moved
	// scores the move alone, since the identical renames it brings along are not changes.
	diff := diffLines(
		"A\tacceptance/bundle/added/script",
		"A\tacceptance/bundle/added/output.txt",
		"M\tacceptance/bundle/modified/script",
		"M\tacceptance/bundle/modified/output.txt",
		"R100\tacceptance/bundle/old/script\tacceptance/bundle/moved/script",
		"R100\tacceptance/bundle/old/output.txt\tacceptance/bundle/moved/output.txt",
		"M\tacceptance/bundle/regenerated/output.txt",
		"M\tacceptance/bundle/untouched/databricks.yml",
	)
	result := fromDiff(t, diff, 10)
	scores := map[string]int{}
	for _, test := range result.Selected {
		scores[test.Dir] = test.Score
	}
	assert.Equal(t, map[string]int{
		"bundle/added":       11,
		"bundle/modified":    6,
		"bundle/untouched":   5,
		"bundle/moved":       1,
		"bundle/regenerated": 1,
	}, scores)
}

func TestFromDiffFixtureBeatsOutputInSameDir(t *testing.T) {
	// A dir with both a fixture and an output change ranks as a fixture change.
	diff := diffLines(
		"M\tacceptance/bundle/modified/output.txt",
		"M\tacceptance/bundle/modified/databricks.yml",
		"M\tacceptance/bundle/regenerated/out.requests.txt",
	)
	result := fromDiff(t, diff, 1)
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
	result := fromDiff(t, diff, 1)
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
	result := fromDiff(t, diff, 2)
	assert.NotContains(t, result.Tests(), "bundle/regenerated")
	assert.Equal(t, 1, result.Dropped)
}

func TestFromDiffNewInvariantConfig(t *testing.T) {
	// A new invariant config adds a variant of every invariant dir, so it scores as a new
	// test on top of the fixture change. Changing an existing config, or adding one of its
	// -init.sh companions, only changes how an existing variant runs.
	for status, want := range map[string]int{"A": 10, "M": 5} {
		result := fromDiff(t, diffLines(status+"\tacceptance/bundle/invariant/configs/job.yml.tmpl"), 10)
		assert.Equal(t, want, result.Selected[0].Score, "status=%s", status)
	}

	result := fromDiff(t, diffLines("A\tacceptance/bundle/invariant/configs/job.yml.tmpl-init.sh"), 10)
	assert.Equal(t, 5, result.Selected[0].Score)
}

func TestFromDiffTwoInvariantConfigs(t *testing.T) {
	// Two changed configs restrict the invariant dirs to the variants of both, as filters
	// naming the same key are alternatives.
	diff := diffLines(
		"M\tacceptance/bundle/invariant/configs/job.yml.tmpl",
		"M\tacceptance/bundle/invariant/configs/pipeline.yml.tmpl",
	)
	result := fromDiff(t, diff, 10)
	both := []string{"INPUT_CONFIG=job.yml.tmpl", "INPUT_CONFIG=pipeline.yml.tmpl"}
	assert.Equal(t, map[string][]string{
		"bundle/invariant/jobs": both,
		"bundle/invariant/apps": both,
	}, result.Tests())
}

func TestFromDiffConfigAndItsInitScript(t *testing.T) {
	// A config and its setup script name the same variant, so the filter is kept.
	diff := diffLines(
		"M\tacceptance/bundle/invariant/configs/job.yml.tmpl",
		"M\tacceptance/bundle/invariant/configs/job.yml.tmpl-init.sh",
	)
	result := fromDiff(t, diff, 10)
	assert.Equal(t, map[string][]string{
		"bundle/invariant/jobs": {"INPUT_CONFIG=job.yml.tmpl"},
		"bundle/invariant/apps": {"INPUT_CONFIG=job.yml.tmpl"},
	}, result.Tests())
}

func TestFromDiffMovedDirWithNewConfig(t *testing.T) {
	// The move is a change to the dir itself, so it runs every variant, including the one
	// the new config adds, and scores the move.
	diff := diffLines(
		"A\tacceptance/bundle/invariant/configs/job.yml.tmpl",
		"R100\tacceptance/bundle/invariant/old/script\tacceptance/bundle/invariant/jobs/script",
	)
	result := fromDiff(t, diff, 10)
	scores := map[string]int{}
	for _, test := range result.Selected {
		scores[test.Name()] = test.Score
	}
	assert.Equal(t, 1, scores["bundle/invariant/jobs"])
	assert.Nil(t, result.Tests()["bundle/invariant/jobs"])
}

func TestFromDiffNewConfigScoresOnlyItsVariant(t *testing.T) {
	// Each variant is scored on its own: adding one config makes that config's variant new
	// without making the variant of a config that was merely changed look new too.
	diff := diffLines(
		"M\tacceptance/bundle/invariant/configs/job.yml.tmpl",
		"A\tacceptance/bundle/invariant/configs/pipeline.yml.tmpl",
	)
	result := fromDiff(t, diff, 10)
	scores := map[string]int{}
	for _, test := range result.Selected {
		scores[test.Name()] = test.Score
	}
	assert.Equal(t, map[string]int{
		"bundle/invariant/apps/INPUT_CONFIG=job.yml.tmpl":      5,
		"bundle/invariant/apps/INPUT_CONFIG=pipeline.yml.tmpl": 10,
		"bundle/invariant/jobs/INPUT_CONFIG=job.yml.tmpl":      5,
		"bundle/invariant/jobs/INPUT_CONFIG=pipeline.yml.tmpl": 10,
	}, scores)
}

func TestFromDiffNestedDir(t *testing.T) {
	// A file maps to the innermost test dir that owns it.
	diff := diffLines("M\tacceptance/cmd/sync/nested/deeper/script")
	result := fromDiff(t, diff, 10)
	assert.Equal(t, map[string][]string{"cmd/sync/nested/deeper": nil}, result.Tests())
}

func TestFromDiffInvariantConfig(t *testing.T) {
	// A changed invariant config re-enables every invariant dir, restricted to that config.
	diff := diffLines("M\tacceptance/bundle/invariant/configs/job.yml.tmpl")
	result := fromDiff(t, diff, 10)
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
	result := fromDiff(t, diff, 10)
	assert.Equal(t, map[string][]string{
		"bundle/invariant/jobs": nil,
		"bundle/invariant/apps": {"INPUT_CONFIG=job.yml.tmpl"},
	}, result.Tests())
}

func TestFromDiffEmptyDiff(t *testing.T) {
	result := fromDiff(t, "", 10)
	assert.Empty(t, result.Tests())
	assert.Zero(t, result.Dropped)
}

func TestFromDiffInvariantTestToml(t *testing.T) {
	// test.toml is hand-written, so changing it selects the dir it belongs to. out.test.toml
	// next to it is generated whenever the matrix changes, and selects nothing.
	result := fromDiff(t, diffLines("M\tacceptance/bundle/invariant/jobs/test.toml"), 10)
	assert.Equal(t, map[string][]string{"bundle/invariant/jobs": nil}, result.Tests())

	result = fromDiff(t, diffLines("M\tacceptance/bundle/invariant/jobs/out.test.toml"), 10)
	assert.Empty(t, result.Tests())
}

func TestFromDiffSkipsConfigNotInMatrix(t *testing.T) {
	// An invariant dir that does not run the changed config would be selected with every
	// variant skipped, spending the limit on a test that runs nothing.
	root := t.TempDir()
	for dir, configs := range map[string]string{
		"bundle/invariant/jobs": `["job.yml.tmpl", "app.yml.tmpl"]`,
		"bundle/invariant/apps": `["app.yml.tmpl"]`,
	} {
		require.NoError(t, os.MkdirAll(filepath.Join(root, dir), 0o755))
		contents := "EnvMatrix.INPUT_CONFIG = " + configs + "\n"
		require.NoError(t, os.WriteFile(filepath.Join(root, dir, "out.test.toml"), []byte(contents), 0o644))
	}

	diff := diffLines("M\tacceptance/bundle/invariant/configs/job.yml.tmpl")
	result := selection.FromDiff(root, diff, testDirs, 10)
	assert.Equal(t, map[string][]string{
		"bundle/invariant/jobs": {"INPUT_CONFIG=job.yml.tmpl"},
	}, result.Tests())
}

func TestFromDiffRenameOutOfTestDir(t *testing.T) {
	// The dir that lost the file may no longer pass, so it runs in full. The whole dir moving
	// away is different: its old dir is gone, so only the new one is selected.
	diff := diffLines("R100\tacceptance/bundle/modified/databricks.yml\tdocs/example.yml")
	result := fromDiff(t, diff, 10)
	assert.Equal(t, map[string][]string{"bundle/modified": nil}, result.Tests())

	diff = diffLines("R100\tacceptance/bundle/old/script\tacceptance/bundle/moved/script")
	result = fromDiff(t, diff, 10)
	assert.Equal(t, map[string][]string{"bundle/moved": nil}, result.Tests())
}

func TestFromDiffMovedScoresMoveAndChange(t *testing.T) {
	// Only an identical rename is a pure move. A rename that rewrote the script, or a move
	// that came with another change, is scored for the change as well.
	scoreOf := func(diff, name string) int {
		for _, test := range fromDiff(t, diff, 10).Selected {
			if test.Name() == name {
				return test.Score
			}
		}
		return 0
	}

	moved := "acceptance/bundle/old/script\tacceptance/bundle/moved/script"
	assert.Equal(t, 1, scoreOf(diffLines("R100\t"+moved), "bundle/moved"))
	assert.Equal(t, 5, scoreOf(diffLines("R090\t"+moved), "bundle/moved"))
	assert.Equal(t, 6, scoreOf(diffLines(
		"R100\t"+moved,
		"M\tacceptance/bundle/moved/databricks.yml",
	), "bundle/moved"))
}

func TestFromDiffTiesAreAlphabetical(t *testing.T) {
	// Tests that score the same are ordered by name, so the limit cuts the same set whatever
	// order the diff lists the changed files in.
	lines := []string{
		"M\tacceptance/bundle/invariant/configs/pipeline.yml.tmpl",
		"M\tacceptance/bundle/invariant/configs/job.yml.tmpl",
	}
	expected := []string{
		"bundle/invariant/apps/INPUT_CONFIG=job.yml.tmpl",
		"bundle/invariant/apps/INPUT_CONFIG=pipeline.yml.tmpl",
	}

	for _, diff := range []string{diffLines(lines...), diffLines(lines[1], lines[0])} {
		result := fromDiff(t, diff, 2)
		var names []string
		for _, test := range result.Selected {
			names = append(names, test.Name())
		}
		assert.Equal(t, expected, names)
	}
}
